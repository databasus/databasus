package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

const rawSizeQuery = `
	SELECT COALESCE(SUM(data_length + index_length), 0) / (1024 * 1024)
	FROM information_schema.tables
	WHERE table_schema = ?
`

// GetRawSizeMb sums the data and index size of the database's tables. MySQL from 8.0 caches these
// statistics for a day by default, so the session turns the cache off on those versions.
func (s *Source) GetRawSizeMb(ctx context.Context, logger *slog.Logger, version Version) (float64, error) {
	db, err := s.openDB()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MySQL database '%s': %w", s.settings.Database, err)
	}
	defer closeDB(ctx, db, logger)

	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire MySQL connection: %w", err)
	}
	defer releaseConn(ctx, conn, logger)

	if version != Version57 {
		if _, err := conn.ExecContext(ctx, "SET SESSION information_schema_stats_expiry = 0"); err != nil {
			return 0, fmt.Errorf("failed to disable MySQL statistics cache: %w", err)
		}
	}

	var sizeMb float64
	if err := conn.QueryRowContext(ctx, rawSizeQuery, s.settings.Database).Scan(&sizeMb); err != nil {
		return 0, fmt.Errorf("failed to query MySQL database size: %w", err)
	}

	return sizeMb, nil
}

func releaseConn(ctx context.Context, conn *sql.Conn, logger *slog.Logger) {
	if err := conn.Close(); err != nil {
		logger.ErrorContext(ctx, "failed to release MySQL connection", slog.Any("error", err))
	}
}
