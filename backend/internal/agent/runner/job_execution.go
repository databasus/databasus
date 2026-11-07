package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/agentclient"
)

type verificationJobSpec struct {
	JobContext        context.Context
	Assignment        *agentclient.VerificationAssignment
	DiagnosticSecrets []string
	Logger            *slog.Logger
}

type verificationRestoreSpec struct {
	verificationJobSpec
	Container            JobContainer
	ArchivePath          string
	ReportIfDiskLimitHit func() bool
}

func (r *Runner) executeJob(ctx context.Context, job *agentclient.VerificationAssignment) {
	runLogger := r.log.With(
		slog.Any("job_id", uuid.New()),
		slog.Any("job_name", jobName),
		slog.Any("verification_id", job.VerificationID),
	)

	diagnosticSecrets := append([]string(nil), r.diagnosticSecrets...)

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register FIRST so the ID is in the heartbeat envelope before any
	// container exists: the backend must never see an artifact-bearing job
	// that is absent from this agent's reported in-flight set.
	r.heartbeat.TrackVerification(job.VerificationID, cancel)
	defer r.heartbeat.UntrackVerification(job.VerificationID)

	jobSpec := verificationJobSpec{
		JobContext:        jobCtx,
		Assignment:        job,
		DiagnosticSecrets: diagnosticSecrets,
		Logger:            runLogger,
	}
	jobContainer, isPrepared := r.prepareVerificationContainer(ctx, jobSpec)
	if !isPrepared {
		return
	}

	diagnosticSecrets = append(
		diagnosticSecrets,
		jobContainer.GetVerifierConn().Password,
		jobContainer.GetInContainerConn().Password,
	)

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

	// A zero budget has no enforceable ceiling; starting the watcher would trip immediately.
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

	jobSpec.DiagnosticSecrets = diagnosticSecrets
	execution := verificationRestoreSpec{
		verificationJobSpec:  jobSpec,
		Container:            jobContainer,
		ArchivePath:          archivePath,
		ReportIfDiskLimitHit: reportIfDiskLimitHit,
	}
	if !r.downloadVerificationBackup(ctx, execution) {
		return
	}

	restoreResult, isRestored := r.restoreVerificationBackup(ctx, execution)
	if !isRestored {
		return
	}

	stats, isVerified := r.collectVerificationStats(ctx, execution, restoreResult)
	if !isVerified {
		return
	}

	if reportIfDiskLimitHit() {
		return
	}

	r.reportSuccess(ctx, job.VerificationID, restoreResult, stats, runLogger)
}

func (r *Runner) prepareVerificationContainer(ctx context.Context, spec verificationJobSpec) (JobContainer, bool) {
	pgMajor, err := pgMajorFromDatabase(spec.Assignment.Database)
	if err != nil {
		r.reportFailure(
			ctx,
			failureReport{
				Cause:          err,
				Secrets:        spec.DiagnosticSecrets,
				VerificationID: spec.Assignment.VerificationID,
				RestoreResult:  nil,
				Message:        fmt.Sprintf("resolve pg major: %v", err),
				Logger:         spec.Logger,
			},
		)

		return nil, false
	}

	jobContainer, err := r.spawner.Spawn(spec.JobContext, SpawnRequest{
		PgMajor:            pgMajor,
		CPUPerJob:          r.capacity.CPUPerJob,
		RAMMbPerJob:        r.capacity.RAMMbPerJob,
		VerificationID:     spec.Assignment.VerificationID,
		TimescaledbVersion: spec.Assignment.TimescaledbVersion,
	})
	if err != nil {
		if spec.JobContext.Err() != nil {
			return nil, false
		}

		r.reportFailure(
			ctx,
			failureReport{
				Cause:          err,
				Secrets:        spec.DiagnosticSecrets,
				VerificationID: spec.Assignment.VerificationID,
				RestoreResult:  nil,
				Message:        fmt.Sprintf("spawn container: %v", err),
				Logger:         spec.Logger,
			},
		)

		return nil, false
	}

	return jobContainer, true
}

func (r *Runner) downloadVerificationBackup(ctx context.Context, spec verificationRestoreSpec) bool {
	if err := r.downloadBackupIntoContainer(
		spec.JobContext,
		spec.Container,
		spec.Assignment.VerificationID,
		spec.ArchivePath,
		spec.Logger,
	); err != nil {
		if spec.ReportIfDiskLimitHit() {
			return false
		}

		if errors.Is(err, errAbortNoReport) {
			return false
		}

		r.reportFailure(
			ctx,
			failureReport{
				Cause:          err,
				Secrets:        spec.DiagnosticSecrets,
				VerificationID: spec.Assignment.VerificationID,
				RestoreResult:  nil,
				Message:        fmt.Sprintf("download backup: %v", err),
				Logger:         spec.Logger,
			},
		)

		return false
	}

	return true
}

func (r *Runner) prepareVerificationRestore(ctx context.Context, spec verificationRestoreSpec) bool {
	archiveOwnerRoleNames, err := r.restorer.EnsureArchiveOwnerRoles(
		spec.JobContext, spec.Container, spec.ArchivePath, spec.Container.GetVerifierConn())
	if err != nil {
		if spec.ReportIfDiskLimitHit() {
			return false
		}

		if spec.JobContext.Err() != nil {
			return false
		}

		r.reportFailure(
			ctx,
			failureReport{
				Cause:          err,
				Secrets:        spec.DiagnosticSecrets,
				VerificationID: spec.Assignment.VerificationID,
				RestoreResult:  nil,
				Message:        fmt.Sprintf("ensure archive owner roles: %v", err),
				Logger:         spec.Logger,
			},
		)

		return false
	}

	if len(archiveOwnerRoleNames) > 0 {
		spec.Logger.InfoContext(
			ctx,
			"archive owner roles present in restore target",
			slog.Any("roles", strings.Join(archiveOwnerRoleNames, ",")),
		)
	}

	isTimescaleDB := spec.Assignment.TimescaledbVersion != ""

	// On failure the container is discarded. The post step is needed only before collecting stats.
	if isTimescaleDB {
		if err := r.restorer.RunTimescalePreRestore(spec.JobContext, spec.Container.GetVerifierConn()); err != nil {
			if spec.ReportIfDiskLimitHit() {
				return false
			}

			if spec.JobContext.Err() != nil {
				return false
			}

			r.reportFailure(
				ctx,
				failureReport{
					Cause:          err,
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  nil,
					Message:        fmt.Sprintf("timescaledb_pre_restore: %v", err),
					Logger:         spec.Logger,
				},
			)

			return false
		}
	}

	return true
}

func (r *Runner) restoreVerificationBackup(ctx context.Context, spec verificationRestoreSpec) (restore.Result, bool) {
	if !r.prepareVerificationRestore(ctx, spec) {
		return restore.Result{}, false
	}

	isTimescaleDB := spec.Assignment.TimescaledbVersion != ""
	parallelJobs := pgrestore.GetParallelJobs(r.capacity.CPUPerJob, isTimescaleDB)

	// RunPgRestore returns Result by value and fills it when it fails with ErrRestoreFailed.
	// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
	restoreResult, err := r.restorer.RunPgRestore(spec.JobContext, spec.Container, restore.PgRestoreSpec{
		DiagnosticSecrets: spec.DiagnosticSecrets,
		ArchivePath:       spec.ArchivePath,
		Conn:              spec.Container.GetInContainerConn(),
		ParallelJobs:      parallelJobs,
		IsTimescaledb:     isTimescaleDB,
	})
	if err != nil {
		if spec.ReportIfDiskLimitHit() {
			return restoreResult, false
		}

		if spec.JobContext.Err() != nil {
			return restoreResult, false
		}

		if !errors.Is(err, restore.ErrRestoreFailed) {
			// Exec-infrastructure failure (no usable exit code).
			r.reportFailure(
				ctx,
				failureReport{
					Cause:          err,
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  nil,
					Message:        fmt.Sprintf("pg_restore exec: %v", err),
					Logger:         spec.Logger,
					Diagnostic:     restoreResult.FailureDiagnostic,
				},
			)

			return restoreResult, false
		}

		if restore.IsDiskExhausted(restoreResult.StderrTail) {
			// Hitting the estimate-derived ceiling is agent infra, not proof the
			// backup is corrupt — omit the exit code so the backend retries
			// (AgentSetupFailed), never BackupRejected.
			r.reportFailure(
				ctx,
				failureReport{
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  nil,
					Message:        fmt.Sprintf("pg_restore hit job disk ceiling: %v", err),
					Logger:         spec.Logger,
					Diagnostic:     restoreResult.FailureDiagnostic,
				},
			)

			return restoreResult, false
		}

		if !restoreResult.HasOnlyIgnorableErrors {
			spec.Logger.ErrorContext(
				ctx,
				"pg_restore failed",
				slog.Any("exit_code", restoreResult.PgRestoreExitCode),
				slog.Any("stderr_tail", restoreResult.StderrTail),
			)

			r.reportFailure(
				ctx,
				failureReport{
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  &restoreResult,
					Message:        fmt.Sprintf("pg_restore failed: %v", err),
					Logger:         spec.Logger,
					Diagnostic:     restoreResult.FailureDiagnostic,
				},
			)

			return restoreResult, false
		}

		// The only items pg_restore skipped are ones that say nothing about the archive's integrity:
		// CREATE/COMMENT EXTENSION for extensions this environment lacks, and a public schema the
		// target already owned — the data restored. Treat the backup as restorable and fall through
		// to stats; the backend's restored-size check is the remaining safety net.
		spec.Logger.WarnContext(
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

	if spec.Assignment.TimescaledbVersion != "" {
		if err := r.restorer.RunTimescalePostRestore(spec.JobContext, spec.Container.GetVerifierConn()); err != nil {
			if spec.ReportIfDiskLimitHit() {
				return restoreResult, false
			}

			if spec.JobContext.Err() != nil {
				return restoreResult, false
			}

			r.reportFailure(
				ctx,
				failureReport{
					Cause:          err,
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  &restoreResult,
					Message:        fmt.Sprintf("timescaledb_post_restore: %v", err),
					Logger:         spec.Logger,
				},
			)

			return restoreResult, false
		}
	}

	return restoreResult, true
}

func (r *Runner) collectVerificationStats(
	ctx context.Context,
	spec verificationRestoreSpec,
	restoreResult restore.Result,
) (verifier.Stats, bool) {
	verifierConn := spec.Container.GetVerifierConn()

	stats, err := r.verifier.CollectStats(spec.JobContext, verifierConn)
	if err != nil {
		if spec.ReportIfDiskLimitHit() {
			return stats, false
		}

		if spec.JobContext.Err() != nil {
			return stats, false
		}

		if !r.connAlive(spec.JobContext, verifierConn) {
			r.reportFailure(
				ctx,
				failureReport{
					Cause:          err,
					Secrets:        spec.DiagnosticSecrets,
					VerificationID: spec.Assignment.VerificationID,
					RestoreResult:  nil,
					Message:        fmt.Sprintf("verify (conn dead): %v", err),
					Logger:         spec.Logger,
				},
			)

			return stats, false
		}

		r.reportFailure(
			ctx,
			failureReport{
				Cause:          err,
				Secrets:        spec.DiagnosticSecrets,
				VerificationID: spec.Assignment.VerificationID,
				RestoreResult:  &restoreResult,
				Message:        fmt.Sprintf("verify: %v", err),
				Logger:         spec.Logger,
			},
		)

		return stats, false
	}

	return stats, true
}
