package backuping

import (
	"context"
	"fmt"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

func prepareIncrementalBackup(ctx context.Context, spec backupExecutionSpec) (backupPreparation, Result, bool) {
	preparation, refusal, canProceed := verifyIncrTimelineCompatibility(ctx, spec)
	if !canProceed {
		return backupPreparation{}, refusal, false
	}

	refusal, canProceed = runSummarizerPreCheck(ctx, spec, preparation.IsAcrossPromotion)
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

	preparation.ManifestPath = manifestPath
	preparation.Cleanup = cleanup

	return preparation, Result{}, true
}

// verifyIncrTimelineCompatibility compares the live timeline with the timeline of the chain's
// newest backup, the parent of this INCREMENTAL, so a chain that already crossed a promotion is
// not compared with its root FULL's timeline again.
func verifyIncrTimelineCompatibility(
	ctx context.Context,
	spec backupExecutionSpec,
) (backupPreparation, Result, bool) {
	rootFull, _, err := spec.BackupRepo.FindByID(ctx, *spec.Backup.RootFullBackupID)
	if err != nil {
		return backupPreparation{}, errorResult(catalog.BackupErrorNetworkFailure, "load root FULL", err), false
	}

	if rootFull == nil || rootFull.Status != catalog.BackupStatusCompleted {
		return backupPreparation{}, parentMissingResult("root FULL is missing or incomplete"), false
	}

	newestBackup, _, err := spec.BackupRepo.FindByID(ctx, spec.ParentManifest.BackupID)
	if err != nil {
		return backupPreparation{}, errorResult(catalog.BackupErrorNetworkFailure, "load parent backup", err), false
	}

	if newestBackup == nil || newestBackup.Status != catalog.BackupStatusCompleted || newestBackup.StopLSN == nil {
		return backupPreparation{}, parentMissingResult("parent backup is missing or incomplete"), false
	}

	decision, err := checkLiveTimelineAgainst(ctx, spec, newestBackup.TimelineID)
	if err != nil {
		return backupPreparation{}, errorResult(
			catalog.BackupErrorNetworkFailure,
			"timeline compatibility check",
			err,
		), false
	}

	if decision.Kind == timelines.TimelineFailoverDetected {
		decision, err = continueAcrossPromotion(ctx, promotionContinuationSpec{
			Extension: chains.ChainExtension{
				RootFull: rootFull, NewestBackup: newestBackup, LiveTimelineID: decision.LiveTimelineID,
			},
			Decision: decision,
			PublishHistory: func(ctx context.Context, timelineID int) (*catalog.WalHistoryFile, error) {
				return timelines.UploadHistoryFile(ctx, newHistoryUploadSpec(spec.CommonBackupSpec, timelineID))
			},
			LoadHistories: func(ctx context.Context) ([]*catalog.WalHistoryFile, error) {
				return spec.HistoryRepo.FindAllByDatabase(ctx, spec.SourceDB.DatabaseID)
			},
			Logger: spec.Logger,
		})
		if err != nil {
			return backupPreparation{}, errorResult(
				catalog.BackupErrorNetworkFailure,
				"timeline line check",
				err,
			), false
		}
	}

	refusal, canProceed := timelineRefusalResult(decision, true)

	return backupPreparation{
		TimelineID:        decision.LiveTimelineID,
		IsAcrossPromotion: decision.Kind == timelines.TimelineContinueAcrossPromotion,
	}, refusal, canProceed
}

func checkLiveTimelineAgainst(
	ctx context.Context,
	spec backupExecutionSpec,
	expectedTimelineID int,
) (*timelines.TimelineDecision, error) {
	conn, err := pgphysical.NewSource(spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return nil, fmt.Errorf("open inspection connection: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	return timelines.CheckIncrementalTimelineCompatibility(ctx, conn, spec.SourceDB, expectedTimelineID)
}

func parentMissingResult(message string) Result {
	return Result{
		Status:       catalog.BackupStatusChainBroken,
		ErrorReason:  new(catalog.BackupErrorParentManifestMissing),
		ErrorMessage: message,
	}
}
