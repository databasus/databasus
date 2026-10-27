package restores_core

import (
	"context"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/storages"
)

type RestoreBackupUsecase interface {
	Execute(
		ctx context.Context,
		backupConfig backups.LogicalBackupConfig,
		restore Restore,
		source SourceDatabase,
		target RestoreBackupRequest,
		backup *backups_core_logical.LogicalBackup,
		storageReader storages.StorageReader,
		options RestoreOptions,
	) error
}
