package usecases

import (
	"context"
	"errors"
	"io"

	mysql_engine "databasus-backend/internal/engines/mysql"
	restores_core "databasus-backend/internal/features/restores/core"
)

type mysqlRestoreTarget struct {
	source *mysql_engine.Source
	spec   mysql_engine.RestoreSpec
}

func (uc *RestoreBackupUsecase) newMysqlRestoreTarget(run restoreRun) (*mysqlRestoreTarget, error) {
	if run.source.Mysql == nil {
		return nil, restores_core.ErrRestoreTypeNotSupported
	}

	targetDatabase := run.target.MysqlDatabase
	if targetDatabase == nil {
		return nil, errors.New("mysql configuration is required for restore")
	}

	return &mysqlRestoreTarget{
		source: mysql_engine.NewSource(
			targetDatabase.GetEngineSettings(), mysql_engine.ClientTools{Root: uc.clientToolsDir},
		),
		spec: mysql_engine.RestoreSpec{
			Version:            targetDatabase.Version,
			SourceDatabaseName: run.source.GetDatabaseName(),
			Logger:             run.logger,
		},
	}, nil
}

func (t *mysqlRestoreTarget) GetStoredCompression() storedCompression {
	return storedCompressionZstd
}

func (t *mysqlRestoreTarget) Restore(ctx context.Context, input io.Reader) error {
	return t.source.Restore(ctx, t.spec, input)
}
