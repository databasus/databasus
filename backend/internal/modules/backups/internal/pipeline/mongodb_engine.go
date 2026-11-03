package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/modules/databases"
)

type mongodbEngine struct {
	databases      connectionSettingsReader
	clientToolsDir string
}

func (e *mongodbEngine) GetStoredCompression() storedCompression {
	return getStoredCompressionNone()
}

func (e *mongodbEngine) GetFileExtension() string {
	return ".archive"
}

func (e *mongodbEngine) OpenDumpSource(
	ctx context.Context,
	databaseID uuid.UUID,
	logger *slog.Logger,
) (logicalDumpSource, error) {
	logger.InfoContext(ctx, "creating mongodb backup via mongodump")

	connectionSettings, err := e.databases.GetMongodbConnectionSettings(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	return openMongodbDumpSource(ctx, connectionSettings, mongodb_engine.ClientTools{Root: e.clientToolsDir}, logger)
}

func (e *mongodbEngine) OpenRestoreTarget(ctx context.Context, spec restoreTargetSpec) (logicalRestoreSink, error) {
	target, isOfEngine := spec.target.(MongodbRestoreTarget)
	if !isOfEngine {
		return nil, ErrTargetOfAnotherEngine
	}

	sourceSettings, err := e.databases.GetMongodbConnectionSettings(ctx, spec.sourceDatabaseID)
	if err != nil {
		return nil, err
	}

	tunnel, err := mongodb_engine.NewSource(target.Settings).OpenTunnel(ctx, spec.logger)
	if err != nil {
		return nil, &TargetConnectionError{Err: err}
	}

	detectCtx, cancelDetect := context.WithTimeout(ctx, targetVersionDetectionTimeout)
	defer cancelDetect()

	targetVersion, err := tunnel.GetSource().DetectVersion(detectCtx, spec.logger)
	if err != nil {
		tunnel.Close()

		return nil, &TargetConnectionError{Err: err}
	}

	isDowngrade, err := mongodb_engine.IsBackupVersionHigherThanRestoreVersion(sourceSettings.Version, targetVersion)
	if err != nil {
		tunnel.Close()

		return nil, err
	}

	if isDowngrade {
		tunnel.Close()

		return nil, fmt.Errorf("%w: the backup is of MongoDB %s and the target runs MongoDB %s",
			ErrTargetVersionTooOld, sourceSettings.Version, targetVersion)
	}

	return &mongodbRestoreSink{
		tunnel: tunnel,
		spec: mongodb_engine.RestoreSpec{
			ClientTools:        mongodb_engine.ClientTools{Root: e.clientToolsDir},
			SourceDatabaseName: sourceSettings.Engine.Database,
			InsertionWorkers:   target.CPUCount,
			Logger:             spec.logger,
		},
	}, nil
}

type mongodbDumpSource struct {
	tunnel *mongodb_engine.Tunnel
	spec   mongodb_engine.DumpSpec
	logger *slog.Logger
}

func openMongodbDumpSource(
	ctx context.Context,
	connectionSettings databases.MongodbConnectionSettings,
	clientTools mongodb_engine.ClientTools,
	logger *slog.Logger,
) (*mongodbDumpSource, error) {
	tunnel, err := mongodb_engine.NewSource(connectionSettings.Engine).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &mongodbDumpSource{
		tunnel: tunnel,
		spec: mongodb_engine.DumpSpec{
			ClientTools:         clientTools,
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
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", slog.Any("error", err))

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mongodbDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mongodbDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mongodbDumpSource) Close() {
	s.tunnel.Close()
}

type mongodbRestoreSink struct {
	tunnel *mongodb_engine.Tunnel
	spec   mongodb_engine.RestoreSpec
}

func (s *mongodbRestoreSink) IsArchiveFileNeeded() bool {
	return false
}

func (s *mongodbRestoreSink) Restore(ctx context.Context, input io.Reader) error {
	return s.tunnel.GetSource().Restore(ctx, s.spec, input)
}

func (s *mongodbRestoreSink) Close() {
	s.tunnel.Close()
}
