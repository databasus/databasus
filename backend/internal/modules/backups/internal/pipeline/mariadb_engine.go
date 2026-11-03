package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/modules/databases"
)

type mariadbEngine struct {
	databases      connectionSettingsReader
	clientToolsDir string
}

func (e *mariadbEngine) GetStoredCompression() storedCompression {
	return newZstdStoredCompression(mariadb_engine.StoredZstdCompressionLevel)
}

func (e *mariadbEngine) GetFileExtension() string {
	return ".sql.zst"
}

func (e *mariadbEngine) OpenDumpSource(
	ctx context.Context,
	databaseID uuid.UUID,
	logger *slog.Logger,
) (logicalDumpSource, error) {
	logger.InfoContext(ctx, "creating mariadb backup via mariadb-dump")

	connectionSettings, err := e.databases.GetMariadbConnectionSettings(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	return openMariadbDumpSource(ctx, connectionSettings, mariadb_engine.ClientTools{Root: e.clientToolsDir}, logger)
}

func (e *mariadbEngine) OpenRestoreTarget(ctx context.Context, spec restoreTargetSpec) (logicalRestoreSink, error) {
	target, isOfEngine := spec.target.(MariadbRestoreTarget)
	if !isOfEngine {
		return nil, ErrTargetOfAnotherEngine
	}

	sourceSettings, err := e.databases.GetMariadbConnectionSettings(ctx, spec.sourceDatabaseID)
	if err != nil {
		return nil, err
	}

	clientTools := mariadb_engine.ClientTools{Root: e.clientToolsDir}

	tunnel, err := mariadb_engine.NewSource(target.Settings, clientTools).OpenTunnel(ctx, spec.logger)
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

	isDowngrade, err := mariadb_engine.IsBackupVersionHigherThanRestoreVersion(sourceSettings.Version, targetVersion)
	if err != nil {
		tunnel.Close()

		return nil, err
	}

	if isDowngrade {
		tunnel.Close()

		return nil, fmt.Errorf("%w: the backup is of MariaDB %s and the target runs MariaDB %s",
			ErrTargetVersionTooOld, sourceSettings.Version, targetVersion)
	}

	return &mariadbRestoreSink{
		tunnel: tunnel,
		spec: mariadb_engine.RestoreSpec{
			Version:             targetVersion,
			SourceDatabaseName:  sourceSettings.Engine.Database,
			IsSkipGaleraDisable: target.IsSkipGaleraDisable,
			Logger:              spec.logger,
		},
	}, nil
}

type mariadbDumpSource struct {
	tunnel *mariadb_engine.Tunnel
	spec   mariadb_engine.DumpSpec
	logger *slog.Logger
}

func openMariadbDumpSource(
	ctx context.Context,
	connectionSettings databases.MariadbConnectionSettings,
	clientTools mariadb_engine.ClientTools,
	logger *slog.Logger,
) (*mariadbDumpSource, error) {
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
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", slog.Any("error", err))

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mariadbDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mariadbDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mariadbDumpSource) Close() {
	s.tunnel.Close()
}

type mariadbRestoreSink struct {
	tunnel *mariadb_engine.Tunnel
	spec   mariadb_engine.RestoreSpec
}

func (s *mariadbRestoreSink) IsArchiveFileNeeded() bool {
	return false
}

func (s *mariadbRestoreSink) Restore(ctx context.Context, input io.Reader) error {
	return s.tunnel.GetSource().Restore(ctx, s.spec, input)
}

func (s *mariadbRestoreSink) Close() {
	s.tunnel.Close()
}
