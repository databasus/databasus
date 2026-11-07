package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
	diagnosticSecrets       []string
	hasRun                  atomic.Bool
	log                     *slog.Logger
}

type Deps struct {
	DiagnosticSecrets []string
	APIClient         APIClient
	Capacity          config.Capacity
	Pool              *Pool
	Spawner           Spawner
	Restorer          Restorer
	StatsCollector    StatsCollector
	Heartbeat         Registrar
	Logger            *slog.Logger
	Group             *background.Group
}

func NewRunner(deps Deps) *Runner {
	return &Runner{
		diagnosticSecrets:       slices.Clone(deps.DiagnosticSecrets),
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
	diagnostic := pgrestore.GetFailureDiagnostic(
		pgrestore.FailureDiagnosticSpec{Output: failMessage, Secrets: r.diagnosticSecrets, IsTruncated: false},
	)
	report := agentclient.VerificationReport{
		Status:      agentclient.VerificationReportStatusFAILED,
		FailMessage: &message,
	}
	if diagnostic != nil {
		report.FailureDiagnostic = &agentclient.VerificationFailureDiagnostic{
			Output:      diagnostic.Output,
			IsTruncated: diagnostic.IsTruncated,
		}
	}
	kind := "DISK_LIMIT_EXCEEDED"
	report.FailureKind = &kind

	r.sendReport(ctx, verificationID, report, logger)
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
	req = sanitizeFailureReport(req, r.diagnosticSecrets)

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
