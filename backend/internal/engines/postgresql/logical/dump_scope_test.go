package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIncludedSchemaNames(t *testing.T) {
	t.Parallel()

	t.Run("returns trimmed unique names when names are pasted multiline", func(t *testing.T) {
		t.Parallel()
		filter := DumpFilter{IncludeSchemas: []string{"\npublic", " reporting ", "public"}}

		assert.Equal(t, []string{"public", "reporting"}, filter.getIncludedSchemaNames())
	})

	t.Run("returns empty non-nil slice when no schemas are configured", func(t *testing.T) {
		t.Parallel()
		filter := DumpFilter{}

		assert.Empty(t, filter.getIncludedSchemaNames())
		assert.NotNil(t, filter.getIncludedSchemaNames())
	})
}

// The cases follow how pg_dump matches --exclude-table patterns.
func TestIsRelationExcluded(t *testing.T) {
	t.Parallel()

	exclusionCases := []struct {
		name                 string
		relation             relationName
		excludeTablePatterns []string
		isExcluded           bool
	}{
		{
			name: "unqualified name matches a table visible on the search path",
			relation: relationName{
				SchemaName: "public", TableName: "audit_logs", IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{"audit_logs"},
			isExcluded:           true,
		},
		{
			name:                 "unqualified name skips a same-named table off the search path",
			relation:             relationName{SchemaName: "reporting", TableName: "audit_logs"},
			excludeTablePatterns: []string{"audit_logs"},
			isExcluded:           false,
		},
		{
			name:                 "star schema part matches a table off the search path",
			relation:             relationName{SchemaName: "reporting", TableName: "audit_logs"},
			excludeTablePatterns: []string{"*.audit_logs"},
			isExcluded:           true,
		},
		{
			name: "unqualified name does not match a different table",
			relation: relationName{
				SchemaName: "public", TableName: "orders", IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{"audit_logs"},
			isExcluded:           false,
		},
		{
			name:                 "schema qualified name matches only its own schema",
			relation:             relationName{SchemaName: "public", TableName: "audit_logs"},
			excludeTablePatterns: []string{"public.audit_logs"},
			isExcluded:           true,
		},
		{
			name:                 "schema qualified name skips the same table in another schema",
			relation:             relationName{SchemaName: "reporting", TableName: "audit_logs"},
			excludeTablePatterns: []string{"public.audit_logs"},
			isExcluded:           false,
		},
		{
			name: "star glob matches a name prefix",
			relation: relationName{
				SchemaName:            "public",
				TableName:             "logs_2026_07",
				IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{"logs_*"},
			isExcluded:           true,
		},
		{
			name:                 "star glob does not cross the schema separator",
			relation:             relationName{SchemaName: "reporting", TableName: "logs_2026_07"},
			excludeTablePatterns: []string{"public.*"},
			isExcluded:           false,
		},
		{
			name:                 "star glob on the schema part matches every table there",
			relation:             relationName{SchemaName: "reporting", TableName: "orders"},
			excludeTablePatterns: []string{"report*.orders"},
			isExcluded:           true,
		},
		{
			name:                 "question mark glob matches a single character",
			relation:             relationName{SchemaName: "public", TableName: "shard_7", IsVisibleOnSearchPath: true},
			excludeTablePatterns: []string{"shard_?"},
			isExcluded:           true,
		},
		{
			name: "question mark glob does not match two characters",
			relation: relationName{
				SchemaName:            "public",
				TableName:             "shard_42",
				IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{"shard_?"},
			isExcluded:           false,
		},
		{
			name:                 "character class glob matches a listed character",
			relation:             relationName{SchemaName: "public", TableName: "shard_b", IsVisibleOnSearchPath: true},
			excludeTablePatterns: []string{"shard_[ab]"},
			isExcluded:           true,
		},
		{
			name: "quoted part is compared literally including the dot",
			relation: relationName{
				SchemaName:            "public",
				TableName:             "weird.name",
				IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{`"weird.name"`},
			isExcluded:           true,
		},
		{
			name: "quoted part does not glob",
			relation: relationName{
				SchemaName:            "public",
				TableName:             "logs_2026",
				IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{`"logs_*"`},
			isExcluded:           false,
		},
		{
			name:                 "quoted schema with unquoted table glob",
			relation:             relationName{SchemaName: "My Schema", TableName: "logs_2026"},
			excludeTablePatterns: []string{`"My Schema".logs_*`},
			isExcluded:           true,
		},
		{
			name: "malformed glob falls back to literal comparison",
			relation: relationName{
				SchemaName:            "public",
				TableName:             "shard_[ab",
				IsVisibleOnSearchPath: true,
			},
			excludeTablePatterns: []string{"shard_[ab"},
			isExcluded:           true,
		},
		{
			name:                 "malformed glob does not match a different name",
			relation:             relationName{SchemaName: "public", TableName: "shard_a", IsVisibleOnSearchPath: true},
			excludeTablePatterns: []string{"shard_[ab"},
			isExcluded:           false,
		},
		{
			name:                 "empty pattern list excludes nothing",
			relation:             relationName{SchemaName: "public", TableName: "orders"},
			excludeTablePatterns: []string{},
			isExcluded:           false,
		},
		{
			name:                 "any matching pattern in the list wins",
			relation:             relationName{SchemaName: "public", TableName: "orders"},
			excludeTablePatterns: []string{"audit_logs", "public.orders"},
			isExcluded:           true,
		},
	}

	for _, exclusionCase := range exclusionCases {
		t.Run(exclusionCase.name, func(t *testing.T) {
			t.Parallel()

			isExcluded := isRelationExcluded(
				exclusionCase.relation,
				exclusionCase.excludeTablePatterns,
			)

			assert.Equal(t, exclusionCase.isExcluded, isExcluded)
		})
	}
}
