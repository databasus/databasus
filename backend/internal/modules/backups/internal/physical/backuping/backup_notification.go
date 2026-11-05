package backuping

import (
	"fmt"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
)

type backupNotificationSpec struct {
	Database      databases.Database
	WorkspaceName string
	Backup        *catalog.Backup
	BackupResult  Result
}

func classifyBackupNotificationSpec(spec backupNotificationSpec) (config.NotificationTrigger, string, string) {
	kind := "FULL"
	if spec.Backup.Kind == catalog.BackupKindIncremental {
		kind = "INCR"
	}
	switch spec.Backup.Status {
	case catalog.BackupStatusCompleted:
		return config.NotificationTriggerBackupSuccess,
			fmt.Sprintf("Physical %s completed for %q (workspace %q)", kind, spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				spec.Backup.ID, spec.BackupResult.BackupSizeMb, spec.BackupResult.BackupDurationMs)

	case catalog.BackupStatusError:
		return config.NotificationTriggerBackupFailed,
			fmt.Sprintf("Physical %s failed for %q (workspace %q)", kind, spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.Backup.ID, reasonOrEmpty(spec.Backup.ErrorReason), spec.BackupResult.ErrorMessage)

	case catalog.BackupStatusChainBroken:
		return config.NotificationTriggerChainBroken,
			fmt.Sprintf("Physical %s chain-broken for %q (workspace %q)", kind, spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.Backup.ID, reasonOrEmpty(spec.Backup.ErrorReason), spec.BackupResult.ErrorMessage)
	case catalog.BackupStatusInProgress, catalog.BackupStatusCanceled:
		return "", "", ""
	}

	return "", "", ""
}
