package usecases_logical

import (
	"context"
	"io"
	"log/slog"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

type postgresqlDumpSource struct {
	tunnel     *pglogical.Tunnel
	dumpFilter pglogical.DumpFilter
	spec       pglogical.DumpSpec
	logger     *slog.Logger
}

func (uc *CreateBackupUsecase) openPostgresqlDumpSource(
	ctx context.Context,
	database databases.Database,
	logger *slog.Logger,
) (*postgresqlDumpSource, error) {
	logger.InfoContext(ctx, "creating postgresql backup via pg_dump custom format")

	connectionSettings, err := uc.getDatabases().GetPostgresqlLogicalConnectionSettings(ctx, database.ID)
	if err != nil {
		return nil, err
	}

	tunnel, err := pglogical.NewSource(connectionSettings.Engine).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &postgresqlDumpSource{
		tunnel:     tunnel,
		dumpFilter: connectionSettings.GetDumpFilter(),
		spec: pglogical.DumpSpec{
			ClientTools:      pgshared.ClientTools{Root: uc.clientToolsDir},
			Version:          connectionSettings.Version,
			IncludeSchemas:   connectionSettings.IncludeSchemas,
			ExcludeTables:    connectionSettings.ExcludeTables,
			ExcludeTableData: connectionSettings.ExcludeTableData,
			Logger:           logger,
		},
		logger: logger,
	}, nil
}

func (s *postgresqlDumpSource) GetRawSizeMb(ctx context.Context) (float64, bool) {
	rawSizeMb, err := s.tunnel.GetSource().GetDumpedSizeMb(ctx, s.logger, s.dumpFilter)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)

		return 0, false
	}

	return rawSizeMb, true
}

func (s *postgresqlDumpSource) GetTimescaleDBVersion(ctx context.Context) string {
	timescaleDBVersion, err := s.tunnel.GetSource().GetTimescaleDBVersion(ctx, s.logger)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to detect timescaledb extension before backup", "error", err)

		return ""
	}

	return timescaleDBVersion
}

func (s *postgresqlDumpSource) GetStoredCompression() storedCompression {
	return storedCompressionNone
}

func (s *postgresqlDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *postgresqlDumpSource) Close() {
	s.tunnel.Close()
}
