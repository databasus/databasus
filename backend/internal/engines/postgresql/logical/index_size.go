package logical

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
)

// An index of a partitioned table has no storage of its own (relkind 'I'); its partitions' indexes
// are ordinary indexes of the partitions, which the scope lists as tables, so each counts once.
const dumpedTableIndexSizesQuery = `
	SELECT n.nspname,
	       c.relname,
	       pg_catalog.pg_table_is_visible(c.oid),
	       ARRAY(
	         SELECT COALESCE(pg_relation_size(i.indexrelid), 0)
	         FROM pg_index i
	         JOIN pg_class ic ON ic.oid = i.indexrelid
	         WHERE i.indrelid = c.oid
	           AND ic.relkind = 'i'
	       )
` + dumpedRelationScope + `
	  AND c.relkind IN ('r', 'p')
`

// GetLargestDumpedIndexSizesMb returns the sizes of the largest indexes a restore of the dump
// builds, largest first, as many as pg_restore runs jobs at most. A verification restore builds
// that many indexes at once, and each spills a sort about the size of its index to disk, so these
// sizes plan the restore's disk. The list is empty, not nil, when the dumped tables have no index.
func (s *Source) GetLargestDumpedIndexSizesMb(
	ctx context.Context,
	logger *slog.Logger,
	filter DumpFilter,
) ([]float64, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	rows, err := queryDumpedRelations(ctx, conn, dumpedTableIndexSizesQuery, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query dumped index sizes: %w", err)
	}

	defer rows.Close()

	excludeTablePatterns := filter.getExcludeTablePatterns()
	excludeTableDataPatterns := filter.getExcludeTableDataPatterns()
	indexSizesBytes := make([]int64, 0)

	for rows.Next() {
		var relation relationName
		var tableIndexSizesBytes []int64

		if err := rows.Scan(
			&relation.SchemaName,
			&relation.TableName,
			&relation.IsVisibleOnSearchPath,
			&tableIndexSizesBytes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dumped index sizes: %w", err)
		}

		if isRelationExcluded(relation, excludeTablePatterns) ||
			isRelationExcluded(relation, excludeTableDataPatterns) {
			continue
		}

		indexSizesBytes = append(indexSizesBytes, tableIndexSizesBytes...)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read dumped index sizes: %w", err)
	}

	return getLargestSizesMb(indexSizesBytes, pgrestore.MaxParallelJobs), nil
}

func getLargestSizesMb(sizesBytes []int64, count int) []float64 {
	slices.SortFunc(sizesBytes, func(left, right int64) int { return cmp.Compare(right, left) })

	largestSizesMb := make([]float64, 0, min(count, len(sizesBytes)))
	for _, sizeBytes := range sizesBytes[:min(count, len(sizesBytes))] {
		largestSizesMb = append(largestSizesMb, float64(sizeBytes)/(1024*1024))
	}

	return largestSizesMb
}
