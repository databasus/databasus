package backuping

import (
	"context"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func runSummarizerPreCheck(ctx context.Context, spec backupExecutionSpec) (Result, bool) {
	conn, err := pgphysical.NewSource(spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"open inspection conn for summarizer pre-check", err), false
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	decision, err := resolveSummarizerDecision(
		ctx,
		conn,
		spec.ParentManifest.StopLSN,
		spec.IncrementalCadence,
		spec.ExecutionClock,
	)
	if err != nil {
		if ctx.Err() != nil {
			return canceledResult(catalog.BackupErrorCanceledByUser,
				"incremental cancelled during summarizer wait"), false
		}

		return errorResult(catalog.BackupErrorNetworkFailure, "summarizer pre-check", err), false
	}

	switch decision.Decision {
	case DecisionGoIncremental:
		return Result{}, true

	case DecisionRetryNextCadence, DecisionWait:
		return summarizerRefusedResult(
			catalog.BackupStatusError,
			catalog.BackupErrorSummarizerFallingBehind,
			"WAL summarizer is not publishing summaries (none yet, or its process is gone); retrying on the next cadence",
		), false

	case DecisionFullNewChain:
		return summarizerRefusedResult(
			catalog.BackupStatusChainBroken,
			*decision.Reason,
			"summarizer pre-check refused incremental; new FULL required",
		), false
	}

	panic("unknown summarizer decision")
}

func summarizerRefusedResult(
	status catalog.BackupStatus,
	reason catalog.BackupErrorReason,
	message string,
) Result {
	return Result{
		Status:       status,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}

func canceledResult(
	reason catalog.BackupErrorReason,
	message string,
) Result {
	return Result{
		Status:       catalog.BackupStatusCanceled,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}
