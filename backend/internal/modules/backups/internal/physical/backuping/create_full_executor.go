package backuping

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
	"databasus-backend/internal/modules/storages"
)

type FullExecutor struct{}

func NewFullExecutor() *FullExecutor { return &FullExecutor{} }

func (uc *FullExecutor) Execute(ctx context.Context, spec FullBackupSpec) (executionResult Result, executionErr error) {
	defer func() {
		if executionResult.CompletedAt.IsZero() {
			executionResult.CompletedAt = spec.Clock.Now()
		}
	}()
	start := spec.Clock.Now()
	executionStartedAt := spec.ExecutionClock.Now()

	password := spec.SourceDB.Engine.Password

	creds, err := pgshared.WriteCredentialFilesToTempDir(spec.SourceDB.Engine.GetCredentialSpec(), password)
	if err != nil {
		return errorResult(catalog.BackupErrorPgBasebackupFailed, "write credentials", err), nil
	}
	defer creds.Remove()

	refusalResult, preflightTimelineID, canProceed := verifyFullTimelineCompatibility(ctx, spec.CommonBackupSpec)
	if !canProceed {
		return refusalResult, nil
	}

	label := buildBackupLabel(spec.DatabaseName, spec.Backup.ID, start, "FULL")

	// Every codec attempt gets its own object key, and the row has to carry it
	// before the bytes leave, or a failed attempt leaves a file nothing names.
	mintAndSaveAttemptName := func() (string, error) {
		attemptName := buildObjectName(
			backupObjectName{DatabaseID: spec.SourceDB.DatabaseID, Label: label, AttemptID: uuid.New()},
		)

		spec.Backup.FileName = &attemptName
		if err := spec.Transactor.Transaction(context.WithoutCancel(ctx), func(ctx context.Context) error {
			if err := spec.Publication.Validate(ctx); err != nil {
				return err
			}
			return spec.BackupRepo.UpdateAttemptFileName(ctx, spec.Backup)
		}); err != nil {
			return "", fmt.Errorf("persist file_name at upload-start: %w", err)
		}

		return attemptName, nil
	}
	var result Result

	slotErr := WithBackupSlot(ctx, spec.SourceDB, spec.Logger, func() error {
		defer func() {
			if result.CompletedAt.IsZero() {
				result.CompletedAt = spec.Clock.Now()
			}
		}()

		streamResult, err := streamWithCodecFallback(ctx, streamAttemptSpec{
			Common:                  spec.CommonBackupSpec,
			BackupID:                spec.Backup.ID,
			Creds:                   creds,
			Label:                   label,
			SystemID:                spec.SourceDB.GetSystemIdentifierUint64(),
			IncrementalManifestPath: "",
			Classify:                classifyFullStreamError,
			MintAndSaveAttemptName:  mintAndSaveAttemptName,
		})
		if err != nil {
			result = errorResult(catalog.BackupErrorPgBasebackupFailed,
				"pg_basebackup stream", err)
			return nil
		}

		if streamResult.Status != catalog.BackupStatusCompleted {
			result = recheckFullStreamFailure(ctx, spec.CommonBackupSpec, preflightTimelineID, streamResult)
			return nil
		}

		validation, valErr := timelines.ValidateStartLsnAgainstHistory(
			ctx,
			spec.SourceDB.DatabaseID,
			streamResult.TimelineID,
			streamResult.StartLSN,
			spec.HistoryRepo,
		)
		if valErr != nil {
			spec.Logger.WarnContext(ctx, "start-lsn history validation failed", slog.Any("error", valErr))
		}

		if validation.Status == timelines.ValidationStatusChainBroken {
			discardArtifactsAfterChainBroken(ctx, spec.CommonBackupSpec, streamResult.FileName)

			reason := catalog.BackupErrorStartLsnOutsideTimeline

			result = Result{
				Status:       catalog.BackupStatusChainBroken,
				ErrorReason:  &reason,
				ErrorMessage: validation.Message,
				FileName:     streamResult.FileName,
				TimelineID:   streamResult.TimelineID,
				StartLSN:     streamResult.StartLSN,
				StopLSN:      streamResult.StopLSN,
			}

			return nil
		}

		if validation.Status == timelines.ValidationStatusOKWithWarning {
			spec.Logger.InfoContext(
				ctx,
				"timeline history has a gap",
				slog.Any("message", validation.Message),
				slog.Any("timeline_id", streamResult.TimelineID),
			)
		}

		if streamResult.TimelineID > 1 {
			uploadHistoryForTimelineSwitch(ctx, spec.CommonBackupSpec, streamResult.TimelineID)
		}

		streamResult.BackupDurationMs = spec.ExecutionClock.Now().Sub(executionStartedAt).Milliseconds()
		streamResult.CompletedAt = spec.Clock.Now()

		metadataReceipt, err := uploadFullMetadata(
			ctx,
			spec.Logger, spec.FileStore, spec.StorageID, spec.SourceDB, spec.Backup, streamResult,
		)
		if err != nil {
			result = errorResult(catalog.BackupErrorStorageUploadFailed, "upload metadata", err)
			return nil
		}

		streamResult.Receipts = append(streamResult.Receipts, metadataReceipt)
		result = streamResult

		return nil
	})

	if slotErr != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"per-backup slot lifecycle", slotErr), nil
	}

	return result, nil
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

func classifyFullStreamError(streamErr error, stderr []byte) streamOutcome {
	reason := catalog.BackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
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
