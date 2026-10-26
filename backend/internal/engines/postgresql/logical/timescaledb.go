package logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// GetTimescaleDBVersion returns the installed timescaledb extension version, or "" when the
// extension is absent. It is captured at backup time, so a restore knows whether the TimescaleDB
// restore procedure is required, and the verification agent knows which engine image to pull.
func (s *Source) GetTimescaleDBVersion(ctx context.Context, logger *slog.Logger) (string, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	var version string

	err = conn.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'").Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to query timescaledb extension version: %w", err)
	}

	return version, nil
}

// TimescaleDB restore procedure, executed against the restore target.
//
// timescaledb_pre_restore() runs `ALTER DATABASE <db> SET timescaledb.restoring='on'`, a
// database-level GUC that persists across new connections, and stops background workers. Because
// the GUC lives on the database rather than the session, the separate pg_restore process that runs
// next inherits restoring mode; that is the whole reason these hooks can run on connections distinct
// from pg_restore's. timescaledb_post_restore() resets the GUC and restarts the workers, so it MUST
// run even when pg_restore fails, or the database is left in restoring mode with its workers
// stopped. Callers run RunTimescaleDBPostRestore via defer.

// RunTimescaleDBPreRestore installs the extension, because a freshly created restore target has none
// and pre_restore needs its functions, and enters restoring mode.
func (s *Source) RunTimescaleDBPreRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(
		ctx,
		logger,
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		"SELECT timescaledb_pre_restore()",
	)
}

func (s *Source) RunTimescaleDBPostRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(ctx, logger, "SELECT timescaledb_post_restore()")
}

func (s *Source) execStatements(ctx context.Context, logger *slog.Logger, statements ...string) error {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to run %q: %w", statement, err)
		}
	}

	return nil
}
