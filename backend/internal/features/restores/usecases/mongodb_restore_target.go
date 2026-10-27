package usecases

import (
	"context"
	"errors"
	"io"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	restores_core "databasus-backend/internal/features/restores/core"
)

type mongodbRestoreTarget struct {
	source *mongodb_engine.Source
	spec   mongodb_engine.RestoreSpec
}

func (uc *RestoreBackupUsecase) newMongodbRestoreTarget(run restoreRun) (*mongodbRestoreTarget, error) {
	if run.source.Mongodb == nil {
		return nil, restores_core.ErrRestoreTypeNotSupported
	}

	targetDatabase := run.target.MongodbDatabase
	if targetDatabase == nil {
		return nil, errors.New("mongodb configuration is required for restore")
	}

	return &mongodbRestoreTarget{
		source: mongodb_engine.NewSource(targetDatabase.GetEngineSettings()),
		spec: mongodb_engine.RestoreSpec{
			ClientTools:        mongodb_engine.ClientTools{Root: uc.clientToolsDir},
			SourceDatabaseName: run.source.GetDatabaseName(),
			InsertionWorkers:   targetDatabase.CpuCount,
			Logger:             run.logger,
		},
	}, nil
}

func (t *mongodbRestoreTarget) GetStoredCompression() storedCompression {
	return storedCompressionNone
}

func (t *mongodbRestoreTarget) Restore(ctx context.Context, input io.Reader) error {
	return t.source.Restore(ctx, t.spec, input)
}
