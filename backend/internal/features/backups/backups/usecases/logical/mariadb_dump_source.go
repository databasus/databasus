package usecases_logical

import (
	"context"
	"io"
	"log/slog"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/modules/databases"
)

type mariadbDumpSource struct {
	tunnel *mariadb_engine.Tunnel
	spec   mariadb_engine.DumpSpec
	logger *slog.Logger
}

func (uc *CreateBackupUsecase) openMariadbDumpSource(
	ctx context.Context,
	database databases.Database,
	logger *slog.Logger,
) (*mariadbDumpSource, error) {
	logger.InfoContext(ctx, "creating mariadb backup via mariadb-dump")

	connectionSettings, err := uc.getDatabases().GetMariadbConnectionSettings(ctx, database.ID)
	if err != nil {
		return nil, err
	}

	clientTools := mariadb_engine.ClientTools{Root: uc.clientToolsDir}

	tunnel, err := mariadb_engine.NewSource(connectionSettings.Engine, clientTools).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &mariadbDumpSource{
		tunnel: tunnel,
		spec: mariadb_engine.DumpSpec{
			Version:         connectionSettings.Version,
			ExcludeTables:   connectionSettings.ExcludeTables,
			Privileges:      connectionSettings.Privileges,
			IsExcludeEvents: connectionSettings.IsExcludeEvents,
			Logger:          logger,
		},
		logger: logger,
	}, nil
}

func (s *mariadbDumpSource) GetRawSizeMb(ctx context.Context) (float64, bool) {
	rawSizeMb, err := s.tunnel.GetSource().GetRawSizeMb(ctx, s.logger)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mariadbDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mariadbDumpSource) GetStoredCompression() storedCompression {
	return newZstdStoredCompression(mariadb_engine.StoredZstdCompressionLevel)
}

func (s *mariadbDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mariadbDumpSource) Close() {
	s.tunnel.Close()
}
