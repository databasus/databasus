package usecases_logical

import (
	"context"
	"io"
	"log/slog"

	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/modules/databases"
)

type mysqlDumpSource struct {
	tunnel *mysql_engine.Tunnel
	spec   mysql_engine.DumpSpec
	logger *slog.Logger
}

func (uc *CreateBackupUsecase) openMysqlDumpSource(
	ctx context.Context,
	database databases.Database,
	logger *slog.Logger,
) (*mysqlDumpSource, error) {
	logger.InfoContext(ctx, "creating mysql backup via mysqldump")

	connectionSettings, err := uc.getDatabases().GetMysqlConnectionSettings(ctx, database.ID)
	if err != nil {
		return nil, err
	}

	clientTools := mysql_engine.ClientTools{Root: uc.clientToolsDir}

	tunnel, err := mysql_engine.NewSource(connectionSettings.Engine, clientTools).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &mysqlDumpSource{
		tunnel: tunnel,
		spec: mysql_engine.DumpSpec{
			Version:       connectionSettings.Version,
			ExcludeTables: connectionSettings.ExcludeTables,
			Privileges:    connectionSettings.Privileges,
			Logger:        logger,
		},
		logger: logger,
	}, nil
}

func (s *mysqlDumpSource) GetRawSizeMb(ctx context.Context) (float64, bool) {
	rawSizeMb, err := s.tunnel.GetSource().GetRawSizeMb(ctx, s.logger, s.spec.Version)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mysqlDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mysqlDumpSource) GetStoredCompression() storedCompression {
	return newZstdStoredCompression(mysql_engine.StoredZstdCompressionLevel)
}

func (s *mysqlDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mysqlDumpSource) Close() {
	s.tunnel.Close()
}
