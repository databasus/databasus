package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/agent/dbconn"
)

const (
	containerReadyTimeout = 60 * time.Second
	readyPollInterval     = 1 * time.Second

	// stockEngineImageRepo is the official Postgres image; the job's major version
	// (from the backend assignment) is appended as the tag. It is the fallback
	// when the configured (extension-bundling) repo can't be pulled.
	stockEngineImageRepo = "postgres"

	// timescaleImageRepo carries the timescaledb extension; the tag is
	// "<extversion>-pg<major>" so the engine matches the dump exactly
	// (pg_restore cannot cross extension versions).
	timescaleImageRepo = "timescale/timescaledb"
)

// containerUser is the unprivileged user every Postgres image the agent runs
// ships. Starting as this user makes the official entrypoint skip its switch
// from root, so no step in the container needs a capability and all of them
// are dropped. It replaces the kernel's no_new_privs flag: on rootful Podman
// with crun and AppArmor that flag stacks the container's AppArmor profile on
// crun's, which denies every signal between the container's processes and
// hangs Postgres.
// With an empty bounding set, a setuid binary still yields no capability.
// The agent's execs (dd, pg_restore, du) run as this user too.
const containerUser = "postgres"

// The restore container is destroyed
// after verification, so durability is irrelevant: fsync=off and
// full_page_writes=off cut WAL volume, faster checkpoints recycle pg_wal
// sooner. max_wal_senders=0 is mandatory under wal_level=minimal — Postgres
// refuses to start otherwise. The first arg stays "postgres" so the official
// image entrypoint still runs initdb.
func getRestoreTunedPostgresCmd() []string {
	return []string{
		"postgres",
		"-c", "fsync=off",
		"-c", "full_page_writes=off",
		"-c", "synchronous_commit=off",
		"-c", "wal_level=minimal",
		"-c", "max_wal_senders=0",
		"-c", "max_wal_size=1GB",
	}
}

type Manager struct {
	engine              *DockerEngine
	agentID             string
	engineImageRepo     string
	isCPULimitSupported bool
	log                 *slog.Logger
}

type Deps struct {
	Engine          *DockerEngine
	AgentID         string
	EngineImageRepo string
	Logger          *slog.Logger
}

func NewManager(deps Deps) *Manager {
	return &Manager{
		engine:              deps.Engine,
		agentID:             deps.AgentID,
		engineImageRepo:     deps.EngineImageRepo,
		isCPULimitSupported: true,
		log:                 deps.Logger,
	}
}

func (m *Manager) StartupSelfCheck(ctx context.Context) error {
	if err := m.engine.Ping(ctx); err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}

	m.log.InfoContext(ctx, "docker daemon is reachable")

	capabilities, err := m.engine.GetHostCapabilities(ctx)
	if err != nil {
		// The CPU limit stays on: dropping it on a transient failure would
		// silently unlimit hosts that support it.
		m.log.WarnContext(ctx, "could not determine docker host capabilities", slog.Any("error", err))
		return nil
	}

	if !capabilities.IsUserNSRemapEnabled {
		m.log.WarnContext(ctx, "docker user-namespace remapping is OFF — the strongest control "+
			"against a container escape is absent; enable userns-remap on the "+
			"verification host")
	}

	m.isCPULimitSupported = capabilities.IsCPULimitSupported
	if !m.isCPULimitSupported {
		m.log.WarnContext(ctx, "docker host does not support CPU CFS quota — verification containers "+
			"run without a CPU limit, so a restore can use every CPU core on the host")
	}

	return nil
}

// PurgeContainers removes every container+network this agent owns so each
// process starts from a clean slate. Call once at startup, after
// StartupSelfCheck (Docker must be reachable) and before any job is claimed.
// See purgeAgentContainers for why blanket removal is safe under one agent.
func (m *Manager) PurgeContainers(ctx context.Context) {
	purgeAgentContainers(ctx, m.engine, m.agentID, m.log)
}

// Spawn's every failure is pre-pg_restore — the runner reports it with no exit
// code (retryable AgentSetupFailed), never BackupRejected.
func (m *Manager) Spawn(jobCtx context.Context, req SpawnRequest) (*PostgresContainer, error) {
	image := imageForJob(req, m.engineImageRepo)

	if err := m.engine.EnsureImage(jobCtx, image); err != nil {
		fallbackImage, canFallBack := stockFallbackImage(req, image)
		if !canFallBack {
			return nil, fmt.Errorf("ensure image: %w", err)
		}

		// The extension-bundling image is unreachable (registry down, air-gapped,
		// not yet published). Fall back to the stock Postgres image; missing
		// extensions then surface as ignored pg_restore errors the runner tolerates.
		m.log.WarnContext(
			jobCtx,
			"bundled verification image unavailable; falling back to stock postgres image",
			slog.Any("bundled_image", image),
			slog.Any("stock_image", fallbackImage),
			slog.Any("error", err),
		)

		image = fallbackImage

		if err := m.engine.EnsureImage(jobCtx, image); err != nil {
			return nil, fmt.Errorf("ensure stock image: %w", err)
		}
	}

	password, err := randomPassword()
	if err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}

	// Both labels are load-bearing for leftover removal: the startup purge
	// filters on LabelAgentID, and the next attempt of this verification on
	// both. A resource created without them leaks until the host is cleaned by
	// hand, and a leftover network blocks every retry by its name.
	labels := attemptLabels(m.agentID, req.VerificationID.String())

	networkID, err := prepareAttemptNetwork(
		jobCtx, m.engine, "databasus-verif-"+req.VerificationID.String(), labels)
	if err != nil {
		return nil, err
	}

	spec := m.buildSpec(spawnPlan{
		verificationID: req.VerificationID,
		image:          image,
		password:       password,
		cpuPerJob:      req.CPUPerJob,
		ramMbPerJob:    req.RAMMbPerJob,
		networkID:      networkID,
		labels:         labels,
	})

	containerID, err := m.engine.CreateContainer(jobCtx, spec)
	if err != nil {
		_ = m.engine.RemoveNetwork(jobCtx, networkID)
		return nil, fmt.Errorf("create container: %w", err)
	}

	c := &PostgresContainer{engine: m.engine, id: containerID, networkID: networkID, password: password, log: m.log}

	if err := m.engine.StartContainer(jobCtx, containerID); err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("start container: %w", err)
	}

	hostPort, err := m.engine.HostPort(jobCtx, containerID, strconv.Itoa(pgInternalPort)+"/tcp")
	if err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("resolve published port: %w", err)
	}

	c.hostPort = hostPort

	if err := waitForReady(jobCtx, c.GetVerifierConn()); err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("container never became ready: %w", err)
	}

	return c, nil
}

func (m *Manager) buildSpec(plan spawnPlan) SpawnSpec {
	return SpawnSpec{
		Name:        "databasus-verif-" + plan.verificationID.String(),
		Image:       plan.image,
		Cmd:         getRestoreTunedPostgresCmd(),
		Env:         []string{"POSTGRES_PASSWORD=" + plan.password},
		Labels:      plan.labels,
		NanoCPUs:    m.getNanoCPUs(plan.cpuPerJob),
		MemoryBytes: int64(plan.ramMbPerJob) * 1024 * 1024,
		PidsLimit:   containerPidsLimit,
		NetworkID:   plan.networkID,

		CapDropAll: true,
		User:       containerUser,
		// rootfs is writable because the official image needs a writable PGDATA;
		// the per-job disk budget is enforced by the agent's disk watcher, not a
		// Docker cap. The dominant controls (unprivileged user, no capabilities,
		// pids, memory, userns, ephemeral lifecycle) remain in force. The job network
		// is a per-job user-defined bridge (job isolation) but NOT --internal:
		// see CreateNetwork in dockerengine.go for why.
		ReadonlyRootfs: false,
	}
}

// getNanoCPUs returns 0 on hosts without CFS quota support, which Docker
// treats as no CPU limit; any other value makes the daemon reject the container.
func (m *Manager) getNanoCPUs(cpuPerJob int) int64 {
	if !m.isCPULimitSupported {
		return 0
	}

	return int64(cpuPerJob) * 1_000_000_000
}

func waitForReady(ctx context.Context, conn dbconn.Conn) error {
	readyCtx, cancel := context.WithTimeout(ctx, containerReadyTimeout)
	defer cancel()

	ticker := time.NewTicker(readyPollInterval)
	defer ticker.Stop()

	// The deadline alone hides why Postgres stayed unreachable: a refused port,
	// a connection nobody answers and a rejected login need different fixes,
	// and only the last ping error tells them apart.
	var lastPingErr error

	readyDeadline, _ := readyCtx.Deadline()

	for {
		// A ping that starts with less than its full interval left ends at the overall
		// deadline, and its timer can fire a moment before the deadline's own does.
		pingHasFullInterval := time.Until(readyDeadline) >= readyPollInterval

		err := pingOnce(readyCtx, conn)
		if err == nil {
			return nil
		}

		// A ping the overall deadline cut short says nothing about Postgres, so
		// it never replaces the error of a ping that finished on its own.
		if (pingHasFullInterval && readyCtx.Err() == nil) || lastPingErr == nil {
			lastPingErr = err
		}

		select {
		case <-readyCtx.Done():
			return fmt.Errorf("%w; last ping error: %w", readyCtx.Err(), lastPingErr)
		case <-ticker.C:
		}
	}
}

func pingOnce(ctx context.Context, conn dbconn.Conn) error {
	pingCtx, cancel := context.WithTimeout(ctx, readyPollInterval)
	defer cancel()

	pgConn, err := pgx.Connect(pingCtx, conn.GetDSN())
	if err != nil {
		return err
	}
	defer func() { _ = pgConn.Close(pingCtx) }()

	return pgConn.Ping(pingCtx)
}

func imageForJob(req SpawnRequest, engineImageRepo string) string {
	if req.TimescaledbVersion != "" {
		return fmt.Sprintf("%s:%s-pg%s", timescaleImageRepo, req.TimescaledbVersion, req.PgMajor)
	}

	return imageForMajor(req.PgMajor, engineImageRepo)
}

func imageForMajor(pgMajor, engineImageRepo string) string {
	return engineImageRepo + ":" + pgMajor
}

// stockFallbackImage decides whether a failed pull of the configured image may
// retry against the official Postgres image. Timescale jobs need their exact
// version-matched image (pg_restore cannot cross extension versions), so they
// never fall back; nor does a job already targeting the stock repo.
func stockFallbackImage(req SpawnRequest, attemptedImage string) (string, bool) {
	if req.TimescaledbVersion != "" {
		return "", false
	}

	stockImage := imageForMajor(req.PgMajor, stockEngineImageRepo)
	if stockImage == attemptedImage {
		return "", false
	}

	return stockImage, true
}

func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
