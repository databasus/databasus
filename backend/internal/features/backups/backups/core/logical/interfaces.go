package backups_core_logical

import (
	"context"
	"io"

	"github.com/google/uuid"

	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
)

type NotificationSender interface {
	Notify(
		ctx context.Context,
		notifierIDs []uuid.UUID,
		notification notifiers.Notification,
	)
}

// BackupFileStore is the only way a backup writes to storage. It hands back a
// receipt instead of a stored file, because the file belongs to nobody until the
// transaction that publishes the backup claims it.
type BackupFileStore interface {
	WriteFile(
		ctx context.Context,
		reference storages.StoredFileReference,
		file io.Reader,
	) (storages.WriteReceipt, error)
	StartBackgroundWrite(ctx context.Context, spec storages.BackgroundWriteSpec) storages.BackgroundWrite
}

type CreateBackupUsecase interface {
	Execute(
		ctx context.Context,
		backup *LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		database databases.Database,
		fileStore BackupFileStore,
		backupProgressListener func(completedMBs float64),
	) (*BackupArtifacts, error)
}

type BackupRemoveListener interface {
	OnBeforeBackupRemove(backup *LogicalBackup) error
}

type BackupCompletionListener interface {
	OnBackupCompleted(backupID uuid.UUID)
}
