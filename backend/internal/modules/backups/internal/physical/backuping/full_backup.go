package backuping

import (
	"context"
	"log/slog"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
	"databasus-backend/internal/modules/storages"
)

func prepareFullBackup(ctx context.Context, spec backupExecutionSpec) (backupPreparation, Result, bool) {
	refusal, timelineID, canProceed := verifyFullTimelineCompatibility(ctx, spec.CommonBackupSpec)
	return backupPreparation{TimelineID: timelineID}, refusal, canProceed
}

func validateFullBackup(ctx context.Context, common CommonBackupSpec, streamResult Result) Result {
	validation, valErr := timelines.ValidateStartLsnAgainstHistory(
		ctx,
		common.SourceDB.DatabaseID,
		streamResult.TimelineID,
		streamResult.StartLSN,
		common.HistoryRepo,
	)
	if valErr != nil {
		common.Logger.WarnContext(ctx, "start-lsn history validation failed", slog.Any("error", valErr))
	}

	if validation.Status == timelines.ValidationStatusChainBroken {
		discardArtifactsAfterChainBroken(ctx, common, streamResult.FileName)

		reason := catalog.BackupErrorStartLsnOutsideTimeline

		return Result{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: validation.Message,
			FileName:     streamResult.FileName,
			TimelineID:   streamResult.TimelineID,
			StartLSN:     streamResult.StartLSN,
			StopLSN:      streamResult.StopLSN,
		}
	}

	if validation.Status == timelines.ValidationStatusOKWithWarning {
		common.Logger.InfoContext(
			ctx,
			"timeline history has a gap",
			slog.Any("message", validation.Message),
			slog.Any("timeline_id", streamResult.TimelineID),
		)
	}

	if streamResult.TimelineID > 1 {
		uploadHistoryForTimelineSwitch(ctx, common, streamResult.TimelineID)
	}

	return streamResult
}

// uploadHistoryForTimelineSwitch best-effort uploads the .history file for a FULL
// that ran on a timeline > 1. A failure here leaves the FULL COMPLETED: the
// artifact itself is valid, and a missing .history only degrades later chain
// validation, which the scheduler tolerates.
func uploadHistoryForTimelineSwitch(ctx context.Context, common CommonBackupSpec, timelineID int) {
	historyConn, err := pgphysical.NewSource(common.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		common.Logger.WarnContext(
			ctx,
			"could not open connection for history upload; full stays completed",
			slog.Any("timeline_id", timelineID),
			slog.Any("error", err),
		)

		return
	}
	defer func() { _ = historyConn.Close(ctx) }()

	if _, err := timelines.UploadHistoryFile(ctx, timelines.HistoryUploadSpec{
		Publication: common.Publication, Inspection: historyConn,
		TimelineID:  timelineID,
		FileStore:   common.FileStore,
		SourceDB:    common.SourceDB,
		StorageID:   common.StorageID,
		HistoryRepo: common.HistoryRepo, Clock: common.Clock, Transactor: common.Transactor,
		Encryption: common.Encryption,
		MasterKey:  common.MasterKey,
		Logger:     common.Logger,
	}); err != nil {
		common.Logger.WarnContext(
			ctx,
			"history upload failed; full stays completed",
			slog.Any("timeline_id", timelineID),
			slog.Any("error", err),
		)
	}
}

// A rejected FULL keeps nothing: its streamed artifact and reconstructed manifest
// go back to cleanup. The .metadata sidecar is written only after this point, so
// naming it here would only cost one idempotent provider call.
func discardArtifactsAfterChainBroken(
	ctx context.Context,
	common CommonBackupSpec,
	fileName string,
) {
	references := []storages.StoredFileReference{
		{StorageID: common.StorageID, FileName: fileName},
		{StorageID: common.StorageID, FileName: fileName + catalog.ManifestSuffix},
	}

	ctx = context.WithoutCancel(ctx)

	if err := common.Transactor.Transaction(ctx, func(ctx context.Context) error {
		return common.FileStore.RequestDiscardedFileDeletions(
			ctx,
			storages.DiscardedFiles{DatabaseID: common.Publication.Publication.DatabaseID, References: references},
		)
	}); err != nil {
		common.Logger.WarnContext(
			ctx,
			"failed to discard artifacts after chain_broken",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)
	}
}
