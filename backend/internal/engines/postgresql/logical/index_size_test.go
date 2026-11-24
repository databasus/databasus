package logical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scenarios run one after another for the same reason as the dumped size scenarios: each
// writes megabytes of rows into the server's in-memory data directory.
func runGetLargestDumpedIndexSizesMbScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns the eight largest indexes largest first when ten tables have one index each", func(t *testing.T) {
		database := server.createDatabase(t)
		schemaName := "index_sizes_" + newNameSuffix()

		database.exec(t, fmt.Sprintf(`CREATE SCHEMA %s`, schemaName))

		indexSizesMb := make([]float64, 0, 10)
		for tableNumber := 1; tableNumber <= 10; tableNumber++ {
			database.exec(t,
				fmt.Sprintf(`CREATE TABLE %s.t%d (payload TEXT NOT NULL)`, schemaName, tableNumber),
				fmt.Sprintf(
					`INSERT INTO %s.t%d SELECT md5(g::text) FROM generate_series(1, %d) g`,
					schemaName, tableNumber, tableNumber*2000,
				),
				fmt.Sprintf(`CREATE INDEX t%d_payload ON %s.t%d (payload)`, tableNumber, schemaName, tableNumber),
			)
			indexSizesMb = append(
				[]float64{database.measureIndexSizeMb(t, schemaName, fmt.Sprintf("t%d_payload", tableNumber))},
				indexSizesMb...,
			)
		}

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t,
			DumpFilter{IncludeSchemas: []string{schemaName}},
		)

		assert.Equal(t, indexSizesMb[:8], largestIndexSizesMb)
	})

	t.Run("counts the index of a primary key", func(t *testing.T) {
		database := server.createDatabase(t)
		seededSchema := database.seedSizeTestSchema(t)

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t,
			DumpFilter{IncludeSchemas: []string{seededSchema.SchemaName}},
		)

		require.Len(t, largestIndexSizesMb, 2)
		assert.InDelta(t,
			database.measureIndexSizeMb(t, seededSchema.SchemaName, seededSchema.HeavyTableName+"_pkey"),
			largestIndexSizesMb[0],
			0,
		)
	})

	t.Run("leaves out the index of a table whose data is excluded", func(t *testing.T) {
		database := server.createDatabase(t)
		seededSchema := database.seedSizeTestSchema(t)

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t, DumpFilter{
			IncludeSchemas:           []string{seededSchema.SchemaName},
			ExcludeTableDataPatterns: []string{seededSchema.SchemaName + "." + seededSchema.HeavyTableName},
		})

		assert.Equal(t,
			[]float64{database.measureIndexSizeMb(t, seededSchema.SchemaName, seededSchema.LightTableName+"_pkey")},
			largestIndexSizesMb,
		)
	})

	t.Run("counts a same-named table in another schema when an unqualified table is excluded", func(t *testing.T) {
		database := server.createDatabase(t)

		database.exec(t,
			`CREATE SCHEMA archive`,
			`CREATE TABLE public.logs (id SERIAL PRIMARY KEY)`,
			`INSERT INTO public.logs SELECT FROM generate_series(1, 40000)`,
			`CREATE TABLE archive.logs (id SERIAL PRIMARY KEY)`,
			`INSERT INTO archive.logs SELECT FROM generate_series(1, 20000)`,
		)

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t, DumpFilter{
			IncludeSchemas:       []string{"public", "archive"},
			ExcludeTablePatterns: []string{"logs"},
		})

		assert.Equal(t, []float64{database.measureIndexSizeMb(t, "archive", "logs_pkey")}, largestIndexSizesMb)
	})

	t.Run("counts each partition's index once when the partitioned table's data is excluded", func(t *testing.T) {
		database := server.createDatabase(t)

		database.exec(t,
			`CREATE TABLE public.events (id BIGINT NOT NULL) PARTITION BY RANGE (id)`,
			`CREATE TABLE public.events_2025 PARTITION OF public.events FOR VALUES FROM (0) TO (30000)`,
			`CREATE TABLE public.events_2026 PARTITION OF public.events FOR VALUES FROM (30000) TO (40000)`,
			`CREATE INDEX events_id ON public.events (id)`,
			`INSERT INTO public.events SELECT generate_series(0, 39999)`,
		)

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t, DumpFilter{
			IncludeSchemas:           []string{"public"},
			ExcludeTableDataPatterns: []string{"public.events"},
		})

		assert.Equal(t,
			[]float64{
				database.measureIndexSizeMb(t, "public", "events_2025_id_idx"),
				database.measureIndexSizeMb(t, "public", "events_2026_id_idx"),
			},
			largestIndexSizesMb,
		)
	})

	t.Run("returns an empty list when the dumped tables have no indexes", func(t *testing.T) {
		database := server.createDatabase(t)
		schemaName := "no_indexes_" + newNameSuffix()

		database.exec(t,
			fmt.Sprintf(`CREATE SCHEMA %s`, schemaName),
			fmt.Sprintf(`CREATE TABLE %s.plain (payload TEXT)`, schemaName),
			fmt.Sprintf(`INSERT INTO %s.plain VALUES ('row')`, schemaName),
		)

		largestIndexSizesMb := database.measureLargestDumpedIndexSizesMb(t,
			DumpFilter{IncludeSchemas: []string{schemaName}},
		)

		require.NotNil(t, largestIndexSizesMb)
		assert.Empty(t, largestIndexSizesMb)
	})
}

func (d testDatabase) measureLargestDumpedIndexSizesMb(t *testing.T, filter DumpFilter) []float64 {
	t.Helper()

	largestIndexSizesMb, err := d.newAdminSource().GetLargestDumpedIndexSizesMb(t.Context(), newTestLogger(t), filter)
	require.NoError(t, err)

	return largestIndexSizesMb
}

func (d testDatabase) measureIndexSizeMb(t *testing.T, schemaName, indexName string) float64 {
	t.Helper()

	var indexSizeBytes int64

	err := d.admin.QueryRow(t.Context(), `
		SELECT pg_relation_size(format('%I.%I', $1::text, $2::text)::regclass)
	`, schemaName, indexName).Scan(&indexSizeBytes)
	require.NoError(t, err)

	return float64(indexSizeBytes) / (1024 * 1024)
}
