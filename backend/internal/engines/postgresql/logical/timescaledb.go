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
