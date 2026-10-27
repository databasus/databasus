package usecases

import (
	"context"
	"log/slog"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type RestoreBackupUsecase struct {
	logger         *slog.Logger
	masterKey      *crypto.MasterKey
	clientToolsDir string
	// tempDir holds the archive of a PostgreSQL restore that needs it as a file.
	tempDir string
}

// restoreRun is one restore as the restorer requested it.
type restoreRun struct {
	source        restores_core.SourceDatabase
	target        restores_core.RestoreBackupRequest
	backup        *backups_core_logical.LogicalBackup
	storageReader storages.StorageReader
	options       restores_core.RestoreOptions
	logger        *slog.Logger
}

func (uc *RestoreBackupUsecase) Execute(
	ctx context.Context,
	_ backups.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	run := restoreRun{
		source:        source,
		target:        target,
		backup:        backup,
		storageReader: storageReader,
		options:       options,
		logger:        uc.logger.With("restore_id", restore.ID, "backup_id", backup.ID),
	}

	restoreTarget, err := uc.newRestoreTarget(run)
	if err != nil {
		return err
	}

	run.logger.InfoContext(ctx, "restoring "+string(source.Database.Type)+" backup")

	return uc.restoreFromStorage(ctx, run, restoreTarget)
}

// newRestoreTarget picks the engine by the database type until the backups module replaces this
// switch with its registry of engines.
func (uc *RestoreBackupUsecase) newRestoreTarget(run restoreRun) (logicalRestoreTarget, error) {
	switch run.source.Database.Type {
	case databases.TypePostgresqlLogical:
		return uc.newPostgresqlRestoreTarget(run)
	case databases.TypeMysql:
		return uc.newMysqlRestoreTarget(run)
	case databases.TypeMariadb:
		return uc.newMariadbRestoreTarget(run)
	case databases.TypeMongodb:
		return uc.newMongodbRestoreTarget(run)
	default:
		return nil, restores_core.ErrRestoreTypeNotSupported
	}
}
