package logical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const heavyTableMinSizeMb = 10.0

type sizeTestSchema struct {
	SchemaName     string
	HeavyTableName string
	LightTableName string
}

// The scenarios run one after another: each writes tens of megabytes, and together they would
// outgrow the server's in-memory data directory.
func runGetDumpedSizeMbScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns positive size after data is inserted", func(t *testing.T) {
		database := server.createDatabase(t)

		database.exec(t,
			`CREATE TABLE size_test (id SERIAL PRIMARY KEY, payload TEXT NOT NULL)`,
			`INSERT INTO size_test (payload) SELECT repeat('x', 1024) FROM generate_series(1, 10000)`,
		)

		assert.Greater(t, database.measureDumpedSizeMb(t, DumpFilter{}), 0.0)
	})

	t.Run("skips excluded table when exclude tables is set", func(t *testing.T) {
		database := server.createDatabase(t)
		seededSchema := database.seedSizeTestSchema(t)
		filter := DumpFilter{IncludeSchemas: []string{seededSchema.SchemaName}}

		sizeWithHeavyTableMb := database.measureDumpedSizeMb(t, filter)

		filter.ExcludeTablePatterns = []string{seededSchema.SchemaName + "." + seededSchema.HeavyTableName}
		sizeWithoutHeavyTableMb := database.measureDumpedSizeMb(t, filter)

		assert.InDelta(t,
			database.measureRelationsSizeMb(t,
				relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.LightTableName},
			),
			sizeWithoutHeavyTableMb,
			0.5,
			"excluded table size must not be counted",
		)
		assert.Greater(t, sizeWithHeavyTableMb-sizeWithoutHeavyTableMb, heavyTableMinSizeMb)
	})

	t.Run("skips table with its indexes when exclude table data is set", func(t *testing.T) {
		database := server.createDatabase(t)
		seededSchema := database.seedSizeTestSchema(t)

		database.exec(
			t,
			fmt.Sprintf(`CREATE INDEX ON %s.%s (payload)`, seededSchema.SchemaName, seededSchema.HeavyTableName),
		)

		filter := DumpFilter{IncludeSchemas: []string{seededSchema.SchemaName}}
		sizeWithHeavyTableDataMb := database.measureDumpedSizeMb(t, filter)

		filter.ExcludeTableDataPatterns = []string{seededSchema.SchemaName + "." + seededSchema.HeavyTableName}
		sizeWithoutHeavyTableDataMb := database.measureDumpedSizeMb(t, filter)

		heavyTableTotalSizeMb := database.measureRelationsSizeMb(t,
			relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.HeavyTableName},
		)

		assert.GreaterOrEqual(t,
			sizeWithHeavyTableDataMb-sizeWithoutHeavyTableDataMb,
			heavyTableTotalSizeMb-0.01,
			"the data-excluded table must leave the estimate together with its indexes",
		)
	})

	t.Run("skips matching tables when exclude tables uses a glob pattern", func(t *testing.T) {
		database := server.createDatabase(t)
		seededSchema := database.seedSizeTestSchema(t)

		filter := DumpFilter{
			IncludeSchemas:       []string{seededSchema.SchemaName},
			ExcludeTablePatterns: []string{seededSchema.SchemaName + ".heavy_*"},
		}

		assert.InDelta(t,
			database.measureRelationsSizeMb(t,
				relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.LightTableName},
			),
			database.measureDumpedSizeMb(t, filter),
			0.5,
			"glob-matched table size must not be counted",
		)
	})

	t.Run(
		"counts same-named table in another schema when unqualified table is excluded by either list",
		func(t *testing.T) {
			database := server.createDatabase(t)
			otherSchema := database.seedSizeTestSchema(t)
			sameNamedTableName := otherSchema.HeavyTableName

			database.exec(
				t,
				fmt.Sprintf(
					`CREATE TABLE public.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL)`,
					sameNamedTableName,
				),
				fmt.Sprintf(
					`INSERT INTO public.%s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000)`,
					sameNamedTableName,
				),
			)

			publicTableSizeMb := database.measureRelationsSizeMb(t,
				relationName{SchemaName: "public", TableName: sameNamedTableName},
			)

			includeSchemas := []string{"public", otherSchema.SchemaName}
			sizeWithoutExclusionMb := database.measureDumpedSizeMb(t, DumpFilter{IncludeSchemas: includeSchemas})
			sizeWithWholeTableExclusionMb := database.measureDumpedSizeMb(t, DumpFilter{
				IncludeSchemas:       includeSchemas,
				ExcludeTablePatterns: []string{sameNamedTableName},
			})
			sizeWithDataExclusionMb := database.measureDumpedSizeMb(t, DumpFilter{
				IncludeSchemas:           includeSchemas,
				ExcludeTableDataPatterns: []string{sameNamedTableName},
			})

			assert.InDelta(t,
				publicTableSizeMb,
				sizeWithoutExclusionMb-sizeWithWholeTableExclusionMb,
				0.5,
				"a whole-table exclusion must leave out only the table visible on the default search path",
			)
			assert.InDelta(t,
				publicTableSizeMb,
				sizeWithoutExclusionMb-sizeWithDataExclusionMb,
				0.5,
				"a data exclusion must leave out only the table visible on the default search path",
			)
		},
	)

	t.Run("counts only included schema when include schemas is set", func(t *testing.T) {
		database := server.createDatabase(t)
		includedSchema := database.seedSizeTestSchema(t)
		otherSchema := database.seedSizeTestSchema(t)

		sizeOfIncludedSchemaMb := database.measureDumpedSizeMb(t,
			DumpFilter{IncludeSchemas: []string{includedSchema.SchemaName}},
		)
		sizeOfBothSchemasMb := database.measureDumpedSizeMb(t,
			DumpFilter{IncludeSchemas: []string{includedSchema.SchemaName, otherSchema.SchemaName}},
		)

		assert.InDelta(t,
			database.measureRelationsSizeMb(t,
				relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.HeavyTableName},
				relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.LightTableName},
			),
			sizeOfIncludedSchemaMb,
			0.5,
			"only the included schema must be counted",
		)
		assert.Greater(t, sizeOfBothSchemasMb-sizeOfIncludedSchemaMb, heavyTableMinSizeMb)
	})

	t.Run("applies both filters when include schemas and exclude tables are set", func(t *testing.T) {
		database := server.createDatabase(t)
		includedSchema := database.seedSizeTestSchema(t)
		skippedSchema := database.seedSizeTestSchema(t)

		filter := DumpFilter{
			IncludeSchemas: []string{includedSchema.SchemaName},
			ExcludeTablePatterns: []string{
				includedSchema.SchemaName + "." + includedSchema.HeavyTableName,
				skippedSchema.SchemaName + "." + skippedSchema.LightTableName,
			},
		}

		assert.InDelta(t,
			database.measureRelationsSizeMb(t,
				relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.LightTableName},
			),
			database.measureDumpedSizeMb(t, filter),
			0.5,
			"schema filter and table exclusion must both apply",
		)
	})

	t.Run("counts every user table when no filters are set", func(t *testing.T) {
		database := server.createDatabase(t)
		firstSchema := database.seedSizeTestSchema(t)
		secondSchema := database.seedSizeTestSchema(t)

		sizeOfWholeDatabaseMb := database.measureDumpedSizeMb(t, DumpFilter{})
		sizeOfFirstSchemaMb := database.measureDumpedSizeMb(t,
			DumpFilter{IncludeSchemas: []string{firstSchema.SchemaName}},
		)

		seededSchemasSizeMb := sizeOfFirstSchemaMb + database.measureRelationsSizeMb(t,
			relationName{SchemaName: secondSchema.SchemaName, TableName: secondSchema.HeavyTableName},
			relationName{SchemaName: secondSchema.SchemaName, TableName: secondSchema.LightTableName},
		)

		assert.GreaterOrEqual(t, sizeOfWholeDatabaseMb, seededSchemasSizeMb-0.5)
		assert.Greater(t, sizeOfWholeDatabaseMb-sizeOfFirstSchemaMb, heavyTableMinSizeMb)
	})

	t.Run("counts every partition once when table is partitioned", func(t *testing.T) {
		database := server.createDatabase(t)
		schemaName := "partitioned_size_" + newNameSuffix()

		database.exec(
			t,
			fmt.Sprintf(`CREATE SCHEMA %s`, schemaName),
			fmt.Sprintf(`CREATE TABLE %s.events (id BIGINT, payload TEXT) PARTITION BY RANGE (id)`, schemaName),
			fmt.Sprintf(
				`CREATE TABLE %s.events_first PARTITION OF %s.events FOR VALUES FROM (0) TO (10000)`,
				schemaName,
				schemaName,
			),
			fmt.Sprintf(
				`CREATE TABLE %s.events_second PARTITION OF %s.events FOR VALUES FROM (10000) TO (20000)`,
				schemaName, schemaName,
			),
			fmt.Sprintf(`INSERT INTO %s.events SELECT generate_series(0, 19999), repeat('x', 1024)`, schemaName),
		)

		assert.InDelta(t,
			database.measureRelationsSizeMb(t,
				relationName{SchemaName: schemaName, TableName: "events_first"},
				relationName{SchemaName: schemaName, TableName: "events_second"},
			),
			database.measureDumpedSizeMb(t, DumpFilter{IncludeSchemas: []string{schemaName}}),
			0.5,
			"partitions must be counted once, without adding the partitioned parent",
		)
	})
}

func (d testDatabase) seedSizeTestSchema(t *testing.T) sizeTestSchema {
	t.Helper()

	suffix := newNameSuffix()
	seededSchema := sizeTestSchema{
		SchemaName:     "size_" + suffix,
		HeavyTableName: "heavy_" + suffix,
		LightTableName: "light_" + suffix,
	}

	d.exec(
		t,
		fmt.Sprintf(`CREATE SCHEMA %s`, seededSchema.SchemaName),
		fmt.Sprintf(
			`CREATE TABLE %s.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL)`,
			seededSchema.SchemaName, seededSchema.HeavyTableName,
		),
		fmt.Sprintf(
			`CREATE TABLE %s.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL)`,
			seededSchema.SchemaName, seededSchema.LightTableName,
		),
		fmt.Sprintf(
			`INSERT INTO %s.%s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000)`,
			seededSchema.SchemaName, seededSchema.HeavyTableName,
		),
		fmt.Sprintf(
			`INSERT INTO %s.%s (payload) VALUES ('light')`,
			seededSchema.SchemaName,
			seededSchema.LightTableName,
		),
	)

	return seededSchema
}

func (d testDatabase) measureDumpedSizeMb(t *testing.T, filter DumpFilter) float64 {
	t.Helper()

	sizeMb, err := d.newAdminSource().GetDumpedSizeMb(t.Context(), newTestLogger(t), filter)
	require.NoError(t, err)

	return sizeMb
}

func (d testDatabase) measureRelationsSizeMb(t *testing.T, relations ...relationName) float64 {
	t.Helper()

	var totalSizeMb float64

	for _, relation := range relations {
		var relationSizeMb float64

		err := d.admin.QueryRow(t.Context(), `
			SELECT pg_total_relation_size(format('%I.%I', $1::text, $2::text)::regclass) / (1024.0 * 1024.0)
		`, relation.SchemaName, relation.TableName).Scan(&relationSizeMb)
		require.NoError(t, err)

		totalSizeMb += relationSizeMb
	}

	return totalSizeMb
}
