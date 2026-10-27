package usecases

import (
	"context"
	"errors"
	"io"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	restores_core "databasus-backend/internal/features/restores/core"
)

type mariadbRestoreTarget struct {
	source *mariadb_engine.Source
	spec   mariadb_engine.RestoreSpec
}

func (uc *RestoreBackupUsecase) newMariadbRestoreTarget(run restoreRun) (*mariadbRestoreTarget, error) {
	if run.source.Mariadb == nil {
		return nil, restores_core.ErrRestoreTypeNotSupported
	}

	targetDatabase := run.target.MariadbDatabase
	if targetDatabase == nil {
		return nil, errors.New("mariadb configuration is required for restore")
	}

	return &mariadbRestoreTarget{
		source: mariadb_engine.NewSource(
			targetDatabase.GetEngineSettings(), mariadb_engine.ClientTools{Root: uc.clientToolsDir},
		),
		spec: mariadb_engine.RestoreSpec{
			Version:             targetDatabase.Version,
			SourceDatabaseName:  run.source.GetDatabaseName(),
			IsSkipGaleraDisable: targetDatabase.IsSkipGaleraDisable,
			Logger:              run.logger,
		},
	}, nil
}

func (t *mariadbRestoreTarget) GetStoredCompression() storedCompression {
	return storedCompressionZstd
}

func (t *mariadbRestoreTarget) Restore(ctx context.Context, input io.Reader) error {
	return t.source.Restore(ctx, t.spec, input)
}
