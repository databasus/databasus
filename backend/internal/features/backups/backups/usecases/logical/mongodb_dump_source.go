package usecases_logical

import (
	"context"
	"io"
	"log/slog"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/modules/databases"
)

type mongodbDumpSource struct {
	tunnel *mongodb_engine.Tunnel
	spec   mongodb_engine.DumpSpec
	logger *slog.Logger
}

func (uc *CreateBackupUsecase) openMongodbDumpSource(
	ctx context.Context,
	database databases.Database,
	logger *slog.Logger,
) (*mongodbDumpSource, error) {
	logger.InfoContext(ctx, "creating mongodb backup via mongodump")

	connectionSettings, err := uc.getDatabases().GetMongodbConnectionSettings(ctx, database.ID)
	if err != nil {
		return nil, err
	}

	tunnel, err := mongodb_engine.NewSource(connectionSettings.Engine).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &mongodbDumpSource{
		tunnel: tunnel,
		spec: mongodb_engine.DumpSpec{
			ClientTools:         mongodb_engine.ClientTools{Root: uc.clientToolsDir},
			ExcludeCollections:  connectionSettings.ExcludeCollections,
			ParallelCollections: connectionSettings.CPUCount,
			Logger:              logger,
		},
		logger: logger,
	}, nil
}

func (s *mongodbDumpSource) GetRawSizeMb(ctx context.Context) (float64, bool) {
	rawSizeMb, err := s.tunnel.GetSource().GetRawSizeMb(ctx, s.logger)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mongodbDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mongodbDumpSource) GetStoredCompression() storedCompression {
	return storedCompressionNone
}

func (s *mongodbDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mongodbDumpSource) Close() {
	s.tunnel.Close()
}
