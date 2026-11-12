package physical

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type historyPrivilegeProbe struct {
	TableName string
	Bystander testRole
}

func createHistoryPrivilegeProbe(t *testing.T, server testServer) historyPrivilegeProbe {
	t.Helper()

	bystander := server.createLoginRole(t, "")
	tableName := "public.history_privileges_" + newNameSuffix()
	execStatements(t, server.connectAs(t, getAdminRole()),
		fmt.Sprintf("CREATE TABLE %s (id int)", tableName),
		fmt.Sprintf(`GRANT SELECT ON %s TO "%s"`, tableName, bystander.Name),
		fmt.Sprintf(`GRANT pg_monitor TO "%s"`, bystander.Name),
	)

	return historyPrivilegeProbe{TableName: tableName, Bystander: bystander}
}

type historyRolePrivileges struct {
	Name               string
	IsSuperuser        bool
	CanCreateRole      bool
	CanCreateDatabase  bool
	CanBypassRLS       bool
	CanReplicate       bool
	CanLogin           bool
	CanInherit         bool
	Memberships        []string
	FunctionPrivileges []string
	TablePrivileges    []string
}

func getHistoryRolePrivileges(
	t *testing.T,
	administrator *pgx.Conn,
	tableName string,
) map[string]historyRolePrivileges {
	t.Helper()

	rows, err := administrator.Query(t.Context(), `
		SELECT source_role.rolname, source_role.rolsuper, source_role.rolcreaterole,
		       source_role.rolcreatedb, source_role.rolbypassrls, source_role.rolreplication,
		       source_role.rolcanlogin, source_role.rolinherit,
		       ARRAY(
		           SELECT format('%s:%s:%s:%s:%s', granted_role.rolname, grantor.rolname,
		                         membership.admin_option, membership.inherit_option, membership.set_option)
		           FROM pg_auth_members AS membership
		           JOIN pg_roles AS granted_role ON granted_role.oid = membership.roleid
		           JOIN pg_roles AS grantor ON grantor.oid = membership.grantor
		           WHERE membership.member = source_role.oid
		           ORDER BY granted_role.rolname, grantor.rolname
		       ),
		       ARRAY(
		           SELECT function.oid::regprocedure::text || '=' ||
		                  has_function_privilege(source_role.oid, function.oid, 'EXECUTE')::text
		           FROM pg_proc AS function
		           JOIN pg_namespace AS namespace ON namespace.oid = function.pronamespace
		           WHERE namespace.nspname = 'pg_catalog'
		             AND function.proname IN ('pg_read_file', 'pg_read_binary_file', 'pg_stat_file',
		                                      'pg_ls_dir', 'pg_switch_wal')
		           ORDER BY function.oid::regprocedure::text
		       ),
		       ARRAY(
		           SELECT privilege || '=' || has_table_privilege(source_role.oid, $1::regclass, privilege)::text
		           FROM unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE', 'REFERENCES', 'TRIGGER']) AS privilege
		           ORDER BY privilege
		       ) || ARRAY['COLUMN_SELECT=' || has_any_column_privilege(source_role.oid, $1::regclass, 'SELECT')::text]
		FROM pg_roles AS source_role
		ORDER BY source_role.rolname
	`, tableName)
	require.NoError(t, err)
	defer rows.Close()

	privileges := make(map[string]historyRolePrivileges)
	for rows.Next() {
		var rolePrivileges historyRolePrivileges
		require.NoError(t, rows.Scan(
			&rolePrivileges.Name,
			&rolePrivileges.IsSuperuser,
			&rolePrivileges.CanCreateRole,
			&rolePrivileges.CanCreateDatabase,
			&rolePrivileges.CanBypassRLS,
			&rolePrivileges.CanReplicate,
			&rolePrivileges.CanLogin,
			&rolePrivileges.CanInherit,
			&rolePrivileges.Memberships,
			&rolePrivileges.FunctionPrivileges,
			&rolePrivileges.TablePrivileges,
		))
		privileges[rolePrivileges.Name] = rolePrivileges
	}
	require.NoError(t, rows.Err())

	return privileges
}
