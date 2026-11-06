package logical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runRLSPrivilegeScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("refuses complete row access when select role is restricted by rls", func(t *testing.T) {
		t.Parallel()
		fixture := createRLSPrivilegeFixture(t, server)

		err := fixture.testConnection(t)

		require.ErrorIs(t, err, ErrRowLevelSecurity)
		assert.Contains(t, err.Error(), fixture.Role.Name)
		assert.Contains(t, err.Error(), fixture.SchemaName+".tenant_rows")
		assert.Contains(t, err.Error(), "RLS")
		assert.NotContains(t, err.Error(), fixture.Role.Password)
	})

	for _, scenario := range []struct {
		name                  string
		statements            string
		filter                DumpFilter
		isRestricted          bool
		isSelectMissing       bool
		isAdmin               bool
		hasInheritedOwnership bool
	}{
		{name: "succeeds with bypassrls even when rls is forced", statements: `ALTER ROLE "%[2]s" BYPASSRLS; ALTER TABLE %[1]s.tenant_rows FORCE ROW LEVEL SECURITY`},
		{name: "succeeds with superuser even when rls is forced", statements: `ALTER TABLE %[1]s.tenant_rows FORCE ROW LEVEL SECURITY`, isAdmin: true},
		{name: "succeeds with table ownership without forced rls", statements: `ALTER TABLE %[1]s.tenant_rows OWNER TO "%[2]s"`},
		{name: "succeeds with inherited table ownership", statements: `CREATE ROLE "%[2]s_owner"; GRANT "%[2]s_owner" TO "%[2]s"; ALTER TABLE %[1]s.tenant_rows OWNER TO "%[2]s_owner"`, hasInheritedOwnership: true},
		{name: "refuses owner when rls is forced", statements: `ALTER TABLE %[1]s.tenant_rows OWNER TO "%[2]s"; ALTER TABLE %[1]s.tenant_rows FORCE ROW LEVEL SECURITY`, isRestricted: true},
		{name: "refuses inherited owner when rls is forced", statements: `CREATE ROLE "%[2]s_owner"; GRANT "%[2]s_owner" TO "%[2]s"; ALTER TABLE %[1]s.tenant_rows OWNER TO "%[2]s_owner"; ALTER TABLE %[1]s.tenant_rows FORCE ROW LEVEL SECURITY`, isRestricted: true, hasInheritedOwnership: true},
		{name: "refuses role when policy permits every row", statements: `ALTER POLICY tenant_access ON %[1]s.tenant_rows USING (true)`, isRestricted: true},
		{name: "succeeds when rls table lies outside included schemas", filter: DumpFilter{IncludeSchemas: []string{"public"}}},
		{name: "succeeds with whole table exclusion", filter: DumpFilter{ExcludeTablePatterns: []string{"*.tenant_rows"}}},
		{name: "succeeds with table data exclusion", filter: DumpFilter{ExcludeTableDataPatterns: []string{"*.tenant_rows"}}},
		{name: "requires select when only table data is excluded", statements: `REVOKE SELECT ON %[1]s.tenant_rows FROM "%[2]s"`, filter: DumpFilter{ExcludeTableDataPatterns: []string{"*.tenant_rows"}}, isSelectMissing: true},
		{name: "succeeds with unqualified data exclusion on default search path", statements: `ALTER ROLE "%[2]s" SET search_path TO %[1]s`, filter: DumpFilter{ExcludeTableDataPatterns: []string{"tenant_rows"}}},
		{name: "refuses table outside search path with unqualified data exclusion", statements: `CREATE TABLE public.tenant_rows (id INT); GRANT SELECT ON public.tenant_rows TO "%[2]s"`, filter: DumpFilter{ExcludeTableDataPatterns: []string{"tenant_rows"}}, isRestricted: true},
		{name: "succeeds when only partition parent has rls", statements: `DROP TABLE %[1]s.tenant_rows; CREATE TABLE %[1]s.tenant_rows (id INT) PARTITION BY RANGE(id); CREATE TABLE %[1]s.tenant_leaf PARTITION OF %[1]s.tenant_rows FOR VALUES FROM (0) TO (10); ALTER TABLE %[1]s.tenant_rows ENABLE ROW LEVEL SECURITY; GRANT SELECT ON ALL TABLES IN SCHEMA %[1]s TO "%[2]s"`},
		{name: "refuses when dumped partition has rls", statements: `DROP TABLE %[1]s.tenant_rows; CREATE TABLE %[1]s.tenant_parent (id INT) PARTITION BY RANGE(id); CREATE TABLE %[1]s.tenant_rows PARTITION OF %[1]s.tenant_parent FOR VALUES FROM (0) TO (10); ALTER TABLE %[1]s.tenant_rows ENABLE ROW LEVEL SECURITY; GRANT SELECT ON ALL TABLES IN SCHEMA %[1]s TO "%[2]s"`, isRestricted: true},
		{name: "refuses leaf when only partition parent is excluded", statements: `DROP TABLE %[1]s.tenant_rows; CREATE TABLE %[1]s.tenant_parent (id INT) PARTITION BY RANGE(id); CREATE TABLE %[1]s.tenant_rows PARTITION OF %[1]s.tenant_parent FOR VALUES FROM (0) TO (10); ALTER TABLE %[1]s.tenant_rows ENABLE ROW LEVEL SECURITY; GRANT SELECT ON ALL TABLES IN SCHEMA %[1]s TO "%[2]s"`, filter: DumpFilter{ExcludeTablePatterns: []string{"*.tenant_parent"}}, isRestricted: true},
		{name: "succeeds when restricted partition data is excluded", statements: `DROP TABLE %[1]s.tenant_rows; CREATE TABLE %[1]s.tenant_parent (id INT) PARTITION BY RANGE(id); CREATE TABLE %[1]s.tenant_rows PARTITION OF %[1]s.tenant_parent FOR VALUES FROM (0) TO (10); ALTER TABLE %[1]s.tenant_rows ENABLE ROW LEVEL SECURITY; GRANT SELECT ON ALL TABLES IN SCHEMA %[1]s TO "%[2]s"`, filter: DumpFilter{ExcludeTableDataPatterns: []string{"*.tenant_rows"}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			fixture := createRLSPrivilegeFixture(t, server)
			if scenario.statements != "" {
				fixture.Database.exec(t, fmt.Sprintf(scenario.statements, fixture.SchemaName, fixture.Role.Name))
			}

			if scenario.hasInheritedOwnership {
				fixture.Database.dropRoleOnCleanup(fixture.Role.Name + "_owner")
			}

			fixture.Check.Filter = scenario.filter
			if scenario.isAdmin {
				fixture.Role = getAdminRole()
			}

			err := fixture.testConnection(t)

			switch {
			case scenario.isRestricted:
				require.ErrorIs(t, err, ErrRowLevelSecurity)
				assert.Contains(t, err.Error(), fixture.Role.Name)
				assert.Contains(t, err.Error(), fixture.SchemaName+".tenant_rows")
				assert.NotContains(t, err.Error(), fixture.Role.Password)
			case scenario.isSelectMissing:
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Missing SELECT")
				require.NotErrorIs(t, err, ErrRowLevelSecurity)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func createRLSPrivilegeFixture(t *testing.T, server testServer) *backupPrivilegeFixture {
	t.Helper()

	fixture := createBackupPrivilegeFixture(t, server, "rls")
	fixture.Database.exec(t,
		fmt.Sprintf(`CREATE TABLE %s.tenant_rows (id INT, tenant_id INT)`, fixture.SchemaName),
		fmt.Sprintf(`INSERT INTO %s.tenant_rows VALUES (1, 1), (2, 2)`, fixture.SchemaName),
		fmt.Sprintf(`ALTER TABLE %s.tenant_rows ENABLE ROW LEVEL SECURITY`, fixture.SchemaName),
		fmt.Sprintf(`CREATE POLICY tenant_access ON %s.tenant_rows USING (tenant_id = 1)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
	)

	return fixture
}
