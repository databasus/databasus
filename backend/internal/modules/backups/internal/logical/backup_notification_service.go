package logical

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
)

const megabytesInGigabyte = 1024

// notifyBackupOutcome sends the notification of a completed or failed backup to the database's
// notifiers when the config names the trigger.
func (s *Service) notifyBackupOutcome(
	ctx context.Context,
	backup LogicalBackup,
	backupDatabase databases.Database,
	backupConfig config.LogicalBackupConfig,
	trigger config.NotificationTrigger,
) {
	if !backupConfig.IsNotifyingOn(trigger) {
		return
	}

	workspaceName, err := s.getWorkspaceName(ctx, backupDatabase.WorkspaceID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the workspace for the backup notification", slog.Any("error", err))

		return
	}

	s.notifiers.Notify(
		ctx,
		backupDatabase.NotifierIDs,
		buildBackupNotification(backup, backupDatabase.Name, workspaceName),
	)
}

func (s *Service) getWorkspaceName(ctx context.Context, workspaceID uuid.UUID) (string, error) {
	foundWorkspaces, err := s.workspaces.GetWorkspaces(ctx, []uuid.UUID{workspaceID})
	if err != nil {
		return "", err
	}

	if len(foundWorkspaces) == 0 {
		return "", fmt.Errorf("%w: %s", errWorkspaceNotFound, workspaceID)
	}

	return foundWorkspaces[0].Name, nil
}

func buildBackupNotification(backup LogicalBackup, databaseName, workspaceName string) notifiers.Notification {
	if backup.Status == StatusFailed {
		failMessage := ""
		if backup.FailMessage != nil {
			failMessage = *backup.FailMessage
		}

		return notifiers.Notification{
			Category: notifiers.CategoryBackupFailed,
			Heading: fmt.Sprintf(
				"❌ Backup failed for database \"%s\" (workspace \"%s\")",
				databaseName,
				workspaceName,
			),
			Message: failMessage,
		}
	}

	return notifiers.Notification{
		Category: notifiers.CategoryBackupSuccess,
		Heading:  fmt.Sprintf("✅ Backup completed for database \"%s\" (workspace \"%s\")", databaseName, workspaceName),
		Message: fmt.Sprintf("Backup completed successfully in %s.\nCompressed backup size: %s",
			formatDuration(backup.BackupDurationMs), formatSize(backup.BackupSizeMb)),
	}
}

func formatDuration(durationMs int64) string {
	minutes := durationMs / (1000 * 60)
	seconds := (durationMs % (1000 * 60)) / 1000

	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

func formatSize(sizeMb float64) string {
	if sizeMb < megabytesInGigabyte {
		return fmt.Sprintf("%.2f MB", sizeMb)
	}

	return fmt.Sprintf("%.2f GB", sizeMb/megabytesInGigabyte)
}
