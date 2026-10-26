package logical

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// Materialized views are left out on purpose: pg_dump emits their definition but
// not their contents, so counting them would overstate what a restore recreates.
const dumpedRelationsSizeQuery = `
	SELECT n.nspname,
	       c.relname,
	       pg_catalog.pg_table_is_visible(c.oid),
	       COALESCE(pg_total_relation_size(c.oid), 0)
` + dumpedRelationScope + `
	  AND c.relkind IN ('r', 'p')
`

// GetDumpedSizeMb sums the relations pg_dump emits for filter. The value feeds verification disk
// planning and the restored-size sanity check, so it covers exactly what the dump holds.
func (s *Source) GetDumpedSizeMb(ctx context.Context, logger *slog.Logger, filter DumpFilter) (float64, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	sizeBytes, err := getDumpedRelationsSizeBytes(ctx, conn, filter)
	if err != nil {
		return 0, err
	}

	return float64(sizeBytes) / (1024 * 1024), nil
}

func getDumpedRelationsSizeBytes(
	ctx context.Context,
	conn *pgx.Conn,
	filter DumpFilter,
) (int64, error) {
	rows, err := queryDumpedRelations(ctx, conn, dumpedRelationsSizeQuery, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to query dumped relations size: %w", err)
	}

	defer rows.Close()

	excludeTablePatterns := filter.getExcludeTablePatterns()
	excludeTableDataPatterns := filter.getExcludeTableDataPatterns()

	var totalSizeBytes int64

	for rows.Next() {
		var relation relationName
		var relationSizeBytes int64

		if err := rows.Scan(
			&relation.SchemaName,
			&relation.TableName,
			&relation.IsVisibleOnSearchPath,
			&relationSizeBytes,
		); err != nil {
			return 0, fmt.Errorf("failed to scan dumped relation size: %w", err)
		}

		if isRelationExcluded(relation, excludeTablePatterns) ||
			isRelationExcluded(relation, excludeTableDataPatterns) {
			continue
		}

		totalSizeBytes += relationSizeBytes
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to read dumped relations size: %w", err)
	}

	return totalSizeBytes, nil
}
