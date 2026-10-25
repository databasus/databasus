package backuping_logical

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	storage_files "databasus-backend/internal/features/storages/files"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	db "databasus-backend/internal/storage"
)

const (
	metadataWriteTimeout = 30 * time.Second

	metadataSuffix = ".metadata"
)

type Backuper struct {
	databaseService          *databases.DatabaseService
	workspaceService         *workspaces_bridge.Access
	backupRepository         *backups_core_logical.BackupRepository
	backupConfigService      *backups_config_logical.BackupConfigService
	fileStore                *storage_files.Store
	notificationSender       backups_core_logical.NotificationSender
	taskCancellationRegistry *tasks_cancellation.Registry
	logger                   *slog.Logger
	createBackupUseCase      backups_core_logical.CreateBackupUsecase
}

func (b *Backuper) MakeBackup(ctx context.Context, backupID uuid.UUID, isCallNotifier bool) {
	backup, err := b.backupRepository.FindByID(backupID)
	if err != nil {
		b.logger.ErrorContext(ctx, "failed to get backup by ID", "backup_id", backupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID
	logger := b.logger.With("backup_id", backupID, "database_id", databaseID)

	database, err := b.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database by ID", "error", err)
		return
	}

	backupConfig, err := b.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		logger.ErrorContext(ctx, "backup config storage ID is not defined")
		return
	}

	// Detached from the caller so a finished HTTP request cannot cancel a running backup.
	executionCtx, cancel := context.WithCancel(context.Background())
	b.taskCancellationRegistry.RegisterTask(backup.ID, cancel)
	defer b.taskCancellationRegistry.UnregisterTask(backup.ID)

	start := time.Now().UTC()

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		if err := b.backupRepository.Save(backup); err != nil {
			logger.ErrorContext(ctx, "failed to update backup progress", "error", err)
		}
	}

	artifacts, err := b.createBackupUseCase.Execute(
		executionCtx,
		backup,
		backupConfig,
		database,
		b.fileStore,
		backupProgressListener,
	)
	if err != nil {
		// Check if backup was already marked as failed by progress listener (e.g., size limit exceeded)
		// If so, skip error handling to avoid overwriting the status
		currentBackup, fetchErr := b.backupRepository.FindByID(backup.ID)
		if fetchErr == nil && currentBackup.Status == backups_core_logical.BackupStatusFailed {
			logger.WarnContext(ctx,
				"backup already marked as failed by progress listener, skipping error handling",
				"backup_id",
				backup.ID,
				"fail_message",
				*currentBackup.FailMessage,
			)

			// Still call notification for size limit failures
			b.SendBackupNotification(ctx,
				backupConfig,
				currentBackup,
				backups_config_logical.NotificationBackupFailed,
				currentBackup.FailMessage,
			)

			return
		}

		errMsg := err.Error()

		// Log detailed error information for debugging
		logger.ErrorContext(ctx, "backup execution failed",
			"backup_id", backup.ID,
			"database_id", databaseID,
			"database_type", database.Type,
			"storage_id", backup.StorageID,
			"error", err,
		)

		// Only a cancel request cancels executionCtx. Shutdown, the time limit and a failed upload
		// end the use case's own context instead, and their errors can still read as a
		// cancellation, so the error decides nothing here.
		if executionCtx.Err() != nil {
			logger.WarnContext(ctx, "backup was cancelled on request", "backup_id", backup.ID)

			backup.Status = backups_core_logical.BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := b.saveTerminalStateAndDiscardFiles(ctx, backup); err != nil {
				logger.ErrorContext(ctx, "failed to save cancelled backup", "error", err)
			}

			return
		}

		backup.FailMessage = &errMsg
		backup.Status = backups_core_logical.BackupStatusFailed
		backup.BackupDurationMs = time.Since(start).Milliseconds()
		backup.BackupSizeMb = 0

		if updateErr := b.databaseService.SetBackupError(databaseID, errMsg); updateErr != nil {
			logger.ErrorContext(ctx,
				"failed to update database last backup time",
				"error",
				updateErr,
			)
		}

		if err := b.saveTerminalStateAndDiscardFiles(ctx, backup); err != nil {
			logger.ErrorContext(ctx, "failed to save backup", "error", err)
		}

		logger.ErrorContext(ctx, fmt.Sprintf("logical backup failed after %d ms: %s",
			backup.BackupDurationMs, errMsg))

		b.SendBackupNotification(ctx,
			backupConfig,
			backup,
			backups_config_logical.NotificationBackupFailed,
			&errMsg,
		)

		return
	}

	backup.BackupDurationMs = time.Since(start).Milliseconds()

	receipts := artifacts.Receipts

	if artifacts.Metadata != nil {
		artifacts.Metadata.BackupID = backup.ID

		if err := artifacts.Metadata.Validate(); err != nil {
			b.failBackup(ctx, logger, backup, backupConfig, fmt.Errorf("validate backup metadata: %w", err))

			return
		}

		backup.EncryptionSalt = artifacts.Metadata.EncryptionSalt
		backup.EncryptionIV = artifacts.Metadata.EncryptionIV
		backup.Encryption = artifacts.Metadata.Encryption

		metadataReceipt, err := b.writeMetadataFile(backup, artifacts.Metadata)
		if err != nil {
			b.failBackup(ctx, logger, backup, backupConfig, err)

			return
		}

		receipts = append(receipts, metadataReceipt)
	}

	backup.Status = backups_core_logical.BackupStatusCompleted

	if err := b.publishBackup(ctx, backup, receipts); err != nil {
		b.failBackup(ctx, logger, backup, backupConfig, err)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("logical backup finished: %.1f MB in %d ms",
		backup.BackupSizeMb, backup.BackupDurationMs), "file_name", backup.FileName)

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := b.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		logger.ErrorContext(ctx,
			"failed to update database last backup time",
			"error",
			updateErr,
		)
	}

	if backup.Status != backups_core_logical.BackupStatusCompleted && !isCallNotifier {
		return
	}

	b.SendBackupNotification(ctx,
		backupConfig,
		backup,
		backups_config_logical.NotificationBackupSuccess,
		nil,
	)
}

func (b *Backuper) SendBackupNotification(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	backup *backups_core_logical.LogicalBackup,
	notificationType backups_config_logical.BackupNotificationType,
	errorMessage *string,
) {
	logger := b.logger.With("backup_id", backup.ID, "database_id", backupConfig.DatabaseID)

	database, err := b.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database for the backup notification", "error", err)

		return
	}

	workspace, err := b.workspaceService.GetWorkspaceByID(ctx, *database.WorkspaceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get workspace for the backup notification", "error", err)

		return
	}

	for _, notifier := range database.Notifiers {
		if !slices.Contains(
			backupConfig.SendNotificationsOn,
			notificationType,
		) {
			logger.DebugContext(ctx, fmt.Sprintf("skipping %s notification, not in the configured set",
				notificationType), "notifier_id", notifier.ID)

			continue
		}

		title := ""
		sentNotificationType := notifier_models.NotificationTypeBackupSuccess

		switch notificationType {
		case backups_config_logical.NotificationBackupFailed:
			sentNotificationType = notifier_models.NotificationTypeBackupFailed
			title = fmt.Sprintf(
				"❌ Backup failed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		case backups_config_logical.NotificationBackupSuccess:
			title = fmt.Sprintf(
				"✅ Backup completed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		}

		message := ""
		if errorMessage != nil {
			message = *errorMessage
		} else {
			// Format size conditionally
			var sizeStr string
			if backup.BackupSizeMb < 1024 {
				sizeStr = fmt.Sprintf("%.2f MB", backup.BackupSizeMb)
			} else {
				sizeGB := backup.BackupSizeMb / 1024
				sizeStr = fmt.Sprintf("%.2f GB", sizeGB)
			}

			// Format duration as "0m 0s 0ms"
			totalMs := backup.BackupDurationMs
			minutes := totalMs / (1000 * 60)
			seconds := (totalMs % (1000 * 60)) / 1000
			durationStr := fmt.Sprintf("%dm %ds", minutes, seconds)

			message = fmt.Sprintf(
				"Backup completed successfully in %s.\nCompressed backup size: %s",
				durationStr,
				sizeStr,
			)
		}

		b.notificationSender.SendNotification(ctx,
			&notifier,
			notifier_models.Notification{
				Type:    sentNotificationType,
				Heading: title,
				Message: message,
			},
		)
	}
}

// The sidecar carries its own context because the dump already succeeded: a caller
// cancelling here would otherwise fail a backup whose expensive part is done.
func (b *Backuper) writeMetadataFile(
	backup *backups_core_logical.LogicalBackup,
	metadata *backups_core_logical.BackupMetadata,
) (storage_files.WriteReceipt, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return storage_files.WriteReceipt{}, fmt.Errorf("marshal backup metadata: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(context.Background(), metadataWriteTimeout)
	defer cancel()

	receipt, err := b.fileStore.WriteFile(
		writeCtx,
		storage_files.StoredFileReference{
			StorageID: backup.StorageID,
			FileName:  backup.FileName + metadataSuffix,
		},
		bytes.NewReader(metadataJSON),
	)
	if err != nil {
		return storage_files.WriteReceipt{}, fmt.Errorf("save backup metadata file: %w", err)
	}

	return receipt, nil
}

// Every file the receipts name is kept only if the row marking the backup completed
// commits with them.
func (b *Backuper) publishBackup(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	receipts []storage_files.WriteReceipt,
) error {
	return db.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := b.fileStore.ConfirmFileWrites(ctx, tx, receipts); err != nil {
			return err
		}

		return b.backupRepository.SaveInTransaction(tx, backup)
	})
}

// Per-attempt naming makes the two names exact, so nothing else can be writing
// them while this runs.
func (b *Backuper) saveTerminalStateAndDiscardFiles(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
) error {
	references := []storage_files.StoredFileReference{
		{StorageID: backup.StorageID, FileName: backup.FileName},
		{StorageID: backup.StorageID, FileName: backup.FileName + metadataSuffix},
	}

	// MakeBackup runs on a context detached from the caller, but a terminal state has
	// to be recorded even when that context is the one that was cancelled.
	ctx = context.WithoutCancel(ctx)

	return db.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := b.fileStore.RequestFileDeletions(ctx, tx, references); err != nil {
			return err
		}

		return b.backupRepository.SaveInTransaction(tx, backup)
	})
}

func (b *Backuper) failBackup(
	ctx context.Context,
	logger *slog.Logger,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	cause error,
) {
	message := cause.Error()

	backup.Status = backups_core_logical.BackupStatusFailed
	backup.FailMessage = &message
	backup.BackupSizeMb = 0

	logger.ErrorContext(ctx, "logical backup failed after the dump", "error", cause)

	if err := b.databaseService.SetBackupError(backup.DatabaseID, message); err != nil {
		logger.ErrorContext(ctx, "failed to record the backup error on the database", "error", err)
	}

	if err := b.saveTerminalStateAndDiscardFiles(ctx, backup); err != nil {
		logger.ErrorContext(ctx, "failed to save failed backup", "error", err)
	}

	b.SendBackupNotification(ctx, backupConfig, backup, backups_config_logical.NotificationBackupFailed, &message)
}
