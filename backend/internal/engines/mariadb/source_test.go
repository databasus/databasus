package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	containers "databasus-backend/internal/lib/testcontainers"
)

type serverVersion struct {
	name    string
	version Version
	image   string
	// MariaDB grew roles in 10.0.5, so 5.5 has none.
	hasRoles bool
	// SHOW CREATE ROUTINE became a privilege of its own in 11.3.
	hasShowCreateRoutine bool
}

func getServerVersions() []serverVersion {
	return []serverVersion{
		{"MariaDB 5.5", Version55, "mariadb:5.5", false, false},
		{"MariaDB 10.1", Version101, "mariadb:10.1", true, false},
		{"MariaDB 10.6", Version106, "mariadb:10.6", true, false},
		{"MariaDB 10.11", Version1011, "mariadb:10.11", true, false},
		{"MariaDB 11.4", Version114, "mariadb:11.4", true, true},
		{"MariaDB 11.8", Version118, "mariadb:11.8", true, true},
		{"MariaDB 12.0", Version120, "mariadb:12.0", true, true},
		{"MariaDB 13.0", Version130, "mariadb:13.0", true, true},
	}
}

// testServer is one MariaDB container shared by the subtests of a version, so every database and
// account a subtest creates carries a suffix of its own.
type testServer struct {
	endpoint containers.Endpoint
	rootDB   *sql.DB
}

func startTestServer(t *testing.T, image string) testServer {
	t.Helper()

	endpoint := containers.StartMariadb(t, image)

	rootDB, err := sql.Open("mysql", fmt.Sprintf(
		"root:%s@tcp(%s:%d)/?parseTime=true",
		containers.MariadbRootPassword,
		endpoint.Host,
		endpoint.Port,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _ = rootDB.Close() })

	return testServer{endpoint: endpoint, rootDB: rootDB}
}

func (s testServer) exec(t *testing.T, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := s.rootDB.ExecContext(t.Context(), statement)
		require.NoError(t, err, statement)
	}
}

func (s testServer) createDatabase(t *testing.T, namePrefix string) string {
	t.Helper()

	databaseName := namePrefix + uuid.New().String()[:8]
	s.exec(t, fmt.Sprintf("CREATE DATABASE `%s`", databaseName))

	t.Cleanup(func() {
		_, _ = s.rootDB.ExecContext(
			context.WithoutCancel(t.Context()),
			fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", databaseName),
		)
	})

	return databaseName
}

func (s testServer) createTable(t *testing.T, databaseName, tableName string) {
	t.Helper()

	s.exec(t, fmt.Sprintf(
		"CREATE TABLE `%s`.`%s` (id INT AUTO_INCREMENT PRIMARY KEY, data VARCHAR(255) NOT NULL)",
		databaseName,
		tableName,
	))
}

// createUser creates an account with the given grants, each written with %[1]s for the database
// and %[2]s for the quoted account.
func (s testServer) createUser(t *testing.T, databaseName string, grants ...string) (string, string) {
	t.Helper()

	username := "user_" + uuid.New().String()[:8]
	password := "userpassword123"
	account := fmt.Sprintf("'%s'@'%%'", username)

	s.exec(t, fmt.Sprintf("CREATE USER %s IDENTIFIED BY '%s'", account, password))
	s.dropUserOnCleanup(t, username)

	for _, grant := range grants {
		s.exec(t, fmt.Sprintf(grant, databaseName, account))
	}

	s.exec(t, "FLUSH PRIVILEGES")

	return username, password
}

func (s testServer) dropUserOnCleanup(t *testing.T, username string) {
	t.Helper()

	t.Cleanup(func() {
		_, _ = s.rootDB.ExecContext(context.WithoutCancel(t.Context()), fmt.Sprintf("DROP USER '%s'@'%%'", username))
	})
}

func (s testServer) newSource(t *testing.T, username, password, databaseName string) *Source {
	t.Helper()

	return NewSource(Settings{
		Host:      s.endpoint.Host,
		Port:      s.endpoint.Port,
		Username:  username,
		Password:  password,
		Database:  databaseName,
		IsHTTPS:   false,
		SSHTunnel: nil,
	}, getClientTools(t))
}

func (s testServer) newRootSource(t *testing.T, databaseName string) *Source {
	t.Helper()

	return s.newSource(t, "root", containers.MariadbRootPassword, databaseName)
}

func (s testServer) connectAs(t *testing.T, username, password, databaseName string) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username,
		password,
		s.endpoint.Host,
		s.endpoint.Port,
		databaseName,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func getClientTools(t *testing.T) ClientTools {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)

	bundleRoot, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	return ClientTools{Root: bundleRoot}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

//nolint:tparallel // Each version's scenarios share one server, and a version stops before the next starts.
func TestSource(t *testing.T) {
	t.Parallel()

	for _, serverVersion := range getServerVersions() {
		// The scenarios of one server run one after another: MariaDB rebuilds its privilege cache on
		// every FLUSH PRIVILEGES and role grant, and a scenario that logs in during another one's
		// rebuild intermittently sees its host or its inherited role grants missing.
		t.Run(serverVersion.name, func(t *testing.T) {
			server := startTestServer(t, serverVersion.image)

			testTestConnection(t, server)
			testInspect(t, server, serverVersion.version)
			testShouldSuggestReadOnlyUser(t, server)
			testGetRawSizeMb(t, server)

			if serverVersion.hasRoles {
				testTestConnectionWithRoles(t, server)
			}

			if serverVersion.hasShowCreateRoutine {
				testShouldSuggestReadOnlyUserWithShowCreateRoutine(t, server)
			}
		})
	}
}

func testTestConnection(t *testing.T, server testServer) {
	t.Helper()

	t.Run("test connection returns insufficient permissions when the account can only insert", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "permission_test")
		// INSERT makes the schema and its tables visible without making them readable, which is
		// what the dump tool aborts on.
		username, password := server.createUser(t, databaseName, "GRANT INSERT ON `%s`.* TO %s")

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "SELECT")
	})

	t.Run("test connection succeeds with the backup grants on the schema", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "backup_test")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO %s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
	})

	t.Run("test connection succeeds with schema grants and a global process grant", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "privilege_test")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT, SHOW VIEW ON %s.* TO %s",
			"GRANT PROCESS ON *.* TO %[2]s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
	})

	// The underscore in the name is a LIKE wildcard in the grant, which the check must match the way
	// the server does.
	t.Run(
		"test connection records SELECT and SHOW VIEW with all privileges on an underscored schema",
		func(t *testing.T) {
			t.Parallel()
			databaseName := server.createDatabase(t, "test_all_db_")
			server.createTable(t, databaseName, "all_priv_test")
			username, password := server.createUser(t, databaseName, "GRANT ALL PRIVILEGES ON `%s`.* TO %s")

			inspection, err := server.newSource(t, username, password, databaseName).
				TestConnection(t.Context(), newTestLogger(), nil)

			require.NoError(t, err)
			assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SELECT")
			assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SHOW VIEW")
		},
	)

	t.Run("test connection succeeds when table grants cover every visible table", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "readable")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT, SHOW VIEW ON `%s`.`readable` TO %s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
	})

	t.Run("test connection names the table when a visible table lacks SELECT", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "readable")
		server.createTable(t, databaseName, "blocked")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT, SHOW VIEW ON `%s`.`readable` TO %s",
			"GRANT INSERT ON `%s`.`blocked` TO %s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "`blocked`")
		assert.NotContains(t, err.Error(), "`readable`")
	})

	t.Run("test connection succeeds when the table lacking SELECT is excluded", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "readable")
		server.createTable(t, databaseName, "blocked")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT, SHOW VIEW ON `%s`.`readable` TO %s",
			"GRANT INSERT ON `%s`.`blocked` TO %s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), []string{"blocked"})

		require.NoError(t, err)
	})

	// Nothing else here creates a view, so this is what pins the TABLE_TYPE mapping: misclassify a
	// view as a base table and SHOW VIEW silently stops being required.
	t.Run("test connection asks for SHOW VIEW until it is granted with a view in the schema", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "viewbase")
		server.exec(t, fmt.Sprintf("CREATE VIEW `%[1]s`.`viewonly` AS SELECT id FROM `%[1]s`.`viewbase`", databaseName))
		username, password := server.createUser(t, databaseName, "GRANT SELECT ON `%s`.* TO %s")
		source := server.newSource(t, username, password, databaseName)

		_, err := source.TestConnection(t.Context(), newTestLogger(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "SHOW VIEW")

		server.exec(t,
			fmt.Sprintf("GRANT SHOW VIEW ON `%s`.* TO '%s'@'%%'", databaseName, username),
			"FLUSH PRIVILEGES",
		)

		_, err = source.TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
	})

	t.Run("test connection returns an error without a database name", func(t *testing.T) {
		_, err := server.newRootSource(t, "").TestConnection(t.Context(), newTestLogger(), nil)

		require.ErrorIs(t, err, ErrDatabaseNameRequired)
	})
}

func testInspect(t *testing.T, server testServer, version Version) {
	t.Helper()

	t.Run("inspect returns the release line of the server and the effective privileges", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		username, password := server.createUser(t, databaseName, "GRANT SELECT ON `%s`.* TO %s")

		inspection, err := server.newSource(t, username, password, databaseName).
			Inspect(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
		assert.Equal(t, version, inspection.Version)
		assert.Equal(t, "SELECT", inspection.Privileges.GetEffectivePrivilegesCsv())
	})
}

func testShouldSuggestReadOnlyUser(t *testing.T, server testServer) {
	t.Helper()

	t.Run("should suggest read-only user returns true with the root account", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")

		shouldSuggestReadOnlyUser, privileges, err := server.newRootSource(t, databaseName).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.True(t, shouldSuggestReadOnlyUser)
		assert.NotEmpty(t, privileges)
	})

	t.Run("create read-only user returns an account that can read but not write", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "readonly_test")
		server.exec(t, fmt.Sprintf("INSERT INTO `%s`.`readonly_test` (data) VALUES ('test1'), ('test2')", databaseName))

		username, password, err := server.newRootSource(t, databaseName).
			CreateReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		server.dropUserOnCleanup(t, username)
		assert.True(t, strings.HasPrefix(username, "pgs-"))
		assert.NotEmpty(t, password)

		shouldSuggestReadOnlyUser, privileges, err := server.newSource(t, username, password, databaseName).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger())
		require.NoError(t, err)
		assert.False(t, shouldSuggestReadOnlyUser)
		assert.Empty(t, privileges)

		readOnlyDB := server.connectAs(t, username, password, databaseName)

		var rowCount int
		require.NoError(
			t,
			readOnlyDB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM readonly_test").Scan(&rowCount),
		)
		assert.Equal(t, 2, rowCount)

		for _, writeStatement := range []string{
			"INSERT INTO readonly_test (data) VALUES ('should-fail')",
			"UPDATE readonly_test SET data = 'hacked' WHERE id = 1",
			"DELETE FROM readonly_test WHERE id = 1",
			"CREATE TABLE hack_table (id INT)",
		} {
			_, err = readOnlyDB.ExecContext(t.Context(), writeStatement)
			require.Error(t, err, writeStatement)
			assert.Contains(t, strings.ToLower(err.Error()), "denied", writeStatement)
		}
	})
}

// issue #568: an account with only SELECT, SHOW VIEW and SHOW CREATE ROUTINE is read-only. A
// substring match once read "CREATE" and "CREATE ROUTINE" inside "SHOW CREATE ROUTINE".
func testShouldSuggestReadOnlyUserWithShowCreateRoutine(t *testing.T, server testServer) {
	t.Helper()

	t.Run("should suggest read-only user returns false with a SHOW CREATE ROUTINE grant", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		username, password := server.createUser(t, databaseName,
			"GRANT SELECT ON *.* TO %[2]s",
			"GRANT SHOW VIEW ON *.* TO %[2]s",
			"GRANT SHOW CREATE ROUTINE ON *.* TO %[2]s",
		)

		shouldSuggestReadOnlyUser, privileges, err := server.newSource(t, username, password, databaseName).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.False(t, shouldSuggestReadOnlyUser)
		assert.Empty(t, privileges)
	})
}

func testGetRawSizeMb(t *testing.T, server testServer) {
	t.Helper()

	t.Run("get raw size returns a positive size after rows are inserted", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.exec(t, fmt.Sprintf(
			"CREATE TABLE `%s`.`size_test` (id INT AUTO_INCREMENT PRIMARY KEY, payload TEXT NOT NULL)",
			databaseName,
		))

		for range 1000 {
			_, err := server.rootDB.ExecContext(t.Context(),
				fmt.Sprintf("INSERT INTO `%s`.`size_test` (payload) VALUES (?)", databaseName),
				strings.Repeat("x", 1024),
			)
			require.NoError(t, err)
		}

		server.exec(t, fmt.Sprintf("ANALYZE TABLE `%s`.`size_test`", databaseName))

		sizeMb, err := server.newRootSource(t, databaseName).GetRawSizeMb(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Greater(t, sizeMb, 0.0)
	})
}

func testTestConnectionWithRoles(t *testing.T, server testServer) {
	t.Helper()

	t.Run("test connection succeeds when the backup grants come from the active role", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		roleName := createRole(t, server)
		server.exec(t, fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO %s",
			databaseName,
			roleName,
		))
		username, password := server.createUser(t, databaseName,
			"GRANT "+roleName+" TO %[2]s",
			"SET DEFAULT ROLE "+roleName+" FOR %[2]s",
		)

		inspection, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
		assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SELECT")
		assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SHOW VIEW")
	})

	// The account's SELECT arrives only through a role granted to its active role, so this pins that
	// one SHOW GRANTS FOR the active role carries the inherited grants: read only the account's own
	// grants and the check falsely reports missing privileges.
	t.Run("test connection succeeds when the backup grants come from a nested role", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		roleName := createRole(t, server)
		nestedRoleName := createRole(t, server)
		server.exec(t,
			fmt.Sprintf("GRANT SELECT, SHOW VIEW ON `%s`.* TO %s", databaseName, nestedRoleName),
			fmt.Sprintf("GRANT %s TO %s", nestedRoleName, roleName),
		)
		username, password := server.createUser(t, databaseName,
			"GRANT "+roleName+" TO %[2]s",
			"SET DEFAULT ROLE "+roleName+" FOR %[2]s",
		)

		inspection, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
		assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SELECT")
		assert.Contains(t, inspection.Privileges.GetEffectivePrivilegesCsv(), "SHOW VIEW")
	})

	// mariadb-dump logs in fresh and a granted but inactive role gives it nothing. The direct INSERT
	// grant puts the table in the dump set; everything needed to read it sits in the role that is
	// never activated.
	t.Run("test connection reports missing SELECT when the role is granted but not activated", func(t *testing.T) {
		databaseName := server.createDatabase(t, "db_")
		server.createTable(t, databaseName, "blocked")
		roleName := createRole(t, server)
		server.exec(t, fmt.Sprintf("GRANT SELECT, SHOW VIEW ON `%s`.* TO %s", databaseName, roleName))
		username, password := server.createUser(t, databaseName,
			"GRANT INSERT ON `%s`.`blocked` TO %s",
			"GRANT "+roleName+" TO %[2]s",
		)

		_, err := server.newSource(t, username, password, databaseName).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), fmt.Sprintf("SELECT on `%s`.*", databaseName))
	})
}

// A MariaDB role carries no host part, unlike a MySQL one.
func createRole(t *testing.T, server testServer) string {
	t.Helper()

	roleName := "backup_role_" + uuid.New().String()[:8]
	server.exec(t, "CREATE ROLE "+roleName)

	t.Cleanup(func() {
		_, _ = server.rootDB.ExecContext(context.WithoutCancel(t.Context()), "DROP ROLE IF EXISTS "+roleName)
	})

	return roleName
}

func TestQuoteRoleName(t *testing.T) {
	t.Parallel()

	t.Run("escapes the backquote with a backquote in the name", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "`plain_role`", quoteRoleName("plain_role"))
		assert.Equal(t, "`odd``role`", quoteRoleName("odd`role"))
	})
}
