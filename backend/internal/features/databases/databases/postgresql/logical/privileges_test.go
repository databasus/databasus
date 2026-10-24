package postgresql_logical

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

// The version's subtests share one server (docs/testing.md, "Isolation"), so every case works in a schema of its own
// and scopes the model to it. Without that, tables another subtest left in public would join the
// dump scope and decide the outcome.
type backupPrivilegeFixture struct {
	Container  *PostgresContainer
	SchemaName string
	RoleName   string
	Model      *PostgresqlLogicalDatabase
}

func createBackupPrivilegeFixture(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
	schemaPrefix string,
) backupPrivilegeFixture {
	t.Helper()

	container := connectToPostgresEndpoint(t, endpoint)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	schemaName := fmt.Sprintf("%s_%s", schemaPrefix, suffix)
	roleName := fmt.Sprintf("backup_role_%s", suffix)
	rolePassword := "backuprolepassword123"

	for _, statement := range []string{
		fmt.Sprintf(`CREATE SCHEMA %s`, schemaName),
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, roleName, rolePassword),
		fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, container.Database, roleName),
	} {
		_, err := container.DB.Exec(statement)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schemaName))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s"`, roleName))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, roleName))
	})

	return backupPrivilegeFixture{
		Container:  container,
		SchemaName: schemaName,
		RoleName:   roleName,
		Model: &PostgresqlLogicalDatabase{
			Version:        tools.GetPostgresqlVersionEnum(version),
			Host:           container.Host,
			Port:           container.Port,
			Username:       roleName,
			Password:       rolePassword,
			Database:       &container.Database,
			SslMode:        postgresql_shared.PostgresSslModeDisable,
			IncludeSchemas: []string{schemaName},
			CpuCount:       1,
		},
	}
}

func (f backupPrivilegeFixture) executeStatements(t *testing.T, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := f.Container.DB.Exec(statement)
		require.NoError(t, err)
	}
}

func (f backupPrivilegeFixture) testConnection() error {
	return f.Model.TestConnection(slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)
}

// Pins issue #749: a role holding SELECT on part of the dump scope used to pass the connection
// test and only fail once pg_dump reached the first unreadable table.
func testConnectionSelectMissingOnSomeTables(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_partial")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.granted_orders (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`CREATE TABLE %s.ungranted_payments (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(
			`GRANT SELECT ON %s.granted_orders TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(t, err.Error(), fixture.SchemaName+".ungranted_payments")
	assert.NotContains(t, err.Error(), fixture.SchemaName+".granted_orders")
}

func testConnectionEveryTableAndSequenceReadable(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_full")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
		fmt.Sprintf(`CREATE SEQUENCE %s.standalone_counter`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(
			`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
		fmt.Sprintf(
			`GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
	)

	assert.NoError(t, fixture.testConnection())
}

func testConnectionSelectMissingOnSequence(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_sequence")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(
			`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), fixture.SchemaName+".orders_id_seq")
}

func testConnectionUnreadableTableExcluded(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_excluded")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.kept (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`CREATE TABLE %s.skipped_audit (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(`GRANT SELECT ON %s.kept TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	fixture.Model.ExcludeTables = []string{fixture.SchemaName + ".skipped_audit"}

	assert.NoError(t, fixture.testConnection())
}

// pg_dump still locks a table whose data it skips, so excluding the data must not lift the check.
func testConnectionUnreadableTableDataExcluded(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_data_excl")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.kept (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`CREATE TABLE %s.logs (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(`GRANT SELECT ON %s.kept TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	fixture.Model.ExcludeTableData = []string{fixture.SchemaName + ".logs"}

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), fixture.SchemaName+".logs")
	assert.Contains(t, err.Error(), "excluding only a table's data still requires SELECT")
}

// pg_dump matches an unqualified pattern only against tables visible on the default search path,
// so it skips public's table but still dumps and locks a same-named table in a schema off that path.
// The fixture schema sorts before public, so its table stays among the names the error lists.
func testConnectionUnqualifiedExclusionDoesNotReachTableOffSearchPath(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_unqualified")
	sameNamedTableName := "logs_" + strings.TrimPrefix(fixture.SchemaName, "perm_unqualified_")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE public.%s (id INT)`, sameNamedTableName),
		fmt.Sprintf(`CREATE TABLE %s.%s (id INT)`, fixture.SchemaName, sameNamedTableName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	t.Cleanup(func() {
		_, _ = fixture.Container.DB.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS public.%s`, sameNamedTableName))
	})

	fixture.Model.IncludeSchemas = []string{fixture.SchemaName, "public"}
	fixture.Model.ExcludeTables = []string{sameNamedTableName}

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), fixture.SchemaName+"."+sameNamedTableName)
	assert.NotContains(t, err.Error(), "public."+sameNamedTableName)
}

// pg_dump receives the exclude list normalized, so an entry pasted with a leading newline still
// excludes the table and must not be re-demanded by the privilege check.
func testConnectionUnreadableTableExcludedWithUnnormalizedEntry(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_dirty_excl")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.kept (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`CREATE TABLE %s.skipped_audit (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(`GRANT SELECT ON %s.kept TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	fixture.Model.ExcludeTables = []string{"\n" + fixture.SchemaName + ".skipped_audit"}

	assert.NoError(t, fixture.testConnection())
}

func testConnectionUnreadableTableOutsideIncludedSchemas(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_scoped_in")
	unscopedSchemaName := fixture.SchemaName + "_unscoped"

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE SCHEMA %s`, unscopedSchemaName),
		fmt.Sprintf(`CREATE TABLE %s.unreadable (id INT)`, unscopedSchemaName),
		fmt.Sprintf(`CREATE TABLE %s.readable (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(`GRANT SELECT ON %s.readable TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	t.Cleanup(func() {
		_, _ = fixture.Container.DB.Exec(
			fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, unscopedSchemaName),
		)
	})

	assert.NoError(t, fixture.testConnection())
}

func testConnectionSchemaUsageMissing(t *testing.T, endpoint containers.Endpoint, version string) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_no_usage")

	fixture.executeStatements(t,
		fmt.Sprintf(`CREATE TABLE %s.orders (id INT)`, fixture.SchemaName),
		fmt.Sprintf(`GRANT SELECT ON %s.orders TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "USAGE")
	assert.Contains(t, err.Error(), fixture.SchemaName)
}

func testConnectionRoleOwnsEveryTable(t *testing.T, endpoint containers.Endpoint, version string) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_owned")

	fixture.executeStatements(t,
		fmt.Sprintf(
			`ALTER SCHEMA %s OWNER TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
		fmt.Sprintf(`CREATE TABLE %s.orders (id SERIAL PRIMARY KEY)`, fixture.SchemaName),
		fmt.Sprintf(
			`ALTER TABLE %s.orders OWNER TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
		fmt.Sprintf(
			`ALTER SEQUENCE %s.orders_id_seq OWNER TO "%s"`,
			fixture.SchemaName,
			fixture.RoleName,
		),
	)

	assert.NoError(t, fixture.testConnection())
}

func testConnectionNoRelationInScope(t *testing.T, endpoint containers.Endpoint, version string) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_empty")

	assert.NoError(t, fixture.testConnection())
}

func testConnectionPartitionUnreadable(t *testing.T, endpoint containers.Endpoint, version string) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_partitioned")

	fixture.executeStatements(t,
		fmt.Sprintf(
			`CREATE TABLE %s.events (id INT) PARTITION BY RANGE (id)`,
			fixture.SchemaName,
		),
		fmt.Sprintf(
			`CREATE TABLE %s.events_2026 PARTITION OF %s.events FOR VALUES FROM (0) TO (1000)`,
			fixture.SchemaName,
			fixture.SchemaName,
		),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
		fmt.Sprintf(`GRANT SELECT ON %s.events TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), fixture.SchemaName+".events_2026")
}

// pg_dump emits no data for an extension member, so requiring SELECT on one would reject
// databases that back up fine today (pg_cron, pg_partman, PostGIS topology).
func testConnectionUnreadableTableBelongsToExtension(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version, "perm_extension")

	fixture.executeStatements(t,
		`CREATE EXTENSION IF NOT EXISTS hstore`,
		fmt.Sprintf(`CREATE TABLE %s.extension_owned (id INT)`, fixture.SchemaName),
		fmt.Sprintf(
			`ALTER EXTENSION hstore ADD TABLE %s.extension_owned`,
			fixture.SchemaName,
		),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, fixture.SchemaName, fixture.RoleName),
	)

	t.Cleanup(func() {
		_, _ = fixture.Container.DB.Exec(fmt.Sprintf(
			`ALTER EXTENSION hstore DROP TABLE %s.extension_owned`,
			fixture.SchemaName,
		))
	})

	assert.NoError(t, fixture.testConnection())
}
