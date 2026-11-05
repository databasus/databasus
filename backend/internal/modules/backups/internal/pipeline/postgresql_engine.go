package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

type postgresqlEngine struct {
	databases      connectionSettingsReader
	clientToolsDir string
	tempDir        string
}

func (e *postgresqlEngine) GetStoredCompression() storedCompression {
	return getStoredCompressionNone()
}

func (e *postgresqlEngine) GetFileExtension() string {
	return ".dump"
}

func (e *postgresqlEngine) OpenDumpSource(
	ctx context.Context,
	databaseID uuid.UUID,
	logger *slog.Logger,
) (logicalDumpSource, error) {
	logger.InfoContext(ctx, "creating postgresql backup via pg_dump custom format")

	connectionSettings, err := e.databases.GetPostgresqlLogicalConnectionSettings(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	return openPostgresqlDumpSource(ctx, connectionSettings, pgshared.ClientTools{Root: e.clientToolsDir}, logger)
}

func (e *postgresqlEngine) OpenRestoreTarget(
	ctx context.Context,
	spec restoreTargetSpec,
) (logicalRestoreSink, error) {
	target, isOfEngine := spec.target.(PostgresqlRestoreTarget)
	if !isOfEngine {
		return nil, ErrTargetOfAnotherEngine
	}

	err := target.Settings.TLS.Validate()
	if err != nil {
		return nil, &InvalidTargetTLSError{Err: err}
	}

	sourceSettings, err := e.databases.GetPostgresqlLogicalConnectionSettings(ctx, spec.sourceDatabaseID)
	if err != nil {
		return nil, err
	}

	tunnel, err := pglogical.NewSource(target.Settings).OpenTunnel(ctx, spec.logger)
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

	if pgshared.IsBackupVersionHigherThanRestoreVersion(sourceSettings.Version, targetVersion) {
		tunnel.Close()

		return nil, fmt.Errorf("%w: the backup is of PostgreSQL %s and the target runs PostgreSQL %s",
			ErrTargetVersionTooOld, sourceSettings.Version, targetVersion)
	}

	return &postgresqlRestoreSink{
		tunnel: tunnel,
		spec: pglogical.RestoreSpec{
			NewCleanupContext:   spec.newCleanupContext,
			ClientTools:         pgshared.ClientTools{Root: e.clientToolsDir},
			Version:             targetVersion,
			SourceDatabaseName:  sourceSettings.Engine.Database,
			ParallelJobs:        target.CPUCount,
			IsTimescaleDB:       spec.isTimescaleDB,
			IsRestoreOwnership:  target.IsRestoreOwnership,
			IsRestorePrivileges: target.IsRestorePrivileges,
			IsExcludeExtensions: target.IsExcludeExtensions,
			IsSkipUserMappings:  sourceSettings.IsSkipUserMappings,
			TempDir:             e.tempDir,
			Logger:              spec.logger,
		},
	}, nil
}

type postgresqlDumpSource struct {
	tunnel     *pglogical.Tunnel
	dumpFilter pglogical.DumpFilter
	spec       pglogical.DumpSpec
	logger     *slog.Logger
}

func openPostgresqlDumpSource(
	ctx context.Context,
	connectionSettings databases.PostgresqlLogicalConnectionSettings,
	clientTools pgshared.ClientTools,
	logger *slog.Logger,
) (*postgresqlDumpSource, error) {
	tunnel, err := pglogical.NewSource(connectionSettings.Engine).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &postgresqlDumpSource{
		tunnel:     tunnel,
		dumpFilter: connectionSettings.GetDumpFilter(),
		spec: pglogical.DumpSpec{
			ClientTools:      clientTools,
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
		s.logger.WarnContext(ctx, "failed to fetch raw db size before backup", slog.Any("error", err))

		return 0, false
	}

	return rawSizeMb, true
}

func (s *postgresqlDumpSource) GetTimescaleDBVersion(ctx context.Context) string {
	timescaleDBVersion, err := s.tunnel.GetSource().GetTimescaleDBVersion(ctx, s.logger)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to detect timescaledb extension before backup", slog.Any("error", err))

		return ""
	}

	return timescaleDBVersion
}

func (s *postgresqlDumpSource) Dump(ctx context.Context, output io.Writer) error {
	return s.tunnel.GetSource().Dump(ctx, s.spec, output)
}

func (s *postgresqlDumpSource) Close() {
	s.tunnel.Close()
}

type postgresqlRestoreSink struct {
	tunnel *pglogical.Tunnel
	spec   pglogical.RestoreSpec
}

func (s *postgresqlRestoreSink) IsArchiveFileNeeded() bool {
	return s.spec.NeedsArchiveFile()
}

func (s *postgresqlRestoreSink) Restore(ctx context.Context, input io.Reader) error {
	restoreErr := s.tunnel.GetSource().Restore(ctx, s.spec, input)
	archiveFailure, hasArchiveFailure := errors.AsType[*pglogical.ArchiveRestoreError](restoreErr)
	if hasArchiveFailure {
		return &archiveRestoreError{Err: archiveFailure.Err}
	}

	return restoreErr
}

func (s *postgresqlRestoreSink) Close() {
	s.tunnel.Close()
}
