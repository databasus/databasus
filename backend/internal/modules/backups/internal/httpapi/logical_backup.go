package httpapi

import "databasus-backend/internal/modules/backups/internal/logical"

func toLogicalBackupBody(backup logical.LogicalBackup) LogicalBackup {
	return LogicalBackup{
		Id:                 backup.ID,
		DatabaseId:         backup.DatabaseID,
		StorageId:          backup.StorageID,
		Status:             LogicalBackupStatus(backup.Status),
		FailMessage:        backup.FailMessage,
		SizeMb:             backup.BackupSizeMb,
		RawDatabaseSizeMb:  backup.BackupRawDbSizeMb,
		DurationMs:         backup.BackupDurationMs,
		Encryption:         BackupEncryption(backup.Encryption),
		VerificationStatus: LogicalBackupVerificationStatus(backup.RestoreVerificationStatus),
		CreatedAt:          backup.CreatedAt,
	}
}
