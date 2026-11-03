package logical

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

// metadataWriteTimeout bounds the sidecar's write, which runs after the dump on a context of its own:
// the expensive part is done, so a cancel arriving now no longer fails the backup.
const metadataWriteTimeout = 30 * time.Second

const logicalBackupProgressInterval = 5 * time.Second

// backupMetadata is the sidecar next to the backup file, which lets the recovery of a storage
// without Databasus decrypt the file.
type backupMetadata struct {
	BackupID       string            `json:"backupId"`
	EncryptionSalt *string           `json:"encryptionSalt"`
	EncryptionIV   *string           `json:"encryptionIV"`
	Encryption     config.Encryption `json:"encryption"`
}

// runBackup dumps the database, writes the sidecar and publishes the backup, then sends the
// notification and publishes BackupCompleted. Every exit writes the backup's terminal state.
func (s *Service) runBackup(runCtx context.Context, backup LogicalBackup, backupDatabase databases.Database) {
	startedAt := s.clock.Now()

	// A stop must not cut this read short: without the config a failure cannot send its notification,
	// and the stop reaches the very next step anyway.
	backupConfig, err := s.configs.GetLogicalBackupConfig(context.WithoutCancel(runCtx), backup.DatabaseID)
	if err != nil {
		s.failBackup(runCtx, backup, backupDatabase, nil, startedAt, err)

		return
	}

	result, err := s.pipeline.RunBackup(runCtx, pipeline.BackupSpec{
		BackupID:     backup.ID,
		DatabaseID:   backup.DatabaseID,
		DatabaseType: backupDatabase.Type,
		StorageID:    backup.StorageID,
		FileName:     backup.FileName,
		IsEncrypted:  backupConfig.Encryption == config.EncryptionEncrypted,
		OnProgress: func(storedSizeMb float64) {
			s.recordProgress(runCtx, backup, storedSizeMb, startedAt)
		},
		ProgressInterval: logicalBackupProgressInterval,
		Logger:           s.logger,
	})
	if errors.Is(err, runcontext.ErrBackupCancelled) {
		s.cancelBackup(runCtx, backup, startedAt)

		return
	}

	if err != nil {
		s.failBackup(runCtx, backup, backupDatabase, &backupConfig, startedAt, err)

		return
	}

	completedBackup, err := s.publishBackup(runCtx, backup, result, startedAt)
	if err != nil {
		s.failBackup(runCtx, backup, backupDatabase, &backupConfig, startedAt, err)

		return
	}

	s.logger.InfoContext(runCtx, "logical backup completed",
		slog.Float64("size_mb", completedBackup.BackupSizeMb),
		slog.Int64("duration_ms", completedBackup.BackupDurationMs),
	)

	s.notifyBackupOutcome(
		runCtx,
		completedBackup,
		backupDatabase,
		backupConfig,
		config.NotificationTriggerBackupSuccess,
	)
	s.publishBackupCompleted(runCtx, completedBackup)
}

func (s *Service) recordProgress(
	runCtx context.Context,
	backup LogicalBackup,
	storedSizeMb float64,
	startedAt time.Time,
) {
	_, err := s.repository.updateProgress(runCtx, backup.ID, storedSizeMb, s.getDurationMs(startedAt))
	if err != nil && runCtx.Err() == nil {
		s.logger.ErrorContext(runCtx, "failed to record the progress of the logical backup", slog.Any("error", err))
	}
}

// publishBackup writes the sidecar, then spends the write receipts, marks the backup completed and
// records the success on the database in one transaction. A database deleted while the backup ran
// fails the publication instead of keeping the backup's row.
func (s *Service) publishBackup(
	runCtx context.Context,
	backup LogicalBackup,
	result pipeline.BackupResult,
	startedAt time.Time,
) (LogicalBackup, error) {
	completedBackup := backup
	completedBackup.Status = StatusCompleted
	completedBackup.BackupSizeMb = result.StoredSizeMb
	completedBackup.BackupRawDbSizeMb = result.RawDatabaseSizeMb
	completedBackup.BackupDurationMs = s.getDurationMs(startedAt)
	completedBackup.TimescaledbVersion = result.TimescaledbVersion

	if result.EncryptionHeader != nil {
		completedBackup.Encryption = config.EncryptionEncrypted
		completedBackup.EncryptionSalt = &result.EncryptionHeader.Salt
		completedBackup.EncryptionIV = &result.EncryptionHeader.Nonce
	}

	metadataReceipt, err := s.writeMetadataFile(runCtx, completedBackup)
	if err != nil {
		return LogicalBackup{}, err
	}

	err = s.transactor.Transaction(context.WithoutCancel(runCtx), func(ctx context.Context) error {
		err := s.storages.ConfirmFileWrites(ctx, []storages.WriteReceipt{result.Receipt, metadataReceipt})
		if err != nil {
			return err
		}

		isCompleted, err := s.repository.complete(ctx, completion{
			backupID:           completedBackup.ID,
			sizeMb:             completedBackup.BackupSizeMb,
			rawDatabaseSizeMb:  completedBackup.BackupRawDbSizeMb,
			durationMs:         completedBackup.BackupDurationMs,
			encryption:         completedBackup.Encryption,
			encryptionSalt:     completedBackup.EncryptionSalt,
			encryptionIV:       completedBackup.EncryptionIV,
			timescaledbVersion: completedBackup.TimescaledbVersion,
		})
		if err != nil {
			return err
		}

		if !isCompleted {
			return errBackupNoLongerRunning
		}

		return s.databases.RecordBackupSuccess(ctx, completedBackup.DatabaseID, s.clock.Now().UTC())
	})
	if err != nil {
		return LogicalBackup{}, err
	}

	return completedBackup, nil
}

func (s *Service) writeMetadataFile(runCtx context.Context, backup LogicalBackup) (storages.WriteReceipt, error) {
	metadataJSON, err := json.Marshal(backupMetadata{
		BackupID:       backup.ID.String(),
		EncryptionSalt: backup.EncryptionSalt,
		EncryptionIV:   backup.EncryptionIV,
		Encryption:     backup.Encryption,
	})
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("marshal the backup metadata: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(runCtx), metadataWriteTimeout)
	defer cancel()

	receipt, err := s.storages.WriteFile(
		writeCtx,
		storages.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName + metadataSuffix},
		bytes.NewReader(metadataJSON),
	)
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("save the backup metadata file: %w", err)
	}

	return receipt, nil
}

// failBackup records the failure on the database and on the backup, discards the backup's files and
// sends the notification. backupConfig is nil when the config could not be read, which sends none.
func (s *Service) failBackup(
	runCtx context.Context,
	backup LogicalBackup,
	backupDatabase databases.Database,
	backupConfig *config.LogicalBackupConfig,
	startedAt time.Time,
	cause error,
) {
	ctx := context.WithoutCancel(runCtx)
	failMessage := cause.Error()

	s.logger.ErrorContext(ctx, "logical backup failed", slog.Any("error", cause))

	err := s.databases.RecordBackupFailure(ctx, backup.DatabaseID, failMessage)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to record the backup error on the database", slog.Any("error", err))
	}

	err = s.endBackupAndDiscardFiles(ctx, backup, func(ctx context.Context) (bool, error) {
		return s.repository.fail(ctx, backup.ID, failMessage, s.getDurationMs(startedAt))
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to record the failed logical backup", slog.Any("error", err))
	}

	if backupConfig == nil {
		return
	}

	failedBackup := backup
	failedBackup.Status = StatusFailed
	failedBackup.FailMessage = &failMessage

	s.notifyBackupOutcome(ctx, failedBackup, backupDatabase, *backupConfig, config.NotificationTriggerBackupFailed)
}

func (s *Service) cancelBackup(runCtx context.Context, backup LogicalBackup, startedAt time.Time) {
	ctx := context.WithoutCancel(runCtx)

	s.logger.WarnContext(ctx, "logical backup cancelled on request")

	err := s.endBackupAndDiscardFiles(ctx, backup, func(ctx context.Context) (bool, error) {
		return s.repository.cancel(ctx, backup.ID, s.getDurationMs(startedAt))
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to record the cancelled logical backup", slog.Any("error", err))
	}
}

// endBackupAndDiscardFiles writes the terminal state and requests the deletion of the files the
// attempt may have written in one transaction. A backup that is no longer running keeps its state
// and its files.
func (s *Service) endBackupAndDiscardFiles(
	ctx context.Context,
	backup LogicalBackup,
	writeTerminalState func(ctx context.Context) (bool, error),
) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		isEnded, err := writeTerminalState(ctx)
		if err != nil || !isEnded {
			return err
		}

		return s.requestFileDeletions(ctx, []LogicalBackup{backup})
	})
}

func (s *Service) publishBackupCompleted(ctx context.Context, backup LogicalBackup) {
	for _, handler := range s.completedBackupHandlers {
		err := handler(ctx, BackupCompleted{BackupID: backup.ID, DatabaseID: backup.DatabaseID})
		if err != nil {
			s.logger.ErrorContext(ctx, "a handler of the completed logical backup failed", slog.Any("error", err))
		}
	}
}

func (s *Service) getDurationMs(startedAt time.Time) int64 {
	return s.clock.Now().Sub(startedAt).Milliseconds()
}
