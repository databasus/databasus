package start

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/config"
	"databasus-backend/internal/agent/container"
	"databasus-backend/internal/agent/heartbeat"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/runner"
	"databasus-backend/internal/agent/upgrade"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/lib/background"
)

func Start(ctx context.Context, cfg *config.Config, log *slog.Logger) (int, error) {
	if _, err := cfg.Validate(); err != nil {
		return 0, err
	}
	return spawnDaemon(ctx, log)
}

func RunDaemon(ctx context.Context, cfg *config.Config, agentVersion string, isDev bool, log *slog.Logger) error {
	capacity, err := cfg.Validate()
	if err != nil {
		return err
	}

	if err := cfg.ValidateTransport(ctx, log, isStdinTTY(), bufio.NewReader(os.Stdin)); err != nil {
		return err
	}

	lockFile, err := AcquireLock(ctx, log)
	if err != nil {
		return err
	}
	defer ReleaseLock(lockFile)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	group := background.New(background.Deps{Logger: log})
	defer func() { cancel(); _ = group.Wait(context.WithoutCancel(ctx)) }()

	watcher, err := NewLockWatcher(lockFile, cancel, log)
	if err != nil {
		return fmt.Errorf("failed to initialize lock watcher: %w", err)
	}
	group.Go(ctx, "agent_lock_watcher", func(ctx context.Context) error { watcher.Run(ctx); return nil })

	apiClient := api.NewClient(api.Deps{Host: cfg.DatabasusHost, Token: cfg.Token, AgentID: cfg.AgentID, Logger: log})

	engine, err := container.NewDockerEngine(log)
	if err != nil {
		return err
	}

	containerManager := container.NewManager(
		container.Deps{
			Engine:          engine,
			AgentID:         cfg.AgentID,
			EngineImageRepo: cfg.GetVerificationPgImageRepo(),
			Logger:          log,
		},
	)
	if err := containerManager.StartupSelfCheck(ctx); err != nil {
		return err
	}

	// Start fresh: remove any container leaked by a prior ungraceful death
	// before claiming the first job. Safe to wipe unconditionally because the
	// lock guarantees a single agent owns no live container the instant it
	// starts.
	containerManager.PurgeContainers(ctx)

	pool := runner.NewPool(capacity.MaxConcurrentJobs)
	heartbeater := heartbeat.NewHeartbeater(apiClient, capacity, log)
	verificationRunner := runner.NewRunner(
		runner.Deps{
			APIClient:      apiClient,
			Capacity:       capacity,
			Pool:           pool,
			Spawner:        containerManagerSpawner{containerManager: containerManager},
			Restorer:       restore.NewRestorer(log),
			StatsCollector: verifier.NewVerifier(log),
			Heartbeat:      heartbeater,
			Logger:         log,
			Group:          group,
		},
	)

	var backgroundUpgrader *upgrade.BackgroundUpgrader
	if agentVersion != "dev" && !isDev {
		backgroundUpgrader = upgrade.NewBackgroundUpgrader(
			upgrade.BackgroundDeps{
				APIClient:      apiClient,
				CurrentVersion: agentVersion,
				IsDev:          isDev,
				Cancel:         cancel,
				Logger:         log,
			},
		)
		group.Go(ctx, "agent_upgrader", func(ctx context.Context) error { backgroundUpgrader.Run(ctx); return nil })
	}

	log.InfoContext(ctx, "agent started")

	group.Go(ctx, "agent_heartbeater", func(ctx context.Context) error { heartbeater.Run(ctx); return nil })
	var runnerError error
	group.Go(ctx, "agent_runner", func(ctx context.Context) error {
		runnerError = background.RunRecovering(ctx, verificationRunner.Run)
		if runnerError != nil {
			cancel()
		}

		return runnerError
	})

	<-ctx.Done()

	// verificationRunner.Run drains in-flight jobs (pool.Wait) before it
	// returns, so the group wait only completes once every container is torn down —
	// strictly before any re-exec on self-update.
	_ = group.Wait(context.WithoutCancel(ctx))

	if runnerError != nil {
		return runnerError
	}

	if backgroundUpgrader != nil {
		backgroundUpgrader.WaitForCompletion(30 * time.Second)

		if backgroundUpgrader.IsUpgraded() {
			// The deferred ReleaseLock runs before this returns to the caller,
			// freeing the flock and removing the lock file. The caller then
			// syscall.Exec's the same PID with os.Args unchanged, so the new
			// image re-enters via the same subcommand (`run` or `_run`) and
			// re-acquires the lock cleanly with no contender.
			return upgrade.ErrUpgradeRestart
		}
	}

	log.InfoContext(ctx, "agent stopped")

	return nil
}

func isStdinTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
