package httpapi

import "databasus-backend/internal/modules/backups/internal/restore"

func toRestoreBody(readRestore restore.Restore) Restore {
	return Restore{
		Id:          readRestore.ID,
		BackupId:    readRestore.BackupID,
		Status:      RestoreStatus(readRestore.Status),
		FailMessage: readRestore.FailMessage,
		DurationMs:  readRestore.RestoreDurationMs,
		CreatedAt:   readRestore.CreatedAt,
	}
}
