package restores_core

import (
	"context"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	storage_files "databasus-backend/internal/features/storages/files"
)

type RestoreBackupUsecase interface {
	Execute(
		ctx context.Context,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		restore Restore,
		originalDB *databases.Database,
		restoringToDB *databases.Database,
		backup *backups_core_logical.LogicalBackup,
		storageReader storage_files.StorageReader,
		options RestoreOptions,
	) error
}
