package logical

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// backupPrivilegeFixture is a database with a schema the dump is scoped to and a role that may
// connect but holds nothing in that schema until a scenario grants it.
type backupPrivilegeFixture struct {
	Database   testDatabase
	SchemaName string
	Role       testRole
	Check      DumpCheck
}

func createBackupPrivilegeFixture(t *testing.T, server testServer, schemaPrefix string) *backupPrivilegeFixture {
	t.Helper()

	database := server.createDatabase(t)
	schemaName := schemaPrefix + "_" + newNameSuffix()

	database.exec(t, fmt.Sprintf(`CREATE SCHEMA %s`, schemaName))

	return &backupPrivilegeFixture{
		Database:   database,
		SchemaName: schemaName,
		Role:       database.createLoginRole(t, "backup_role"),
		Check: DumpCheck{
			Filter:             DumpFilter{IncludeSchemas: []string{schemaName}},
			IsSkipUserMappings: false,
		},
	}
}

func (f *backupPrivilegeFixture) testConnection(t *testing.T) error {
	t.Helper()

	_, err := f.Database.newSource(f.Role).TestConnection(t.Context(), newTestLogger(t), f.Check)

	return err
}

func runDumpPrivilegeScenarios(t *testing.T, server testServer) {
	t.Helper()

	// Pins issue #749: a role holding SELECT on part of the dump scope used to pass the connection
	// test and only fail once pg_dump reached the first unreadable table.
	t.Run("returns error naming only unreadable table when select is missing on some tables", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_partial")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE %s.granted_orders (id INT)`, fixture.SchemaName),
			fmt.Sprintf(`CREATE TABLE %s.ungranted_payments (id INT)`, fixture.SchemaName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON %s.granted_orders TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), fixture.SchemaName+".ungranted_payments")
		assert.NotContains(t, err.Error(), fixture.SchemaName+".granted_orders")
	})

	t.Run("succeeds when every table and sequence is readable", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_full")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
			fmt.Sprintf(`CREATE SEQUENCE %s.standalone_counter`, fixture.SchemaName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		assert.NoError(t, fixture.testConnection(t))
	})

	t.Run("returns error naming sequence when select is missing on it", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_sequence")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), fixture.SchemaName+".orders_id_seq")
	})

	t.Run("succeeds when unreadable table is excluded", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_excluded")
		fixture.grantSelectOnKeptTableOnly(t, "skipped_audit")

		fixture.Check.Filter.ExcludeTablePatterns = []string{fixture.SchemaName + ".skipped_audit"}

		assert.NoError(t, fixture.testConnection(t))
	})

	// pg_dump receives the exclude list normalized, so an entry pasted with a leading newline still
	// excludes the table and must not be re-demanded by the privilege check.
	t.Run("succeeds when unreadable table is excluded by an unnormalized entry", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_dirty_excl")
		fixture.grantSelectOnKeptTableOnly(t, "skipped_audit")

		fixture.Check.Filter.ExcludeTablePatterns = []string{"\n" + fixture.SchemaName + ".skipped_audit"}

		assert.NoError(t, fixture.testConnection(t))
	})

	// pg_dump still locks a table whose data it skips, so excluding the data must not lift the check.
	t.Run("returns error naming table when only its data is excluded", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_data_excl")
		fixture.grantSelectOnKeptTableOnly(t, "logs")

		fixture.Check.Filter.ExcludeTableDataPatterns = []string{fixture.SchemaName + ".logs"}

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), fixture.SchemaName+".logs")
		assert.Contains(t, err.Error(), "excluding only a table's data still requires SELECT")
	})

	// pg_dump matches an unqualified pattern only against tables visible on the default search
	// path, so it skips public's table but still dumps and locks a same-named table in a schema off
	// that path. The fixture schema sorts before public, so its table stays among the names the
	// error lists.
	t.Run("returns error naming table off search path when unqualified exclusion names it", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_unqualified")
		sameNamedTableName := "logs_" + strings.TrimPrefix(fixture.SchemaName, "perm_unqualified_")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE public.%s (id INT)`, sameNamedTableName),
			fmt.Sprintf(`CREATE TABLE %s.%s (id INT)`, fixture.SchemaName, sameNamedTableName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		fixture.Check.Filter.IncludeSchemas = []string{fixture.SchemaName, "public"}
		fixture.Check.Filter.ExcludeTablePatterns = []string{sameNamedTableName}

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), fixture.SchemaName+"."+sameNamedTableName)
		assert.NotContains(t, err.Error(), "public."+sameNamedTableName)
	})

	t.Run("succeeds when unreadable table is outside included schemas", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_scoped_in")
		unscopedSchemaName := fixture.SchemaName + "_unscoped"

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE SCHEMA %s`, unscopedSchemaName),
			fmt.Sprintf(`CREATE TABLE %s.unreadable (id INT)`, unscopedSchemaName),
			fmt.Sprintf(`CREATE TABLE %s.readable (id INT)`, fixture.SchemaName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON %s.readable TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		assert.NoError(t, fixture.testConnection(t))
	})

	t.Run("returns usage error naming schema when schema usage is missing", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_no_usage")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE %s.orders (id INT)`, fixture.SchemaName),
			fmt.Sprintf(`GRANT SELECT ON %s.orders TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "USAGE")
		assert.Contains(t, err.Error(), fixture.SchemaName)
	})

	t.Run("succeeds when role owns every table", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_owned")

		fixture.Database.exec(t,
			fmt.Sprintf(`ALTER SCHEMA %s OWNER TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
			fmt.Sprintf(`ALTER TABLE %s.orders OWNER TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`ALTER SEQUENCE %s.orders_id_seq OWNER TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		assert.NoError(t, fixture.testConnection(t))
	})

	t.Run("succeeds when no relation is in scope", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_empty")

		assert.NoError(t, fixture.testConnection(t))
	})

	t.Run("returns error naming partition when partition is unreadable", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_partitioned")

		fixture.Database.exec(t,
			fmt.Sprintf(`CREATE TABLE %s.events (id INT) PARTITION BY RANGE (id)`, fixture.SchemaName),
			fmt.Sprintf(
				`CREATE TABLE %s.events_2026 PARTITION OF %s.events FOR VALUES FROM (0) TO (1000)`,
				fixture.SchemaName, fixture.SchemaName,
			),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
			fmt.Sprintf(`GRANT SELECT ON %s.events TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), fixture.SchemaName+".events_2026")
	})

	// pg_dump emits no data for an extension member, so requiring SELECT on one would reject
	// databases that back up fine today (pg_cron, pg_partman, PostGIS topology).
	t.Run("succeeds when unreadable table belongs to an extension", func(t *testing.T) {
		t.Parallel()
		fixture := createBackupPrivilegeFixture(t, server, "perm_extension")

		fixture.Database.exec(t,
			`CREATE EXTENSION IF NOT EXISTS hstore`,
			fmt.Sprintf(`CREATE TABLE %s.extension_owned (id INT)`, fixture.SchemaName),
			fmt.Sprintf(`ALTER EXTENSION hstore ADD TABLE %s.extension_owned`, fixture.SchemaName),
			fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		)

		assert.NoError(t, fixture.testConnection(t))
	})

	t.Run("returns user mapping error when mapping options are unreadable", func(t *testing.T) {
		t.Parallel()
		fixture := createUnreadableUserMappingFixture(t, server)

		err := fixture.testConnection(t)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user mapping")
	})

	t.Run("succeeds when mapping options are unreadable with user mappings skipped", func(t *testing.T) {
		t.Parallel()
		fixture := createUnreadableUserMappingFixture(t, server)
		fixture.Check.IsSkipUserMappings = true

		assert.NoError(t, fixture.testConnection(t))
	})
}

// grantSelectOnKeptTableOnly creates a readable table "kept" and an unreadable table next to it.
func (f *backupPrivilegeFixture) grantSelectOnKeptTableOnly(t *testing.T, unreadableTableName string) {
	t.Helper()

	f.Database.exec(t,
		fmt.Sprintf(`CREATE TABLE %s.kept (id INT)`, f.SchemaName),
		fmt.Sprintf(`CREATE TABLE %s.%s (id INT)`, f.SchemaName, unreadableTableName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, f.SchemaName, f.Role.Name),
		fmt.Sprintf(`GRANT SELECT ON %s.kept TO "%s"`, f.SchemaName, f.Role.Name),
	)
}

// createUnreadableUserMappingFixture sets up a superuser-owned user mapping and a role that passes
// the read checks yet cannot read the mapping's options. postgres_fdw stands in for oracle_fdw,
// which is not installable in CI.
func createUnreadableUserMappingFixture(t *testing.T, server testServer) *backupPrivilegeFixture {
	t.Helper()

	fixture := createBackupPrivilegeFixture(t, server, "um_test_schema")
	serverName := "um_test_srv_" + newNameSuffix()

	fixture.Database.exec(t,
		fmt.Sprintf(`CREATE TABLE %s.um_test_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`, fixture.SchemaName),
		fmt.Sprintf(`INSERT INTO %s.um_test_table (data) VALUES ('row1')`, fixture.SchemaName),
		`CREATE EXTENSION IF NOT EXISTS postgres_fdw`,
		fmt.Sprintf(
			`CREATE SERVER %s FOREIGN DATA WRAPPER postgres_fdw OPTIONS (host 'localhost', dbname 'postgres')`,
			serverName,
		),
		fmt.Sprintf(
			`CREATE USER MAPPING FOR CURRENT_USER SERVER %s OPTIONS ("user" 'remote', password 'secret')`,
			serverName,
		),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
		fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.Role.Name),
	)

	return fixture
}
