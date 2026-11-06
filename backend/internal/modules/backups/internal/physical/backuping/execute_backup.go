package backuping

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func executeBackup(ctx context.Context, spec backupExecutionSpec) (executionResult Result) {
	defer func() {
		if executionResult.CompletedAt.IsZero() {
			executionResult.CompletedAt = spec.Clock.Now()
		}
	}()
	start := spec.Clock.Now()
	executionStartedAt := spec.ExecutionClock.Now()

	creds, err := pgshared.WriteCredentialFilesToTempDir(
		spec.SourceDB.Engine.GetCredentialSpec(),
		spec.SourceDB.Engine.Password,
	)
	if err != nil {
		return errorResult(catalog.BackupErrorPgBasebackupFailed, "write credentials", err)
	}
	defer creds.Remove()

	strategy := newBackupStrategy(spec)
	preparation, refusal, canProceed := strategy.prepare(ctx, spec)
	if !canProceed {
		return refusal
	}

	if preparation.Cleanup != nil {
		defer preparation.Cleanup()
	}

	label := buildBackupLabel(spec.DatabaseName, spec.Backup.ID, start, strategy.labelKind)
	mintAndSaveAttemptName := func() (string, error) {
		attemptName := buildObjectName(backupObjectName{
			DatabaseID: spec.SourceDB.DatabaseID, Label: label, AttemptID: uuid.New(),
		})
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
			Common: spec.CommonBackupSpec, BackupID: spec.Backup.ID, Creds: creds,
			Label: label, SystemID: spec.SourceDB.GetSystemIdentifierUint64(),
			IncrementalManifestPath: preparation.ManifestPath,
			Classify:                strategy.classify, MintAndSaveAttemptName: mintAndSaveAttemptName,
		})
		if err != nil {
			result = errorResult(catalog.BackupErrorPgBasebackupFailed, strategy.streamOperation, err)
			return nil
		}

		if streamResult.Status != catalog.BackupStatusCompleted {
			result = strategy.recheckFailure(ctx, preparation.TimelineID, streamResult)
			return nil
		}

		if strategy.validate != nil {
			streamResult = strategy.validate(ctx, streamResult)
			if streamResult.Status != catalog.BackupStatusCompleted {
				result = streamResult
				return nil
			}
		}

		streamResult.BackupDurationMs = spec.ExecutionClock.Now().Sub(executionStartedAt).Milliseconds()
		streamResult.CompletedAt = spec.Clock.Now()

		metadataReceipt, err := uploadMetadata(ctx, spec, streamResult)
		if err != nil {
			result = errorResult(catalog.BackupErrorStorageUploadFailed, "upload metadata", err)
			return nil
		}

		streamResult.Receipts = append(streamResult.Receipts, metadataReceipt)
		result = streamResult

		return nil
	})
	if slotErr != nil {
		return errorResult(catalog.BackupErrorNetworkFailure, "per-backup slot lifecycle", slotErr)
	}

	return result
}
