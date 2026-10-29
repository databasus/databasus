package logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
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

func (s *Source) runTimescaleDBPreRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(ctx, logger, pgrestore.GetTimescaleDBPreRestoreStatements()...)
}

func (s *Source) runTimescaleDBPostRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(ctx, logger, pgrestore.GetTimescaleDBPostRestoreStatements()...)
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
