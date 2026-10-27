package usecases

import (
	"context"
	"errors"
	"io"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	restores_core "databasus-backend/internal/features/restores/core"
)

type postgresqlRestoreTarget struct {
	source *pglogical.Source
	spec   pglogical.RestoreSpec
}

func (uc *RestoreBackupUsecase) newPostgresqlRestoreTarget(run restoreRun) (*postgresqlRestoreTarget, error) {
	if run.source.PostgresqlLogical == nil {
		return nil, restores_core.ErrRestoreTypeNotSupported
	}

	targetDatabase := run.target.PostgresqlLogicalDatabase
	if targetDatabase == nil {
		return nil, errors.New("postgresql configuration is required for restore")
	}

	return &postgresqlRestoreTarget{
		source: pglogical.NewSource(targetDatabase.GetEngineSettings()),
		spec: pglogical.RestoreSpec{
			ClientTools:         pgshared.ClientTools{Root: uc.clientToolsDir},
			Version:             targetDatabase.Version,
			SourceDatabaseName:  run.source.GetDatabaseName(),
			ParallelJobs:        targetDatabase.CpuCount,
			IsTimescaleDB:       run.backup.TimescaledbVersion != "",
			IsRestoreOwnership:  targetDatabase.IsRestoreOwnership,
			IsRestorePrivileges: targetDatabase.IsRestorePrivileges,
			IsExcludeExtensions: run.options.IsExcludeExtensions,
			IsSkipUserMappings:  run.options.IsSkipUserMappings,
			TempDir:             uc.tempDir,
			Logger:              run.logger,
		},
	}, nil
}

func (t *postgresqlRestoreTarget) GetStoredCompression() storedCompression {
	return storedCompressionNone
}

func (t *postgresqlRestoreTarget) Restore(ctx context.Context, input io.Reader) error {
	return t.source.Restore(ctx, t.spec, input)
}
