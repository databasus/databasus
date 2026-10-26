package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	chain_view "databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/modules/storages"
	db "databasus-backend/internal/storage"
)

type CreateFullBackupUsecase struct{}

func NewCreateFullBackupUsecase() *CreateFullBackupUsecase { return &CreateFullBackupUsecase{} }

func (uc *CreateFullBackupUsecase) Execute(ctx context.Context, spec FullBackupSpec) (PhysicalBackupResult, error) {
	start := time.Now().UTC()

	password, err := postgresql_shared.DecryptFieldIfNeeded(spec.SourceDB.Password, spec.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed, "decrypt password", err), nil
	}

	creds, err := postgresql_shared.WriteCredentialFilesToTempDir(
		spec.SourceDB.CredentialSpec(), password, spec.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed, "write credentials", err), nil
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
		attemptName := buildObjectName(label, uuid.New())

		spec.Backup.FileName = &attemptName
		if err := spec.FullRepo.Save(spec.Backup); err != nil {
			return "", fmt.Errorf("persist file_name at upload-start: %w", err)
		}

		return attemptName, nil
	}
	var result PhysicalBackupResult

	slotErr := WithBackupSlot(ctx, spec.SourceDB, spec.FieldEncryptor, spec.Logger, func() error {
		streamResult, err := streamWithCodecFallback(ctx, streamAttemptSpec{
			Common:                  spec.CommonBackupSpec,
			BackupID:                spec.Backup.ID,
			Creds:                   creds,
			Label:                   label,
			SystemID:                spec.SourceDB.SystemIdentifierUint64(),
			IncrementalManifestPath: "",
			Classify:                classifyFullStreamError,
			MintAndSaveAttemptName:  mintAndSaveAttemptName,
		})
		if err != nil {
			result = errorResult(physical_enums.PhysicalBackupErrorPgBasebackupFailed,
				"pg_basebackup stream", err)
			return nil
		}

		if streamResult.Status != physical_enums.PhysicalBackupStatusCompleted {
			result = recheckFullStreamFailure(ctx, spec.CommonBackupSpec, preflightTimelineID, streamResult)
			return nil
		}

		validation, valErr := ValidateStartLsnAgainstHistory(
			spec.SourceDB.ParentDatabaseID(),
			streamResult.TimelineID,
			streamResult.StartLSN,
			spec.HistoryRepo,
		)
		if valErr != nil {
			spec.Logger.Warn("start-LSN history validation failed",
				"error", valErr)
		}

		if validation.Status == chain_view.ValidationStatusChainBroken {
			discardArtifactsAfterChainBroken(
				ctx, spec.FileStore, spec.StorageID, streamResult.FileName, spec.Logger)

			reason := physical_enums.PhysicalBackupErrorStartLsnOutsideTimeline

			result = PhysicalBackupResult{
				Status:       physical_enums.PhysicalBackupStatusChainBroken,
				ErrorReason:  &reason,
				ErrorMessage: validation.Message,
				FileName:     streamResult.FileName,
				TimelineID:   streamResult.TimelineID,
				StartLSN:     streamResult.StartLSN,
				StopLSN:      streamResult.StopLSN,
				CompletedAt:  time.Now().UTC(),
			}

			return nil
		}

		if validation.Status == chain_view.ValidationStatusOKWithWarning {
			spec.Logger.Info(validation.Message,
				"timeline_id", streamResult.TimelineID)
		}

		if streamResult.TimelineID > 1 {
			uploadHistoryForTimelineSwitch(ctx, spec.CommonBackupSpec, streamResult.TimelineID)
		}

		streamResult.BackupDurationMs = time.Since(start).Milliseconds()
		streamResult.CompletedAt = time.Now().UTC()

		metadataReceipt, err := uploadFullMetadata(
			spec.Logger, spec.FileStore, spec.StorageID, spec.SourceDB, spec.Backup, streamResult,
		)
		if err != nil {
			result = errorResult(physical_enums.PhysicalBackupErrorStorageUploadFailed, "upload metadata", err)
			return nil
		}

		streamResult.Receipts = append(streamResult.Receipts, metadataReceipt)
		result = streamResult

		return nil
	})

	if slotErr != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"per-backup slot lifecycle", slotErr), nil
	}

	return result, nil
}

// uploadHistoryForTimelineSwitch best-effort uploads the .history file for a FULL
// that ran on a timeline > 1. A failure here leaves the FULL COMPLETED: the
// artifact itself is valid, and a missing .history only degrades later chain
// validation, which the scheduler tolerates.
func uploadHistoryForTimelineSwitch(ctx context.Context, common CommonBackupSpec, timelineID int) {
	historyConn, err := common.SourceDB.OpenInspectionConn(ctx, common.FieldEncryptor)
	if err != nil {
		common.Logger.Warn("could not open connection for history upload; FULL stays COMPLETED",
			"timeline_id", timelineID,
			"error", err)

		return
	}
	defer func() { _ = historyConn.Close(ctx) }()

	if _, err := UploadHistoryFile(ctx, HistoryUploadSpec{
		Conn:           historyConn,
		TimelineID:     timelineID,
		FileStore:      common.FileStore,
		SourceDB:       common.SourceDB,
		StorageID:      common.StorageID,
		HistoryRepo:    common.HistoryRepo,
		Encryption:     common.Encryption,
		MasterKey:      common.MasterKey,
		FieldEncryptor: common.FieldEncryptor,
		Logger:         common.Logger,
	}); err != nil {
		common.Logger.Warn("history upload failed; FULL stays COMPLETED",
			"timeline_id", timelineID,
			"error", err)
	}
}

func classifyFullStreamError(streamErr error, stderr []byte) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
	}
}

// A rejected FULL keeps nothing: its streamed artifact and reconstructed manifest
// go back to cleanup. The .metadata sidecar is written only after this point, so
// naming it here would only cost one idempotent provider call.
func discardArtifactsAfterChainBroken(
	ctx context.Context,
	fileStore FileStore,
	storageID uuid.UUID,
	fileName string,
	logger *slog.Logger,
) {
	references := []storages.StoredFileReference{
		{StorageID: storageID, FileName: fileName},
		{StorageID: storageID, FileName: fileName + manifestSuffix},
	}

	ctx = context.WithoutCancel(ctx)

	if err := db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		return fileStore.RequestFileDeletions(ctx, references)
	}); err != nil {
		logger.Warn("failed to discard artifacts after CHAIN_BROKEN", "file_name", fileName, "error", err)
	}
}
