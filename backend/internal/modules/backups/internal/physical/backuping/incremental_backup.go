package backuping

import (
	"context"
	"fmt"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

func prepareIncrementalBackup(ctx context.Context, spec backupExecutionSpec) (backupPreparation, Result, bool) {
	refusal, timelineID, canProceed := verifyIncrTimelineCompatibility(ctx, spec)
	if !canProceed {
		return backupPreparation{}, refusal, false
	}

	refusal, canProceed = runSummarizerPreCheck(ctx, spec)
	if !canProceed {
		return backupPreparation{}, refusal, false
	}

	manifestPath, cleanup, err := downloadParentManifest(ctx, spec)
	if err != nil {
		return backupPreparation{}, Result{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  new(catalog.BackupErrorParentManifestMissing),
			ErrorMessage: fmt.Sprintf("download parent manifest: %v", err),
		}, false
	}

	return backupPreparation{TimelineID: timelineID, ManifestPath: manifestPath, Cleanup: cleanup}, Result{}, true
}

func verifyIncrTimelineCompatibility(
	ctx context.Context,
	spec backupExecutionSpec,
) (Result, int, bool) {
	rootFull, _, err := spec.BackupRepo.FindByID(ctx, *spec.Backup.RootFullBackupID)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure, "load root FULL", err), 0, false
	}

	if rootFull == nil || rootFull.Status != catalog.BackupStatusCompleted {
		reason := catalog.BackupErrorParentManifestMissing

		return Result{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: "root FULL is missing or incomplete",
		}, 0, false
	}

	conn, err := pgphysical.NewSource(spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"open inspection connection", err), 0, false
	}
	defer func() { _ = conn.Close(ctx) }()

	decision, err := timelines.CheckIncrementalTimelineCompatibility(ctx, conn, spec.SourceDB, rootFull.TimelineID)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"timeline compatibility check", err), 0, false
	}

	refusalResult, canProceed := timelineRefusalResult(decision, true)

	return refusalResult, rootFull.TimelineID, canProceed
}
