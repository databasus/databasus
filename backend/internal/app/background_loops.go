package app

import (
	"context"
	"fmt"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_bridge "databasus-backend/internal/features/backups/config"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	"databasus-backend/internal/features/telemetry"
	verification_runs "databasus-backend/internal/features/verification/runs"
	"databasus-backend/internal/lib/background"
)

type backgroundLoop struct {
	name string
	run  func(ctx context.Context)
}

func (a *App) startBackgroundLoops(ctx context.Context) {
	loops := []backgroundLoop{
		{"verification scheduler", verification_runs.GetVerificationScheduler().Run},
		{"physical backup scheduler background service", backuping_physical.GetPhysicalBackupsScheduler().Run},
		{"physical backup cleaner background service", backuping_physical.GetPhysicalBackupCleaner().Run},
		{"healthcheck attempt background service", healthcheck_attempt.GetHealthcheckAttemptBackgroundService().Run},
		{"physical wal stream supervisor background service", backuping_physical.GetPhysicalWalStreamSupervisor().Run},
		{"telemetry background service", telemetry.GetTelemetryBackgroundService().Run},
	}

	a.logger.InfoContext(ctx, "starting background tasks")

	for _, loop := range loops {
		a.backgroundGroup.Go(ctx, loop.name, func(loopCtx context.Context) error {
			loop.run(loopCtx)

			return nil
		})
	}
}

func (a *App) StartPhysicalBackupLoops(ctx context.Context) func(context.Context) error {
	group := background.New(background.Deps{Logger: a.logger})
	loops := []backgroundLoop{
		{"physical backup scheduler", backuping_physical.NewPhysicalBackupsScheduler().Run},
		{"physical wal stream supervisor", backuping_physical.NewPhysicalWalStreamSupervisor().Run},
	}
	for _, loop := range loops {
		group.Go(ctx, loop.name, func(loopCtx context.Context) error {
			loop.run(loopCtx)

			return nil
		})
	}

	return func(stopCtx context.Context) error {
		if err := group.Wait(stopCtx); err != nil {
			return fmt.Errorf("stop physical backup loops: %w", err)
		}

		return backups_bridge.GetCancellationRegistry().WaitUntilEmpty(stopCtx)
	}
}

func (a *App) RunPhysicalBackupCleanerPass(ctx context.Context) error {
	return backuping_physical.NewPhysicalBackupCleaner().RunPass(ctx)
}
