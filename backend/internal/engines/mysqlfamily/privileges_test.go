package mysqlfamily

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBackupPrivilegesSpec(grantLines, dumpedTableNames []string) BackupPrivilegesSpec {
	dumpedTables := make([]DumpedTable, 0, len(dumpedTableNames))
	for _, tableName := range dumpedTableNames {
		dumpedTables = append(dumpedTables, DumpedTable{Name: tableName})
	}

	return BackupPrivilegesSpec{
		SchemaName:   "appdb",
		Grantee:      "'backup'@'%'",
		DumpedTables: dumpedTables,
		Grants:       CollectSchemaGrants(grantLines, "appdb"),
	}
}

func TestGetBackupPrivileges(t *testing.T) {
	t.Parallel()

	t.Run("reports sufficient when a schema grant covers the dump set", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{"GRANT SELECT, SHOW VIEW, TRIGGER ON `appdb`.* TO 'backup'@'%'"},
			[]string{"orders", "payments"},
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
		assert.Equal(t, "SELECT,SHOW VIEW,TRIGGER", backupPrivileges.GetEffectivePrivilegesCsv())
	})

	t.Run("reports sufficient when table grants cover every dumped table", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{
				"GRANT SELECT, SHOW VIEW ON `appdb`.`orders` TO 'backup'@'%'",
				"GRANT SELECT, SHOW VIEW ON `appdb`.`payments` TO 'backup'@'%'",
			},
			[]string{"orders", "payments"},
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
		assert.Equal(t, "SELECT,SHOW VIEW", backupPrivileges.GetEffectivePrivilegesCsv())
	})

	// mysqldump needs SHOW VIEW only to emit a view's definition, so a schema of plain tables must
	// not be rejected for lacking it.
	t.Run("does not require SHOW VIEW when the schema holds no views", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{"GRANT SELECT ON `appdb`.* TO 'backup'@'%'"},
			[]string{"orders", "payments"},
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
	})

	// The base table sits in the fixture to show it is not implicated by a view's missing privilege.
	t.Run("reports the schema scope when every view lacks SHOW VIEW", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(BackupPrivilegesSpec{
			SchemaName: "appdb",
			Grantee:    "'backup'@'%'",
			DumpedTables: []DumpedTable{
				{Name: "orders"},
				{Name: "orders_view", IsView: true},
			},
			Grants: CollectSchemaGrants(
				[]string{"GRANT SELECT ON `appdb`.* TO 'backup'@'%'"},
				"appdb",
			),
		})

		require.Len(t, backupPrivileges.Missing, 1)
		assert.Equal(t, "SHOW VIEW", backupPrivileges.Missing[0].Privilege)
		assert.Equal(t, GrantScopeSchema, backupPrivileges.Missing[0].Scope)
	})

	t.Run("lists the view when one of two views lacks SHOW VIEW", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(BackupPrivilegesSpec{
			SchemaName: "appdb",
			Grantee:    "'backup'@'%'",
			DumpedTables: []DumpedTable{
				{Name: "orders"},
				{Name: "granted_view", IsView: true},
				{Name: "ungranted_view", IsView: true},
			},
			Grants: CollectSchemaGrants(
				[]string{
					"GRANT SELECT ON `appdb`.* TO 'backup'@'%'",
					"GRANT SHOW VIEW ON `appdb`.`granted_view` TO 'backup'@'%'",
				},
				"appdb",
			),
		})

		require.Len(t, backupPrivileges.Missing, 1)
		assert.Equal(t, "SHOW VIEW", backupPrivileges.Missing[0].Privilege)
		assert.Equal(t, GrantScopeTable, backupPrivileges.Missing[0].Scope)
		assert.Equal(t, []string{"ungranted_view"}, backupPrivileges.Missing[0].Tables)
	})

	t.Run("lists the tables when a privilege is missing on some of them", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{
				"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
				"GRANT SELECT ON `appdb`.`orders` TO 'backup'@'%'",
			},
			[]string{"orders", "payments"},
		))

		require.Len(t, backupPrivileges.Missing, 1)
		assert.Equal(t, "SELECT", backupPrivileges.Missing[0].Privilege)
		assert.Equal(t, GrantScopeTable, backupPrivileges.Missing[0].Scope)
		assert.Equal(t, []string{"payments"}, backupPrivileges.Missing[0].Tables)
		assert.False(t, backupPrivileges.IsSufficientForDump())
	})

	// A privilege is recorded only when it covers the whole dump, because the dump flags it gates
	// are all-or-nothing for the run.
	t.Run("omits TRIGGER when it covers only some tables", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{
				"GRANT SELECT, SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
				"GRANT TRIGGER ON `appdb`.`orders` TO 'backup'@'%'",
			},
			[]string{"orders", "payments"},
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
		assert.NotContains(t, backupPrivileges.EffectivePrivileges, "TRIGGER")
	})

	t.Run("falls back to schema grants when the schema has no tables", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{"GRANT SELECT, SHOW VIEW ON `appdb`.* TO 'backup'@'%'"},
			nil,
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
	})

	t.Run("reports the missing SELECT when the role holds nothing", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{"GRANT USAGE ON *.* TO 'backup'@'%'"},
			[]string{"orders"},
		))

		require.Len(t, backupPrivileges.Missing, 1)
		assert.Equal(t, "SELECT", backupPrivileges.Missing[0].Privilege)
		assert.Equal(t, GrantScopeSchema, backupPrivileges.Missing[0].Scope)
		assert.False(t, backupPrivileges.IsSufficientForDump())
	})

	t.Run("omits EVENT when all privileges are table scoped", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{"GRANT ALL PRIVILEGES ON `appdb`.`orders` TO 'backup'@'%'"},
			[]string{"orders"},
		))

		assert.True(t, backupPrivileges.IsSufficientForDump())
		assert.NotContains(t, backupPrivileges.EffectivePrivileges, "EVENT")
	})
}

func TestNewInsufficiencyError(t *testing.T) {
	t.Parallel()

	t.Run("suggests a schema wide grant when a privilege is missing on some tables", func(t *testing.T) {
		t.Parallel()

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{
				"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
				"GRANT SELECT ON `appdb`.`orders` TO 'backup'@'%'",
			},
			[]string{"orders", "payments"},
		))

		err := backupPrivileges.NewInsufficiencyError()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "`appdb`.`payments`")
		assert.NotContains(t, err.Error(), "`appdb`.`orders`")
		assert.Contains(t, err.Error(), "GRANT SELECT ON `appdb`.* TO 'backup'@'%';")
	})

	t.Run("caps the table list when many tables are missing SELECT", func(t *testing.T) {
		t.Parallel()

		var dumpedTables []string
		for tableIndex := range 9 {
			dumpedTables = append(dumpedTables, fmt.Sprintf("table_%d", tableIndex))
		}

		backupPrivileges := GetBackupPrivileges(newBackupPrivilegesSpec(
			[]string{
				"GRANT SHOW VIEW ON `appdb`.* TO 'backup'@'%'",
				"GRANT SELECT ON `appdb`.`table_0` TO 'backup'@'%'",
			},
			dumpedTables,
		))

		err := backupPrivileges.NewInsufficiencyError()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "and 3 more")
	})
}
