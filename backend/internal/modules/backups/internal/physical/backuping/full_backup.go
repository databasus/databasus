package backuping

import (
	"context"
	"log/slog"

	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

func prepareFullBackup(ctx context.Context, spec backupExecutionSpec) (backupPreparation, Result, bool) {
	refusal, timelineID, canProceed := verifyFullTimelineCompatibility(ctx, spec.CommonBackupSpec)
	return backupPreparation{TimelineID: timelineID}, refusal, canProceed
}

func validateFullBackup(ctx context.Context, common CommonBackupSpec, streamResult Result) Result {
	if streamResult.TimelineID > 1 {
		uploadHistoryForTimelineSwitch(ctx, common, streamResult.TimelineID)
	}

	return streamResult
}

// A valid FULL remains usable independently of separately archived timeline history.
func uploadHistoryForTimelineSwitch(ctx context.Context, common CommonBackupSpec, timelineID int) {
	if _, err := timelines.UploadHistoryFile(ctx, timelines.HistoryUploadSpec{
		Publication: common.Publication,
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
