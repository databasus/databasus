package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/modules/databases"
)

type mysqlEngine struct {
	databases      connectionSettingsReader
	clientToolsDir string
}

func (e *mysqlEngine) GetStoredCompression() storedCompression {
	return newZstdStoredCompression(mysql_engine.StoredZstdCompressionLevel)
}

func (e *mysqlEngine) GetFileExtension() string {
	return ".sql.zst"
}

func (e *mysqlEngine) OpenDumpSource(
	ctx context.Context,
	databaseID uuid.UUID,
	logger *slog.Logger,
) (logicalDumpSource, error) {
	logger.InfoContext(ctx, "creating mysql backup via mysqldump")

	connectionSettings, err := e.databases.GetMysqlConnectionSettings(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	return openMysqlDumpSource(ctx, connectionSettings, mysql_engine.ClientTools{Root: e.clientToolsDir}, logger)
}

func (e *mysqlEngine) OpenRestoreTarget(ctx context.Context, spec restoreTargetSpec) (logicalRestoreSink, error) {
	target, isOfEngine := spec.target.(MysqlRestoreTarget)
	if !isOfEngine {
		return nil, ErrTargetOfAnotherEngine
	}

	sourceSettings, err := e.databases.GetMysqlConnectionSettings(ctx, spec.sourceDatabaseID)
	if err != nil {
		return nil, err
	}

	clientTools := mysql_engine.ClientTools{Root: e.clientToolsDir}

	tunnel, err := mysql_engine.NewSource(target.Settings, clientTools).OpenTunnel(ctx, spec.logger)
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

	isDowngrade, err := mysql_engine.IsBackupVersionHigherThanRestoreVersion(sourceSettings.Version, targetVersion)
	if err != nil {
		tunnel.Close()

		return nil, err
	}

	if isDowngrade {
		tunnel.Close()

		return nil, fmt.Errorf("%w: the backup is of MySQL %s and the target runs MySQL %s",
			ErrTargetVersionTooOld, sourceSettings.Version, targetVersion)
	}

	return &mysqlRestoreSink{
		tunnel: tunnel,
		spec: mysql_engine.RestoreSpec{
			Version:            targetVersion,
			SourceDatabaseName: sourceSettings.Engine.Database,
			Logger:             spec.logger,
		},
	}, nil
}

type mysqlDumpSource struct {
	tunnel *mysql_engine.Tunnel
	spec   mysql_engine.DumpSpec
	logger *slog.Logger
}

func openMysqlDumpSource(
	ctx context.Context,
	connectionSettings databases.MysqlConnectionSettings,
	clientTools mysql_engine.ClientTools,
	logger *slog.Logger,
) (*mysqlDumpSource, error) {
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
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", slog.Any("error", err))

		return 0, false
	}

	return rawSizeMb, true
}

func (s *mysqlDumpSource) GetTimescaleDBVersion(context.Context) string {
	return ""
}

func (s *mysqlDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *mysqlDumpSource) Close() {
	s.tunnel.Close()
}

type mysqlRestoreSink struct {
	tunnel *mysql_engine.Tunnel
	spec   mysql_engine.RestoreSpec
}

func (s *mysqlRestoreSink) IsArchiveFileNeeded() bool {
	return false
}

func (s *mysqlRestoreSink) Restore(ctx context.Context, input io.Reader) error {
	return s.tunnel.GetSource().Restore(ctx, s.spec, input)
}

func (s *mysqlRestoreSink) Close() {
	s.tunnel.Close()
}
