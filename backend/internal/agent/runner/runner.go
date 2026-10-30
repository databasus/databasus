package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/config"
	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/background"
)

const (
	jobName = "verification_runner"

	claimErrorBackoff         = 5 * time.Second
	idleClaimBackoff          = 15 * time.Second
	maxBackupDownloadAttempts = 5
	streamIdleTimeout         = 60 * time.Second
	maxBackupDownloadBackoff  = 32 * time.Second

	// archive layout inside the per-job container's writable volume.
	inContainerArchiveDir = "/tmp"
)

func getSupportedMajors() []string {
	return []string{"12", "13", "14", "15", "16", "17", "18"}
}

type SpawnRequest struct {
	PgMajor            string
	CPUPerJob          int
	RAMMbPerJob        int
	VerificationID     uuid.UUID
	TimescaledbVersion string
}

type Runner struct {
	group                   *background.Group
	diskWatchInterval       time.Duration
	backupDownloadBackoffFn func(int) time.Duration
	api                     APIClient
	capacity                config.Capacity
	pool                    *Pool
	spawner                 Spawner
	restorer                Restorer
	verifier                StatsCollector
	heartbeat               Registrar
	connAlive               func(ctx context.Context, conn dbconn.Conn) bool
	hasRun                  atomic.Bool
	log                     *slog.Logger
}

type Deps struct {
	APIClient      APIClient
	Capacity       config.Capacity
	Pool           *Pool
	Spawner        Spawner
	Restorer       Restorer
	StatsCollector StatsCollector
	Heartbeat      Registrar
	Logger         *slog.Logger
	Group          *background.Group
}

func NewRunner(deps Deps) *Runner {
	return &Runner{
		group:                   deps.Group,
		diskWatchInterval:       3 * time.Second,
		backupDownloadBackoffFn: backupDownloadBackoff,
		api:                     deps.APIClient,
		capacity:                deps.Capacity,
		pool:                    deps.Pool,
		spawner:                 deps.Spawner,
		restorer:                deps.Restorer,
		verifier:                deps.StatsCollector,
		heartbeat:               deps.Heartbeat,
		connAlive:               verifier.ProbeConnAlive,
		log:                     deps.Logger,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	if r.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", r))
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobErrors := make(chan error, 1)

	logger := r.log.With(slog.Any("job_id", uuid.New()), slog.Any("job_name", jobName))
	logger.InfoContext(ctx, "runner loop started")

	for ctx.Err() == nil {
		// Gate claiming on local capacity so we never hold a server-claimed
		// job we cannot start immediately.
		if r.pool.IsSaturated() {
			if !sleepOrDone(ctx, 1*time.Second) {
				break
			}

			continue
		}

		capacity := agentclient.AgentCapacity{
			MaxCPU:            r.capacity.MaxCPU,
			MaxRAMMb:          r.capacity.MaxRAMMb,
			MaxDiskGb:         r.capacity.MaxDiskGb,
			MaxConcurrentJobs: r.capacity.MaxConcurrentJobs,
		}

		assignment, err := r.api.ClaimVerification(ctx, capacity)
		if err != nil {
			logger.WarnContext(ctx, "claim failed", slog.Any("error", err))

			if !sleepOrDone(ctx, claimErrorBackoff) {
				break
			}

			continue
		}

		if assignment == nil {
			if !sleepOrDone(ctx, idleClaimBackoff) {
				break
			}

			continue
		}

		r.pool.Go(ctx, r.group, func(ctx context.Context) error {
			err := background.RunRecovering(ctx, func(ctx context.Context) error {
				r.executeJob(ctx, assignment)
				return nil
			})
			if err != nil {
				select {
				case jobErrors <- err:
				default:
				}
				cancel()
			}

			return err
		})
	}

	logger.InfoContext(ctx, "runner loop draining in-flight jobs")
	r.pool.Wait()
	logger.InfoContext(ctx, "runner loop stopped")

	select {
	case err := <-jobErrors:
		return err
	default:
		return nil
	}
}

func (r *Runner) executeJob(ctx context.Context, job *agentclient.VerificationAssignment) {
	runLogger := r.log.With(
		slog.Any("job_id", uuid.New()),
		slog.Any("job_name", jobName),
		slog.Any("verification_id", job.VerificationID),
	)

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register FIRST so the ID is in the heartbeat envelope before any
	// container exists: the backend must never see an artifact-bearing job
	// that is absent from this agent's reported in-flight set.
	r.heartbeat.TrackVerification(job.VerificationID, cancel)
	defer r.heartbeat.UntrackVerification(job.VerificationID)

	pgMajor, err := pgMajorFromDatabase(job.Database)
	if err != nil {
		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("resolve pg major: %v", err), runLogger)

		return
	}

	jobContainer, err := r.spawner.Spawn(jobCtx, SpawnRequest{
		PgMajor:            pgMajor,
		CPUPerJob:          r.capacity.CPUPerJob,
		RAMMbPerJob:        r.capacity.RAMMbPerJob,
		VerificationID:     job.VerificationID,
		TimescaledbVersion: job.TimescaledbVersion,
	})
	if err != nil {
		if jobCtx.Err() != nil {
			return
		}

		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("spawn container: %v", err), runLogger)

		return
	}

	defer func() {
		if termErr := jobContainer.Terminate(ctx); termErr != nil {
			runLogger.ErrorContext(ctx, "failed to terminate container", slog.Any("error", termErr))
		}
	}()

	// Server-authoritative per-job disk budget (downloaded archive + restored
	// DB + safe gap). The agent enforces it by watching the container's written
	// bytes and aborting the instant the budget is reached — uniform on every
	// storage driver, unlike a Docker --storage-opt cap.
	diskBudgetMb := int64(job.MaxContainerDiskMb)
	diskBudgetFailMessage := fmt.Sprintf("restore exceeded the per-job disk budget of %d MB", diskBudgetMb)

	var isDiskLimitHit atomic.Bool

	// Every stage running under jobCtx must consult this before treating a
	// cancelled context as a silent abort: the watcher cancels jobCtx itself, so
	// a bare `jobCtx.Err() != nil` early return would swallow the disk verdict
	// and leave the verification hanging until the backend reclaims it.
	reportIfDiskLimitHit := func() bool {
		if !isDiskLimitHit.Load() {
			return false
		}

		r.reportDiskLimitExceeded(ctx, job.VerificationID, diskBudgetFailMessage, runLogger)

		return true
	}

	// A zero/absent budget (older backend) means no enforceable ceiling — skip
	// the watcher rather than trip instantly on used >= 0.
	if diskBudgetMb > 0 {
		watchCtx, watchStop := context.WithCancel(jobCtx)
		defer watchStop()

		watcher := newDiskWatcher(diskWatchSpec{
			probe:           jobContainer,
			ceilingBytes:    diskBudgetMb * 1024 * 1024,
			interval:        r.diskWatchInterval,
			onLimitExceeded: func() { isDiskLimitHit.Store(true); cancel() },
			log:             runLogger,
		})
		r.group.Go(
			watchCtx,
			"verification_disk_watcher",
			func(ctx context.Context) error { watcher.run(ctx); return nil },
		)
	} else {
		runLogger.WarnContext(ctx, "no per-job disk budget from server; disk watcher disabled for this job")
	}

	archivePath := fmt.Sprintf("%s/%s.dump", inContainerArchiveDir, job.VerificationID)

	if err := r.downloadBackupIntoContainer(
		jobCtx,
		jobContainer,
		job.VerificationID,
		archivePath,
		runLogger,
	); err != nil {
		if reportIfDiskLimitHit() {
			return
		}

		if errors.Is(err, errAbortNoReport) {
			return
		}

		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("download backup: %v", err), runLogger)

		return
	}

	archiveOwnerRoleNames, err := r.restorer.EnsureArchiveOwnerRoles(
		jobCtx, jobContainer, archivePath, jobContainer.GetVerifierConn())
	if err != nil {
		if reportIfDiskLimitHit() {
			return
		}

		if jobCtx.Err() != nil {
			return
		}

		r.reportFailure(ctx, job.VerificationID, nil,
			fmt.Sprintf("ensure archive owner roles: %v", err), runLogger)

		return
	}

	if len(archiveOwnerRoleNames) > 0 {
		runLogger.InfoContext(
			ctx,
			"archive owner roles present in restore target",
			slog.Any("roles", strings.Join(archiveOwnerRoleNames, ",")),
		)
	}

	isTimescaleDB := job.TimescaledbVersion != ""
	parallelJobs := pgrestore.GetParallelJobs(r.capacity.CPUPerJob, isTimescaleDB)

	// On failure the container is discarded. The post step is needed only before collecting stats.
	if isTimescaleDB {
		if err := r.restorer.RunTimescalePreRestore(jobCtx, jobContainer.GetVerifierConn()); err != nil {
			if reportIfDiskLimitHit() {
				return
			}

			if jobCtx.Err() != nil {
				return
			}

			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("timescaledb_pre_restore: %v", err), runLogger)

			return
		}
	}

	// RunPgRestore returns Result by value and fills it when it fails with ErrRestoreFailed.
	// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
	restoreResult, err := r.restorer.RunPgRestore(jobCtx, jobContainer, restore.PgRestoreSpec{
		ArchivePath:   archivePath,
		Conn:          jobContainer.GetInContainerConn(),
		ParallelJobs:  parallelJobs,
		IsTimescaledb: isTimescaleDB,
	})
	if err != nil {
		if reportIfDiskLimitHit() {
			return
		}

		if jobCtx.Err() != nil {
			return
		}

		if !errors.Is(err, restore.ErrRestoreFailed) {
			// Exec-infrastructure failure (no usable exit code).
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("pg_restore exec: %v", err), runLogger)

			return
		}

		if restore.IsDiskExhausted(restoreResult.StderrTail) {
			// Hitting the estimate-derived ceiling is agent infra, not proof the
			// backup is corrupt — omit the exit code so the backend retries
			// (AgentSetupFailed), never BackupRejected.
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("pg_restore hit job disk ceiling: %v", err), runLogger)

			return
		}

		if !restoreResult.HasOnlyIgnorableErrors {
			runLogger.ErrorContext(
				ctx,
				"pg_restore failed",
				slog.Any("exit_code", restoreResult.PgRestoreExitCode),
				slog.Any("stderr_tail", restoreResult.StderrTail),
			)

			r.reportFailure(ctx, job.VerificationID, &restoreResult,
				fmt.Sprintf("pg_restore failed: %v", err), runLogger)

			return
		}

		// The only items pg_restore skipped are ones that say nothing about the archive's integrity:
		// CREATE/COMMENT EXTENSION for extensions this environment lacks, and a public schema the
		// target already owned — the data restored. Treat the backup as restorable and fall through
		// to stats; the backend's restored-size check is the remaining safety net.
		runLogger.WarnContext(
			ctx,
			"pg_restore completed with only ignorable errors; treating backup as restorable",
			slog.Any("exit_code", restoreResult.PgRestoreExitCode),
			slog.Any(
				"skipped_extensions",
				strings.Join(restoreResult.UnavailableExtensions, ","),
			),
			slog.Any("stderr_tail", restoreResult.StderrTail),
		)
	}

	if job.TimescaledbVersion != "" {
		if err := r.restorer.RunTimescalePostRestore(jobCtx, jobContainer.GetVerifierConn()); err != nil {
			if reportIfDiskLimitHit() {
				return
			}

			if jobCtx.Err() != nil {
				return
			}

			r.reportFailure(ctx, job.VerificationID, &restoreResult,
				fmt.Sprintf("timescaledb_post_restore: %v", err), runLogger)

			return
		}
	}

	verifierConn := jobContainer.GetVerifierConn()

	stats, err := r.verifier.CollectStats(jobCtx, verifierConn)
	if err != nil {
		if reportIfDiskLimitHit() {
			return
		}

		if jobCtx.Err() != nil {
			return
		}

		if !r.connAlive(jobCtx, verifierConn) {
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("verify (conn dead): %v", err), runLogger)

			return
		}

		r.reportFailure(ctx, job.VerificationID, &restoreResult,
			fmt.Sprintf("verify: %v", err), runLogger)

		return
	}

	if reportIfDiskLimitHit() {
		return
	}

	r.reportSuccess(ctx, job.VerificationID, restoreResult, stats, runLogger)
}

func (r *Runner) downloadBackupIntoContainer(
	jobCtx context.Context,
	jobContainer JobContainer,
	verificationID uuid.UUID,
	archivePath string,
	logger *slog.Logger,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxBackupDownloadAttempts; attempt++ {
		if jobCtx.Err() != nil {
			return errAbortNoReport
		}

		err := r.downloadBackupIntoContainerOnce(jobCtx, jobContainer, verificationID, archivePath)
		if err == nil {
			return nil
		}

		var respErr *api.ResponseError
		if errors.As(err, &respErr) {
			if respErr.IsGone() {
				logger.InfoContext(jobCtx, "backup stream gone (410); dropping without report")

				return errAbortNoReport
			}

			if !respErr.IsRetryable() {
				return fmt.Errorf("backup stream non-retryable: %w", err)
			}
		}

		lastErr = err
		logger.WarnContext(
			jobCtx,
			"backup stream attempt failed",
			slog.Any("attempt", attempt),
			slog.Any("max_attempts", maxBackupDownloadAttempts),
			slog.Any("error", err),
		)

		if !sleepOrDone(jobCtx, r.backupDownloadBackoffFn(attempt)) {
			return errAbortNoReport
		}
	}

	return fmt.Errorf("backup stream failed after %d attempts: %w", maxBackupDownloadAttempts, lastErr)
}

func (r *Runner) downloadBackupIntoContainerOnce(
	jobCtx context.Context,
	jobContainer JobContainer,
	verificationID uuid.UUID,
	archivePath string,
) error {
	body, err := r.api.DownloadBackup(jobCtx, verificationID)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	dlCtx, dlCancel := context.WithCancelCause(jobCtx)
	defer dlCancel(nil)

	idleReader := api.NewIdleTimeoutReader(body, streamIdleTimeout, dlCancel)
	defer idleReader.Stop()

	return r.restorer.StageBackupViaExec(dlCtx, jobContainer, idleReader, archivePath)
}

func (r *Runner) reportSuccess(
	ctx context.Context,
	verificationID uuid.UUID,
	restoreResult restore.Result,
	stats verifier.Stats,
	logger *slog.Logger,
) {
	tableStats := make([]agentclient.ReportTableStat, 0, len(stats.TableStats))
	for _, ts := range stats.TableStats {
		tableStats = append(tableStats, agentclient.ReportTableStat{
			SchemaName: ts.SchemaName,
			Name:       ts.Name,
			RowCount:   ts.RowCount,
		})
	}

	req := agentclient.VerificationReport{
		Status:                  agentclient.VerificationReportStatusCOMPLETED,
		PgRestoreExitCode:       &restoreResult.PgRestoreExitCode,
		RestoreDurationMs:       &restoreResult.DurationMs,
		DBSizeBytesAfterRestore: &stats.DBSizeBytes,
		TableCount:              &stats.TableCount,
		SchemaCount:             &stats.SchemaCount,
		TableStats:              tableStats,
	}

	r.sendReport(ctx, verificationID, req, logger)
}

func (r *Runner) reportFailure(
	ctx context.Context,
	verificationID uuid.UUID,
	restoreResult *restore.Result,
	failMessage string,
	logger *slog.Logger,
) {
	if r.shouldSkipFailedReport(ctx, verificationID, logger) {
		return
	}

	message := failMessage
	req := agentclient.VerificationReport{
		Status:      agentclient.VerificationReportStatusFAILED,
		FailMessage: &message,
	}

	if restoreResult != nil {
		req.PgRestoreExitCode = &restoreResult.PgRestoreExitCode
		req.RestoreDurationMs = &restoreResult.DurationMs
	}

	r.sendReport(ctx, verificationID, req, logger)
}

// reportDiskLimitExceeded posts the terminal disk-budget verdict. FailureKind
// makes the backend classify it DISK_LIMIT_EXCEEDED (terminal) instead of the
// retryable nil-exit-code path: the budget is server-computed, so retrying the
// same job against the same budget would fail identically.
func (r *Runner) reportDiskLimitExceeded(
	ctx context.Context,
	verificationID uuid.UUID,
	failMessage string,
	logger *slog.Logger,
) {
	if r.shouldSkipFailedReport(ctx, verificationID, logger) {
		return
	}

	message := failMessage
	kind := "DISK_LIMIT_EXCEEDED"
	r.sendReport(ctx, verificationID, agentclient.VerificationReport{
		Status:      agentclient.VerificationReportStatusFAILED,
		FailMessage: &message,
		FailureKind: &kind,
	}, logger)
}

// shouldSkipFailedReport is the abort/report-race guard (the only agent-side
// lever with a frozen FAILED path): never POST FAILED for an aborted/cancelled
// ID — a spurious FAILED would flip CANCELED→FAILED or resurrect a cancelled
// row.
func (r *Runner) shouldSkipFailedReport(
	ctx context.Context, verificationID uuid.UUID, logger *slog.Logger,
) bool {
	if ctx.Err() != nil || r.heartbeat.IsAborted(verificationID) {
		logger.InfoContext(ctx, "skipping FAILED report: verification aborted/cancelled")

		return true
	}

	return false
}

func (r *Runner) sendReport(
	ctx context.Context,
	verificationID uuid.UUID,
	req agentclient.VerificationReport,
	logger *slog.Logger,
) {
	err := r.api.Report(ctx, verificationID, req)
	if err == nil {
		logger.InfoContext(ctx, "report accepted", slog.Any("status", req.Status))

		return
	}

	switch {
	case errors.Is(err, api.ErrReportGone):
		logger.InfoContext(ctx, "report dropped: verification no longer owned by this agent (410)")
	case errors.Is(err, api.ErrReportBudgetExhausted):
		logger.WarnContext(ctx, "report abandoned: retry budget exhausted; backend will reclaim on next heartbeat")
	default:
		logger.WarnContext(ctx, "report failed", slog.Any("error", err))
	}
}

func pgMajorFromDatabase(db agentclient.AssignedDatabase) (string, error) {
	if db.Type != "POSTGRES_LOGICAL" {
		return "", fmt.Errorf("unsupported database type %q (v1 is Postgres only)", db.Type)
	}

	if db.PostgresqlLogical == nil || db.PostgresqlLogical.Version == "" {
		return "", errors.New("assignment missing postgresql version")
	}

	if !slices.Contains(getSupportedMajors(), db.PostgresqlLogical.Version) {
		return "", fmt.Errorf("unsupported postgres major %q", db.PostgresqlLogical.Version)
	}

	return db.PostgresqlLogical.Version, nil
}

func backupDownloadBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	d := time.Duration(1<<attempt) * time.Second
	if d > maxBackupDownloadBackoff {
		return maxBackupDownloadBackoff
	}

	return d
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
