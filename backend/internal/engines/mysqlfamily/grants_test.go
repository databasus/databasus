package mysqlfamily

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGrantLine(t *testing.T) {
	t.Parallel()

	grantCases := []struct {
		name          string
		grantLine     string
		privileges    []string
		scope         GrantScope
		schemaPattern string
		tableName     string
	}{
		{
			// issue #568: the privilege list must be split before it is read, or "SHOW CREATE
			// ROUTINE" registers as a CREATE grant.
			name:       "keeps SHOW CREATE ROUTINE whole without splitting out CREATE",
			grantLine:  "GRANT SELECT, SHOW VIEW, SHOW CREATE ROUTINE ON *.* TO 'backup'@'%'",
			privileges: []string{"SELECT", "SHOW VIEW", "SHOW CREATE ROUTINE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:       "reads global write privileges",
			grantLine:  "GRANT SELECT, INSERT, UPDATE ON *.* TO 'x'@'%'",
			privileges: []string{"SELECT", "INSERT", "UPDATE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:          "reads all privileges with a schema scope",
			grantLine:     "GRANT ALL PRIVILEGES ON `appdb`.* TO 'x'@'%'",
			privileges:    []string{"ALL PRIVILEGES"},
			scope:         GrantScopeSchema,
			schemaPattern: "appdb",
		},
		{
			name:       "reads a usage only line",
			grantLine:  "GRANT USAGE ON *.* TO 'x'@'%'",
			privileges: []string{"USAGE"},
			scope:      GrantScopeGlobal,
		},
		{
			name:          "drops a column level privilege with a column list",
			grantLine:     "GRANT SELECT (col1, col2), SHOW VIEW ON `appdb`.`orders` TO 'x'@'%'",
			privileges:    []string{"SHOW VIEW"},
			scope:         GrantScopeTable,
			schemaPattern: "appdb",
			tableName:     "orders",
		},
		{
			name:          "reads a table scope with a table grant",
			grantLine:     "GRANT SELECT, SHOW VIEW ON `appdb`.`orders` TO 'backup'@'%'",
			privileges:    []string{"SELECT", "SHOW VIEW"},
			scope:         GrantScopeTable,
			schemaPattern: "appdb",
			tableName:     "orders",
		},
		{
			name:          "keeps a wildcard schema pattern as written",
			grantLine:     "GRANT SELECT ON `app%`.* TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeSchema,
			schemaPattern: "app%",
		},
		{
			name:          "keeps an escaped underscore in the schema pattern",
			grantLine:     "GRANT SELECT ON `test\\_db`.* TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeSchema,
			schemaPattern: `test\_db`,
		},
		{
			name:          "reads the schema with a dot in its name",
			grantLine:     "GRANT SELECT ON `my.db`.`orders` TO 'x'@'%'",
			privileges:    []string{"SELECT"},
			scope:         GrantScopeTable,
			schemaPattern: "my.db",
			tableName:     "orders",
		},
		{
			name:       "ignores the trailer with grant option",
			grantLine:  "GRANT SELECT, INSERT ON *.* TO 'x'@'%' WITH GRANT OPTION",
			privileges: []string{"SELECT", "INSERT"},
			scope:      GrantScopeGlobal,
		},
		{
			name:       "reads privileges with mixed case keywords",
			grantLine:  "grant Select, Update on *.* to 'x'@'%'",
			privileges: []string{"SELECT", "UPDATE"},
			scope:      GrantScopeGlobal,
		},
	}

	for _, grantCase := range grantCases {
		t.Run(grantCase.name, func(t *testing.T) {
			t.Parallel()

			grant := ParseGrantLine(grantCase.grantLine)

			require.NotNil(t, grant)
			assert.Equal(t, grantCase.privileges, grant.Privileges)
			assert.Equal(t, grantCase.scope, grant.Scope)
			assert.Equal(t, grantCase.schemaPattern, grant.SchemaPattern)
			assert.Equal(t, grantCase.tableName, grant.TableName)
		})
	}

	ignoredGrantLines := []struct {
		name      string
		grantLine string
	}{
		{"returns nil with a role grant that has no ON clause", "GRANT `reporting_role`@`%` TO 'u'@'%'"},
		{"returns nil with a proxy grant that names an account", "GRANT PROXY ON 'other'@'%' TO 'u'@'%'"},
		{"returns nil with a revoke line", "REVOKE SELECT ON `appdb`.* FROM 'u'@'%'"},
		{
			"returns nil when every privilege is column qualified",
			"GRANT SELECT (on) ON `appdb`.`orders` TO 'x'@'%'",
		},
	}

	for _, ignoredGrantLine := range ignoredGrantLines {
		t.Run(ignoredGrantLine.name, func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, ParseGrantLine(ignoredGrantLine.grantLine))
		})
	}
}

func TestIsMatchingSchemaPattern(t *testing.T) {
	t.Parallel()

	matchCases := []struct {
		name          string
		schemaPattern string
		schemaName    string
		isMatching    bool
	}{
		{"matches itself with a plain name", "appdb", "appdb", true},
		{"rejects another name with a plain name", "appdb", "otherdb", false},
		{"matches a suffix with a percent", "app%", "appdb", true},
		{"matches everything with a lone percent", "%", "anything", true},
		{"matches one character with an underscore", "app_b", "appXb", true},
		{"matches a literal underscore with an escaped one", `test\_db`, "test_db", true},
		{"rejects another character with an escaped underscore", `test\_db`, "testXdb", false},
		{"rejects a longer name without a wildcard", "app", "appdb", false},
	}

	for _, matchCase := range matchCases {
		t.Run(matchCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				matchCase.isMatching,
				isMatchingSchemaPattern(matchCase.schemaPattern, matchCase.schemaName),
			)
		})
	}
}

func TestCollectSchemaGrants(t *testing.T) {
	t.Parallel()

	t.Run("grants no EVENT when all privileges are table scoped", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{"GRANT ALL PRIVILEGES ON `appdb`.`orders` TO 'x'@'%'"},
			"appdb",
		)

		assert.True(t, collectedGrants.HasForTable("SELECT", "orders"))
		assert.True(t, collectedGrants.HasForTable("SHOW VIEW", "orders"))
		assert.False(t, collectedGrants.HasForTable("EVENT", "orders"))
	})

	t.Run("ignores a grant when it targets another schema", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{"GRANT SELECT ON `otherdb`.* TO 'x'@'%'"},
			"appdb",
		)

		assert.False(t, collectedGrants.HasForSchema("SELECT"))
	})

	t.Run("subtracts from the global grant when the schema is partially revoked", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{
				"GRANT SELECT, SHOW VIEW ON *.* TO 'x'@'%'",
				"REVOKE SELECT ON `appdb`.* FROM 'x'@'%'",
			},
			"appdb",
		)

		assert.False(t, collectedGrants.HasForSchema("SELECT"))
		assert.True(t, collectedGrants.HasForSchema("SHOW VIEW"))
	})

	t.Run("subtracts from the global grant when the revoke uses a schema pattern", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{
				"GRANT SELECT ON *.* TO 'x'@'%'",
				"REVOKE SELECT ON `app%`.* FROM 'x'@'%'",
			},
			"appdb",
		)

		assert.False(t, collectedGrants.HasForSchema("SELECT"))
	})

	t.Run("keeps the global grant when another schema is revoked", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{
				"GRANT SELECT ON *.* TO 'x'@'%'",
				"REVOKE SELECT ON `otherdb`.* FROM 'x'@'%'",
			},
			"appdb",
		)

		assert.True(t, collectedGrants.HasForSchema("SELECT"))
	})

	t.Run("keeps the schema grant when a revoked schema is granted explicitly", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{
				"GRANT SELECT ON *.* TO 'x'@'%'",
				"REVOKE SELECT ON `appdb`.* FROM 'x'@'%'",
				"GRANT SELECT ON `appdb`.* TO 'x'@'%'",
			},
			"appdb",
		)

		assert.True(t, collectedGrants.HasForSchema("SELECT"))
	})

	t.Run("removes every required privilege when all privileges are revoked", func(t *testing.T) {
		t.Parallel()

		collectedGrants := CollectSchemaGrants(
			[]string{
				"GRANT ALL PRIVILEGES ON *.* TO 'x'@'%'",
				"REVOKE ALL PRIVILEGES ON `appdb`.* FROM 'x'@'%'",
			},
			"appdb",
		)

		assert.False(t, collectedGrants.HasForSchema("SELECT"))
		assert.False(t, collectedGrants.HasForSchema("SHOW VIEW"))
		assert.False(t, collectedGrants.HasForSchema("TRIGGER"))
		assert.False(t, collectedGrants.HasForSchema("EVENT"))
	})
}
