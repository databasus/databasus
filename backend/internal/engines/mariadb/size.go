package mariadb

import (
	"context"
	"fmt"
	"log/slog"
)

const rawSizeQuery = `
	SELECT COALESCE(SUM(data_length + index_length), 0) / (1024 * 1024)
	FROM information_schema.tables
	WHERE table_schema = ?
`

// GetRawSizeMb sums the data and index size of the database's tables.
func (s *Source) GetRawSizeMb(ctx context.Context, logger *slog.Logger) (float64, error) {
	db, err := s.openDB()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MariaDB database '%s': %w", s.settings.Database, err)
	}
	defer closeDB(ctx, db, logger)

	var sizeMb float64
	if err := db.QueryRowContext(ctx, rawSizeQuery, s.settings.Database).Scan(&sizeMb); err != nil {
		return 0, fmt.Errorf("failed to query MariaDB database size: %w", err)
	}

	return sizeMb, nil
}
