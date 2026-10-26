package usecases

import (
	"context"
	"errors"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	usecases_mariadb "databasus-backend/internal/features/restores/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/restores/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/restores/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/restores/usecases/postgresql"
	storage_files "databasus-backend/internal/features/storages/files"
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
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backup *backups_core_logical.LogicalBackup,
	storageReader storage_files.StorageReader,
	options restores_core.RestoreOptions,
) error {
	switch originalDB.Type {
	case databases.DatabaseTypePostgresLogical:
		return uc.restorePostgresqlBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storageReader,
			options,
		)
	case databases.DatabaseTypeMysql:
		return uc.restoreMysqlBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	case databases.DatabaseTypeMariadb:
		return uc.restoreMariadbBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	case databases.DatabaseTypeMongodb:
		return uc.restoreMongodbBackupUsecase.Execute(
			ctx,
			originalDB,
			restoringToDB,
			backupConfig,
			restore,
			backup,
			storageReader,
		)
	default:
		return errors.New("database type not supported")
	}
}
