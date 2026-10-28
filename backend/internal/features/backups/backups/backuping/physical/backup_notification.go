package backuping_physical

import (
	"fmt"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

type backupNotificationSpec struct {
	Database          databases.Database
	WorkspaceName     string
	FullBackup        *physical_models.PhysicalFullBackup
	IncrementalBackup *physical_models.PhysicalIncrementalBackup
	BackupResult      postgresql_executor.PhysicalBackupResult
}

func classifyFullNotification(spec backupNotificationSpec) (backups.NotificationTrigger, string, string) {
	switch spec.FullBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups.NotificationTriggerBackupSuccess,
			fmt.Sprintf("Physical FULL completed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				spec.FullBackup.ID, spec.BackupResult.BackupSizeMb, spec.BackupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups.NotificationTriggerBackupFailed,
			fmt.Sprintf("Physical FULL failed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.FullBackup.ID, reasonOrEmpty(spec.FullBackup.ErrorReason), spec.BackupResult.ErrorMessage)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups.NotificationTriggerChainBroken,
			fmt.Sprintf("Physical FULL chain-broken for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				spec.FullBackup.ID, reasonOrEmpty(spec.FullBackup.ErrorReason), spec.BackupResult.ErrorMessage)
	}

	return "", "", ""
}

func classifyIncrNotification(spec backupNotificationSpec) (backups.NotificationTrigger, string, string) {
	switch spec.IncrementalBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups.NotificationTriggerBackupSuccess,
			fmt.Sprintf("Physical INCR completed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				spec.IncrementalBackup.ID, spec.BackupResult.BackupSizeMb, spec.BackupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups.NotificationTriggerBackupFailed,
			fmt.Sprintf("Physical INCR failed for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf(
				"backup_id=%s reason=%s message=%s",
				spec.IncrementalBackup.ID,
				reasonOrEmpty(spec.IncrementalBackup.ErrorReason),
				spec.BackupResult.ErrorMessage,
			)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups.NotificationTriggerChainBroken,
			fmt.Sprintf("Physical INCR chain-broken for %q (workspace %q)", spec.Database.Name, spec.WorkspaceName),
			fmt.Sprintf(
				"backup_id=%s reason=%s message=%s",
				spec.IncrementalBackup.ID,
				reasonOrEmpty(spec.IncrementalBackup.ErrorReason),
				spec.BackupResult.ErrorMessage,
			)
	}

	return "", "", ""
}
