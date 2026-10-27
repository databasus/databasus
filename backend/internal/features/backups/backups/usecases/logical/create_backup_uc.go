package usecases_logical

import (
	"context"
	"errors"
	"log/slog"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases"
)

type CreateBackupUsecase struct {
	logger    *slog.Logger
	masterKey *crypto.MasterKey
	// getDatabases is read per call: each test application installs its own module.
	getDatabases   func() databases.Module
	clientToolsDir string
}

func (uc *CreateBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	logger := uc.logger.With("database_id", database.ID, "storage_id", backup.StorageID, "backup_id", backup.ID)

	dumpSource, err := uc.openDumpSource(ctx, database, logger)
	if err != nil {
		return nil, err
	}
	defer dumpSource.Close()

	if rawSizeMb, isMeasured := dumpSource.GetRawSizeMb(ctx); isMeasured {
		backup.BackupRawDbSizeMb = rawSizeMb
	}

	backup.TimescaledbVersion = dumpSource.GetTimescaleDBVersion(ctx)

	return uc.streamDumpToStorage(ctx, backupStream{
		backup:           backup,
		backupConfig:     backupConfig,
		dumpSource:       dumpSource,
		fileStore:        fileStore,
		progressListener: backupProgressListener,
		logger:           logger,
	})
}

// openDumpSource picks the engine by the database type until the backups module replaces this
// switch with its registry of engines.
func (uc *CreateBackupUsecase) openDumpSource(
	ctx context.Context,
	database databases.Database,
	logger *slog.Logger,
) (logicalDumpSource, error) {
	switch database.Type {
	case databases.TypePostgresqlLogical:
		return uc.openPostgresqlDumpSource(ctx, database, logger)
	case databases.TypeMysql:
		return uc.openMysqlDumpSource(ctx, database, logger)
	case databases.TypeMariadb:
		return uc.openMariadbDumpSource(ctx, database, logger)
	case databases.TypeMongodb:
		return uc.openMongodbDumpSource(ctx, database, logger)
	default:
		return nil, errors.New("database type not supported")
	}
}
