package mariadb

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

// The version's subtests share one server (docs/testing.md, "Isolation"), so every name carries a suffix and the
// tables a case creates are dropped again before the next one runs.
type backupPrivilegeFixture struct {
	Container *MariadbContainer
	Suffix    string
	Username  string
	RoleName  string
	Model     *MariadbDatabase
}

func createBackupPrivilegeFixture(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) backupPrivilegeFixture {
	t.Helper()

	container := connectToMariadbEndpoint(t, endpoint, version)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	username := fmt.Sprintf("backup_%s", suffix)
	password := "backuprolepassword123"

	_, err := container.DB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		username,
		password,
	))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	})

	return backupPrivilegeFixture{
		Container: container,
		Suffix:    suffix,
		Username:  username,
		RoleName:  fmt.Sprintf("backup_role_%s", suffix),
		Model: &MariadbDatabase{
			Version:  version,
			Host:     container.Host,
			Port:     container.Port,
			Username: username,
			Password: password,
			Database: &container.Database,
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

// A MariaDB role carries no host part, unlike a MySQL one.
func (f backupPrivilegeFixture) createRole(t *testing.T) {
	t.Helper()

	f.executeStatements(t, fmt.Sprintf("CREATE ROLE %s", f.RoleName))

	t.Cleanup(func() {
		_, _ = f.Container.DB.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s", f.RoleName))
	})
}

// Every table in the schema joins the dump set, so a case granting per table has to drop the
// tables it created or it would decide the outcome of the cases that follow.
func (f backupPrivilegeFixture) createTables(t *testing.T, tableNames ...string) {
	t.Helper()

	for _, tableName := range tableNames {
		f.executeStatements(t, fmt.Sprintf("CREATE TABLE `%s` (id INT PRIMARY KEY)", tableName))

		t.Cleanup(func() {
			_, _ = f.Container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
		})
	}
}

func (f backupPrivilegeFixture) createView(t *testing.T, viewName, baseTableName string) {
	t.Helper()

	f.executeStatements(t, fmt.Sprintf(
		"CREATE VIEW `%s` AS SELECT id FROM `%s`",
		viewName,
		baseTableName,
	))

	t.Cleanup(func() {
		_, _ = f.Container.DB.Exec(fmt.Sprintf("DROP VIEW IF EXISTS `%s`", viewName))
	})
}

func (f backupPrivilegeFixture) testConnection() error {
	return f.Model.TestConnection(slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)
}

func Test_QuoteRoleName_WithEmbeddedBackquote_EscapesIt(t *testing.T) {
	assert.Equal(t, "`plain_role`", quoteRoleName("plain_role"))
	assert.Equal(t, "`odd``role`", quoteRoleName("odd`role"))
}

func testTestConnectionBackupPrivilegesFromActiveRole(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	fixture.createRole(t)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO %s",
			fixture.Container.Database,
			fixture.RoleName,
		),
		fmt.Sprintf("GRANT %s TO '%s'@'%%'", fixture.RoleName, fixture.Username),
		fmt.Sprintf("SET DEFAULT ROLE %s FOR '%s'@'%%'", fixture.RoleName, fixture.Username),
		"FLUSH PRIVILEGES",
	)

	require.NoError(t, fixture.testConnection())
	assert.Contains(t, fixture.Model.Privileges, "SELECT")
	assert.Contains(t, fixture.Model.Privileges, "SHOW VIEW")
}

// The account's SELECT arrives only through a role granted to its active role, so this is what
// pins that a single SHOW GRANTS FOR the active role really carries the transitively inherited
// grants: read only the user's own grants and the check falsely reports missing privileges.
func testTestConnectionBackupPrivilegesFromNestedRole(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	fixture.createRole(t)
	nestedRoleName := "nested_role_" + fixture.Suffix

	fixture.executeStatements(t, fmt.Sprintf("CREATE ROLE %s", nestedRoleName))
	t.Cleanup(func() {
		_, _ = fixture.Container.DB.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s", nestedRoleName))
	})

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.* TO %s",
			fixture.Container.Database,
			nestedRoleName,
		),
		fmt.Sprintf("GRANT %s TO %s", nestedRoleName, fixture.RoleName),
		fmt.Sprintf("GRANT %s TO '%s'@'%%'", fixture.RoleName, fixture.Username),
		fmt.Sprintf("SET DEFAULT ROLE %s FOR '%s'@'%%'", fixture.RoleName, fixture.Username),
		"FLUSH PRIVILEGES",
	)

	require.NoError(t, fixture.testConnection())
	assert.Contains(t, fixture.Model.Privileges, "SELECT")
	assert.Contains(t, fixture.Model.Privileges, "SHOW VIEW")
}

// mariadb-dump logs in fresh and a granted-but-inactive role gives it nothing, so reading only the
// active role is what matches what the dump will have. The direct INSERT grant is what lets the
// account reach the schema at all and puts the table in the dump set.
func testTestConnectionRoleGrantedButNotActivated(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	fixture.createRole(t)
	blockedTableName := "blocked_" + fixture.Suffix

	fixture.createTables(t, blockedTableName)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT INSERT ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			blockedTableName,
			fixture.Username,
		),
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.* TO %s",
			fixture.Container.Database,
			fixture.RoleName,
		),
		fmt.Sprintf("GRANT %s TO '%s'@'%%'", fixture.RoleName, fixture.Username),
		"FLUSH PRIVILEGES",
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(
		t,
		err.Error(),
		fmt.Sprintf("SELECT on `%s`.*", fixture.Container.Database),
	)
}

func testTestConnectionTableGrantsCoverEveryVisibleTable(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	readableTableName := "readable_" + fixture.Suffix

	fixture.createTables(t, readableTableName)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			readableTableName,
			fixture.Username,
		),
		"FLUSH PRIVILEGES",
	)

	assert.NoError(t, fixture.testConnection())
}

func testTestConnectionVisibleTableLacksSelect(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	readableTableName := "readable_" + fixture.Suffix
	blockedTableName := "blocked_" + fixture.Suffix

	fixture.createTables(t, readableTableName, blockedTableName)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			readableTableName,
			fixture.Username,
		),
		fmt.Sprintf(
			"GRANT INSERT ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			blockedTableName,
			fixture.Username,
		),
		"FLUSH PRIVILEGES",
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(t, err.Error(), blockedTableName)
	assert.NotContains(t, err.Error(), readableTableName)
}

func testTestConnectionVisibleTableLackingSelectIsExcluded(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	readableTableName := "readable_" + fixture.Suffix
	blockedTableName := "blocked_" + fixture.Suffix

	fixture.createTables(t, readableTableName, blockedTableName)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			readableTableName,
			fixture.Username,
		),
		fmt.Sprintf(
			"GRANT INSERT ON `%s`.`%s` TO '%s'@'%%'",
			fixture.Container.Database,
			blockedTableName,
			fixture.Username,
		),
		"FLUSH PRIVILEGES",
	)

	fixture.Model.ExcludeTables = []string{blockedTableName}

	assert.NoError(t, fixture.testConnection())
}

// Nothing else in this package creates a view, so this is what pins readDumpedTables' TABLE_TYPE
// mapping: misclassify a view as a base table and SHOW VIEW silently stops being required.
func testTestConnectionViewLacksShowView(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	fixture := createBackupPrivilegeFixture(t, endpoint, version)
	baseTableName := "viewbase_" + fixture.Suffix
	viewName := "viewonly_" + fixture.Suffix

	fixture.createTables(t, baseTableName)
	fixture.createView(t, viewName, baseTableName)

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SELECT ON `%s`.* TO '%s'@'%%'",
			fixture.Container.Database,
			fixture.Username,
		),
		"FLUSH PRIVILEGES",
	)

	err := fixture.testConnection()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(t, err.Error(), "SHOW VIEW")

	fixture.executeStatements(t,
		fmt.Sprintf(
			"GRANT SHOW VIEW ON `%s`.* TO '%s'@'%%'",
			fixture.Container.Database,
			fixture.Username,
		),
		"FLUSH PRIVILEGES",
	)

	assert.NoError(t, fixture.testConnection())
}
