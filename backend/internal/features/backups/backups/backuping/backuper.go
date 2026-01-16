package backuping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"databasus-backend/internal/config"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	task_registry "databasus-backend/internal/features/tasks/registry"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"

	"github.com/google/uuid"
)

const (
	heartbeatTickerInterval     = 15 * time.Second
	backuperHeathcheckThreshold = 5 * time.Minute
)

type BackuperNode struct {
	databaseService     *databases.DatabaseService
	fieldEncryptor      util_encryption.FieldEncryptor
	workspaceService    *workspaces_services.WorkspaceService
	backupRepository    *backups_core.BackupRepository
	backupConfigService *backups_config.BackupConfigService
	storageService      *storages.StorageService
	notificationSender  backups_core.NotificationSender
	backupCancelManager *tasks_cancellation.TaskCancelManager
	tasksRegistry       *task_registry.TaskNodesRegistry
	logger              *slog.Logger
	createBackupUseCase backups_core.CreateBackupUsecase
	nodeID              uuid.UUID

	lastHeartbeat time.Time
}

func (n *BackuperNode) Run(ctx context.Context) {
	n.lastHeartbeat = time.Now().UTC()

	throughputMBs := config.GetEnv().NodeNetworkThroughputMBs

	backupNode := task_registry.TaskNode{
		ID:            n.nodeID,
		ThroughputMBs: throughputMBs,
	}

	if err := n.tasksRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), backupNode); err != nil {
		n.logger.Error("Failed to register node in registry", "error", err)
		panic(err)
	}

	backupHandler := func(backupID uuid.UUID, isCallNotifier bool) {
		n.MakeBackup(backupID, isCallNotifier)
		if err := n.tasksRegistry.PublishTaskCompletion(n.nodeID.String(), backupID); err != nil {
			n.logger.Error(
				"Failed to publish backup completion",
				"error",
				err,
				"backupID",
				backupID,
			)
		}
	}

	if err := n.tasksRegistry.SubscribeNodeForTasksAssignment(n.nodeID.String(), backupHandler); err != nil {
		n.logger.Error("Failed to subscribe to backup assignments", "error", err)
		panic(err)
	}
	defer func() {
		if err := n.tasksRegistry.UnsubscribeNodeForTasksAssignments(); err != nil {
			n.logger.Error("Failed to unsubscribe from backup assignments", "error", err)
		}
	}()

	ticker := time.NewTicker(heartbeatTickerInterval)
	defer ticker.Stop()

	n.logger.Info("Backup node started", "nodeID", n.nodeID, "throughput", throughputMBs)

	for {
		select {
		case <-ctx.Done():
			n.logger.Info("Shutdown signal received, unregistering node", "nodeID", n.nodeID)

			if err := n.tasksRegistry.UnregisterNodeFromRegistry(backupNode); err != nil {
				n.logger.Error("Failed to unregister node from registry", "error", err)
			}

			return
		case <-ticker.C:
			n.sendHeartbeat(&backupNode)
		}
	}
}

func (n *BackuperNode) IsBackuperRunning() bool {
	return n.lastHeartbeat.After(time.Now().UTC().Add(-backuperHeathcheckThreshold))
}

func (n *BackuperNode) MakeBackup(backupID uuid.UUID, isCallNotifier bool) {
	backup, err := n.backupRepository.FindByID(backupID)
	if err != nil {
		n.logger.Error("Failed to get backup by ID", "backupId", backupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := n.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		n.logger.Error("Failed to get database by ID", "databaseId", databaseID, "error", err)
		return
	}

	backupConfig, err := n.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		n.logger.Error("Failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		n.logger.Error("Backup config storage ID is not defined")
		return
	}

	storage, err := n.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		n.logger.Error("Failed to get storage by ID", "error", err)
		return
	}

	start := time.Now().UTC()

	backupProgressListener := func(
		completedMBs float64,
	) {
		backup.BackupSizeMb = completedMBs
		backup.BackupDurationMs = time.Since(start).Milliseconds()

		if err := n.backupRepository.Save(backup); err != nil {
			n.logger.Error("Failed to update backup progress", "error", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.backupCancelManager.RegisterTask(backup.ID, cancel)
	defer n.backupCancelManager.UnregisterTask(backup.ID)

	backupMetadata, err := n.createBackupUseCase.Execute(
		ctx,
		backup.ID,
		backupConfig,
		database,
		storage,
		backupProgressListener,
	)
	if err != nil {
		errMsg := err.Error()

		// Log detailed error information for debugging
		n.logger.Error("Backup execution failed",
			"backupId", backup.ID,
			"databaseId", databaseID,
			"databaseType", database.Type,
			"storageId", storage.ID,
			"storageType", storage.Type,
			"error", err,
			"errorMessage", errMsg,
		)

		// Check if backup was cancelled (not due to shutdown)
		isCancelled := strings.Contains(errMsg, "backup cancelled") ||
			strings.Contains(errMsg, "context canceled") ||
			errors.Is(err, context.Canceled)
		isShutdown := strings.Contains(errMsg, "shutdown")

		if isCancelled && !isShutdown {
			n.logger.Warn("Backup was cancelled by user or system",
				"backupId", backup.ID,
				"isCancelled", isCancelled,
				"isShutdown", isShutdown,
			)

			backup.Status = backups_core.BackupStatusCanceled
			backup.BackupDurationMs = time.Since(start).Milliseconds()
			backup.BackupSizeMb = 0

			if err := n.backupRepository.Save(backup); err != nil {
				n.logger.Error("Failed to save cancelled backup", "error", err)
			}

			// Delete partial backup from storage
			storage, storageErr := n.storageService.GetStorageByID(backup.StorageID)
			if storageErr == nil {
				if deleteErr := storage.DeleteFile(n.fieldEncryptor, backup.ID); deleteErr != nil {
					n.logger.Error(
						"Failed to delete partial backup file",
						"backupId",
						backup.ID,
						"error",
						deleteErr,
					)
				}
			}

			return
		}

		backup.FailMessage = &errMsg
		backup.Status = backups_core.BackupStatusFailed
		backup.BackupDurationMs = time.Since(start).Milliseconds()
		backup.BackupSizeMb = 0

		if updateErr := n.databaseService.SetBackupError(databaseID, errMsg); updateErr != nil {
			n.logger.Error(
				"Failed to update database last backup time",
				"databaseId",
				databaseID,
				"error",
				updateErr,
			)
		}

		if err := n.backupRepository.Save(backup); err != nil {
			n.logger.Error("Failed to save backup", "error", err)
		}

		n.SendBackupNotification(
			backupConfig,
			backup,
			backups_config.NotificationBackupFailed,
			&errMsg,
		)

		return
	}

	backup.Status = backups_core.BackupStatusCompleted
	backup.BackupDurationMs = time.Since(start).Milliseconds()

	// Update backup with encryption metadata if provided
	if backupMetadata != nil {
		backup.EncryptionSalt = backupMetadata.EncryptionSalt
		backup.EncryptionIV = backupMetadata.EncryptionIV
		backup.Encryption = backupMetadata.Encryption
	}

	if err := n.backupRepository.Save(backup); err != nil {
		n.logger.Error("Failed to save backup", "error", err)
		return
	}

	// Update database last backup time
	now := time.Now().UTC()
	if updateErr := n.databaseService.SetLastBackupTime(databaseID, now); updateErr != nil {
		n.logger.Error(
			"Failed to update database last backup time",
			"databaseId",
			databaseID,
			"error",
			updateErr,
		)
	}

	if backup.Status != backups_core.BackupStatusCompleted && !isCallNotifier {
		return
	}

	n.SendBackupNotification(
		backupConfig,
		backup,
		backups_config.NotificationBackupSuccess,
		nil,
	)
}

func (n *BackuperNode) SendBackupNotification(
	backupConfig *backups_config.BackupConfig,
	backup *backups_core.Backup,
	notificationType backups_config.BackupNotificationType,
	errorMessage *string,
) {
	database, err := n.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil {
		return
	}

	workspace, err := n.workspaceService.GetWorkspaceByID(*database.WorkspaceID)
	if err != nil {
		return
	}

	for _, notifier := range database.Notifiers {
		if !slices.Contains(
			backupConfig.SendNotificationsOn,
			notificationType,
		) {
			continue
		}

		title := ""
		switch notificationType {
		case backups_config.NotificationBackupFailed:
			title = fmt.Sprintf(
				"❌ Backup failed for database \"%s\" (workspace \"%s\")",
				database.Name,
				workspace.Name,
			)
		case backups_config.NotificationBackupSuccess:
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

		n.notificationSender.SendNotification(
			&notifier,
			title,
			message,
		)
	}
}

func (n *BackuperNode) sendHeartbeat(backupNode *task_registry.TaskNode) {
	n.lastHeartbeat = time.Now().UTC()
	if err := n.tasksRegistry.HearthbeatNodeInRegistry(time.Now().UTC(), *backupNode); err != nil {
		n.logger.Error("Failed to send heartbeat", "error", err)
	}
}
