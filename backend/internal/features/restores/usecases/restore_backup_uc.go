package usecases

import (
	"context"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	usecases_mariadb "databasus-backend/internal/features/restores/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/restores/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/restores/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/restores/usecases/postgresql"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type RestoreBackupUsecase struct {
	restorePostgresqlBackupUsecase *usecases_postgresql.RestorePostgresqlBackupUsecase
	restoreMysqlBackupUsecase      *usecases_mysql.RestoreMysqlBackupUsecase
	restoreMariadbBackupUsecase    *usecases_mariadb.RestoreMariadbBackupUsecase
	restoreMongodbBackupUsecase    *usecases_mongodb.RestoreMongodbBackupUsecase
}

func (uc *RestoreBackupUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	switch source.Database.Type {
	case databases.TypePostgresqlLogical:
		return uc.restorePostgresqlBackupUsecase.Execute(
			ctx,
			source,
			target,
			backupConfig,
			restore,
			backup,
			storageReader,
			options,
		)
	case databases.TypeMysql:
		return uc.restoreMysqlBackupUsecase.Execute(
			ctx,
			source,
			target,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	case databases.TypeMariadb:
		return uc.restoreMariadbBackupUsecase.Execute(
			ctx,
			source,
			target,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	case databases.TypeMongodb:
		return uc.restoreMongodbBackupUsecase.Execute(
			ctx,
			source,
			target,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	default:
		return restores_core.ErrRestoreTypeNotSupported
	}
}
