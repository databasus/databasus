package backuping

import (
	"fmt"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
)

type backupNotificationSpec struct {
	Database          databases.Database
	WorkspaceName     string
	FullBackup        *catalog.FullBackup
	IncrementalBackup *catalog.IncrementalBackup
	BackupResult      Result
}

func classifyFullNotification(spec backupNotificationSpec) (config.NotificationTrigger, string, string) {
	switch spec.FullBackup.Status {
	case catalog.BackupStatusCompleted:
		return config.NotificationTriggerBackupSuccess,
			fmt.Sprintf("Physical FULL completed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				spec.FullBackup.ID, spec.BackupResult.BackupSizeMb, spec.BackupResult.BackupDurationMs)

	case catalog.BackupStatusError:
		return config.NotificationTriggerBackupFailed,
			fmt.Sprintf("Physical FULL failed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.FullBackup.ID, reasonOrEmpty(spec.FullBackup.ErrorReason), spec.BackupResult.ErrorMessage)

	case catalog.BackupStatusChainBroken:
		return config.NotificationTriggerChainBroken,
			fmt.Sprintf("Physical FULL chain-broken for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.FullBackup.ID, reasonOrEmpty(spec.FullBackup.ErrorReason), spec.BackupResult.ErrorMessage)
	case catalog.BackupStatusInProgress, catalog.BackupStatusCanceled:
		return "", "", ""
	}

	return "", "", ""
}

func classifyIncrNotification(spec backupNotificationSpec) (config.NotificationTrigger, string, string) {
	switch spec.IncrementalBackup.Status {
	case catalog.BackupStatusCompleted:
		return config.NotificationTriggerBackupSuccess,
			fmt.Sprintf("Physical INCR completed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				spec.IncrementalBackup.ID, spec.BackupResult.BackupSizeMb, spec.BackupResult.BackupDurationMs)

	case catalog.BackupStatusError:
		return config.NotificationTriggerBackupFailed,
			fmt.Sprintf("Physical INCR failed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf(
				"backup_id=%s reason=%s message=%s",
				spec.IncrementalBackup.ID,
				reasonOrEmpty(spec.IncrementalBackup.ErrorReason),
				spec.BackupResult.ErrorMessage,
			)

	case catalog.BackupStatusChainBroken:
		return config.NotificationTriggerChainBroken,
			fmt.Sprintf("Physical INCR chain-broken for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf(
				"backup_id=%s reason=%s message=%s",
				spec.IncrementalBackup.ID,
				reasonOrEmpty(spec.IncrementalBackup.ErrorReason),
				spec.BackupResult.ErrorMessage,
			)
	case catalog.BackupStatusInProgress, catalog.BackupStatusCanceled:
		return "", "", ""
	}

	return "", "", ""
}
