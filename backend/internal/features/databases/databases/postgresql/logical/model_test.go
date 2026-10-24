package postgresql_logical

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

const heavyTableMinSizeMb = 10.0

func Test_TestConnection_PasswordContainingSpaces_TestedSuccessfully(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	passwordWithSpaces := "test password with spaces"
	usernameWithSpaces := fmt.Sprintf("testuser_spaces_%s", uuid.New().String()[:8])

	_, err := container.DB.Exec(`
		DROP TABLE IF EXISTS password_test CASCADE;
		CREATE TABLE password_test (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO password_test (data) VALUES ('test1');
	`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`,
		usernameWithSpaces,
		passwordWithSpaces,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		usernameWithSpaces,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT USAGE ON SCHEMA public TO "%s"`,
		usernameWithSpaces,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%s"`,
		usernameWithSpaces,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "%s"`,
		usernameWithSpaces,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, usernameWithSpaces))
	}()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum("16"),
		Host:     container.Host,
		Port:     container.Port,
		Username: usernameWithSpaces,
		Password: passwordWithSpaces,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = pgModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func Test_TestConnection_WrongPassword_ReturnsError(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum("16"),
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password + "-wrong",
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := pgModel.TestConnection(logger, nil)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "password")
}

func Test_TestConnection_UnreachableHost_ReturnsError(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum("16"),
		Host:     container.Host,
		Port:     1, // no PostgreSQL is listening on this port
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := pgModel.TestConnection(logger, nil)
	assert.Error(t, err)
}

type postgresModelVersion struct {
	name  string
	tag   string
	image string
}

var postgresModelVersions = []postgresModelVersion{
	{"PostgreSQL 12", "12", "postgres:12"},
	{"PostgreSQL 13", "13", "postgres:13"},
	{"PostgreSQL 14", "14", "postgres:14"},
	{"PostgreSQL 15", "15", "postgres:15"},
	{"PostgreSQL 16", "16", "postgres:16"},
	{"PostgreSQL 17", "17", "postgres:17"},
}

// Test_PostgresqlModel_AcrossSupportedVersions boots each PostgreSQL version once and runs every
// matrix model test against it as a subtest. Only one container is alive per package at a time. See docs/testing.md.
func Test_PostgresqlModel_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range postgresModelVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartPostgres(t, dbVersion.image)

			t.Run("Test_TestConnection_InsufficientPermissions_ReturnsError", func(t *testing.T) {
				testConnectionInsufficientPermissions(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_SufficientPermissions_Success", func(t *testing.T) {
				testConnectionSufficientPermissions(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_ShouldSuggestReadOnlyUser_AdminUser_ReturnsTrue", func(t *testing.T) {
				testShouldSuggestReadOnlyUserAdminUser(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_CreateReadOnlyUser_UserCanReadButNotWrite", func(t *testing.T) {
				testCreateReadOnlyUserCanReadButNotWrite(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_CreateReadOnlyUser_WithPublicSchema_Success", func(t *testing.T) {
				testCreateReadOnlyUserWithPublicSchema(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_CreateReadOnlyUser_WithoutPublicSchema_Success", func(t *testing.T) {
				testCreateReadOnlyUserWithoutPublicSchema(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_CreateReadOnlyUser_PublicSchemaExistsButNoPermissions_ReturnsError", func(t *testing.T) {
				testCreateReadOnlyUserPublicSchemaExistsButNoPermissions(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_ShouldSuggestReadOnlyUser_WhenRlsTableInScope_DoesNotSuggest", func(t *testing.T) {
				testShouldSuggestReadOnlyUserWhenRlsTableInScope(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_ShouldSuggestReadOnlyUser_WhenRlsTableOutsideIncludeSchemas_Suggests", func(t *testing.T) {
				testShouldSuggestReadOnlyUserWhenRlsTableOutsideIncludeSchemas(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_CreateReadOnlyUser_WhenRlsTableInScope_ReturnsErrorAndCreatesNoRole", func(t *testing.T) {
				testCreateReadOnlyUserWhenRlsTableInScope(t, endpoint, dbVersion.tag)
			})

			t.Run(
				"Test_TestConnection_WhenSelectIsMissingOnSomeTables_ReturnsErrorNamingOnlyUnreadableTable",
				func(t *testing.T) {
					testConnectionSelectMissingOnSomeTables(t, endpoint, dbVersion.tag)
				},
			)

			t.Run("Test_TestConnection_WhenEveryTableAndSequenceIsReadable_Success", func(t *testing.T) {
				testConnectionEveryTableAndSequenceReadable(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenSelectIsMissingOnSequence_ReturnsError", func(t *testing.T) {
				testConnectionSelectMissingOnSequence(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenUnreadableTableIsExcluded_Success", func(t *testing.T) {
				testConnectionUnreadableTableExcluded(t, endpoint, dbVersion.tag)
			})

			t.Run(
				"Test_TestConnection_WhenExcludedEntryIsUnnormalized_Success",
				func(t *testing.T) {
					testConnectionUnreadableTableExcludedWithUnnormalizedEntry(
						t,
						endpoint,
						dbVersion.tag,
					)
				},
			)

			t.Run(
				"Test_TestConnection_WhenUnreadableTableDataIsExcluded_ReturnsErrorNamingTable",
				func(t *testing.T) {
					testConnectionUnreadableTableDataExcluded(t, endpoint, dbVersion.tag)
				},
			)

			t.Run(
				"Test_TestConnection_WhenUnqualifiedExclusionNamesTableOffSearchPath_ReturnsErrorNamingIt",
				func(t *testing.T) {
					testConnectionUnqualifiedExclusionDoesNotReachTableOffSearchPath(
						t,
						endpoint,
						dbVersion.tag,
					)
				},
			)

			t.Run("Test_TestConnection_WhenUnreadableTableIsOutsideIncludedSchemas_Success", func(t *testing.T) {
				testConnectionUnreadableTableOutsideIncludedSchemas(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenSchemaUsageIsMissing_ReturnsError", func(t *testing.T) {
				testConnectionSchemaUsageMissing(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenRoleOwnsEveryTable_Success", func(t *testing.T) {
				testConnectionRoleOwnsEveryTable(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenNoRelationIsInScope_Success", func(t *testing.T) {
				testConnectionNoRelationInScope(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenPartitionIsUnreadable_ReturnsError", func(t *testing.T) {
				testConnectionPartitionUnreadable(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenUnreadableTableBelongsToExtension_Success", func(t *testing.T) {
				testConnectionUnreadableTableBelongsToExtension(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenUserMappingUnreadableAndFlagFalse_ReturnsError", func(t *testing.T) {
				testConnectionUserMappingUnreadableFlagFalse(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_TestConnection_WhenUserMappingUnreadableAndFlagTrue_NoError", func(t *testing.T) {
				testConnectionUserMappingUnreadableFlagTrue(t, endpoint, dbVersion.tag)
			})
		})
	}
}

// setupUnreadableUserMappingModel sets up a superuser-owned user mapping and a limited role that
// passes the backup-permission checks (CONNECT/USAGE/SELECT) yet cannot read the mapping's options.
// postgres_fdw stands in for oracle_fdw, which is not installable in CI.
func setupUnreadableUserMappingModel(
	t *testing.T,
	container *PostgresContainer,
	version string,
) *PostgresqlLogicalDatabase {
	suffix := uuid.New().String()[:8]
	limitedUsername := fmt.Sprintf("um_limited_%s", suffix)
	limitedPassword := "limitedpassword123"
	serverName := fmt.Sprintf("um_test_srv_%s", suffix)
	// The version's subtests share one server (docs/testing.md, "Isolation"), so an own schema keeps the dump scope
	// free of the tables other subtests leave behind in public.
	schemaName := fmt.Sprintf("um_test_schema_%s", suffix)

	setupStatements := []string{
		fmt.Sprintf(`CREATE SCHEMA %s`, schemaName),
		fmt.Sprintf(`CREATE TABLE %s.um_test_table (id SERIAL PRIMARY KEY, data TEXT NOT NULL)`, schemaName),
		fmt.Sprintf(`INSERT INTO %s.um_test_table (data) VALUES ('row1')`, schemaName),
		`CREATE EXTENSION IF NOT EXISTS postgres_fdw`,
		fmt.Sprintf(
			`CREATE SERVER %s FOREIGN DATA WRAPPER postgres_fdw OPTIONS (host 'localhost', dbname 'postgres')`,
			serverName,
		),
		fmt.Sprintf(
			`CREATE USER MAPPING FOR CURRENT_USER SERVER %s OPTIONS ("user" 'remote', password 'secret')`,
			serverName,
		),
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, limitedUsername, limitedPassword),
		fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, container.Database, limitedUsername),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "%s"`, schemaName, limitedUsername),
		fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA %s TO "%s"`, schemaName, limitedUsername),
		fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA %s TO "%s"`, schemaName, limitedUsername),
	}

	for _, statement := range setupStatements {
		_, err := container.DB.Exec(statement)
		assert.NoError(t, err)
	}

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP SERVER IF EXISTS %s CASCADE`, serverName))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schemaName))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s"`, limitedUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, limitedUsername))
	})

	return &PostgresqlLogicalDatabase{
		Version:        tools.GetPostgresqlVersionEnum(version),
		Host:           container.Host,
		Port:           container.Port,
		Username:       limitedUsername,
		Password:       limitedPassword,
		Database:       &container.Database,
		SslMode:        postgresql_shared.PostgresSslModeDisable,
		IncludeSchemas: []string{schemaName},
		CpuCount:       1,
	}
}

func testConnectionUserMappingUnreadableFlagFalse(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	limitedModel := setupUnreadableUserMappingModel(t, container, version)
	limitedModel.IsSkipUserMappings = false

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := limitedModel.TestConnection(logger, nil)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "user mapping")
	}
}

func testConnectionUserMappingUnreadableFlagTrue(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	limitedModel := setupUnreadableUserMappingModel(t, container, version)
	limitedModel.IsSkipUserMappings = true

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := limitedModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testConnectionInsufficientPermissions(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
				DROP TABLE IF EXISTS permission_test CASCADE;
				CREATE TABLE permission_test (
					id SERIAL PRIMARY KEY,
					data TEXT NOT NULL
				);
				INSERT INTO permission_test (data) VALUES ('test1');
			`)
	assert.NoError(t, err)

	limitedUsername := fmt.Sprintf("limited_user_%s", uuid.New().String()[:8])
	limitedPassword := "limitedpassword123"

	_, err = container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`,
		limitedUsername,
		limitedPassword,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		limitedUsername,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, limitedUsername))
	}()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum(version),
		Host:     container.Host,
		Port:     container.Port,
		Username: limitedUsername,
		Password: limitedPassword,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = pgModel.TestConnection(logger, nil)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "insufficient permissions")
	}
}

func testConnectionSufficientPermissions(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
				DROP TABLE IF EXISTS backup_test CASCADE;
				CREATE TABLE backup_test (
					id SERIAL PRIMARY KEY,
					data TEXT NOT NULL
				);
				INSERT INTO backup_test (data) VALUES ('test1');
			`)
	assert.NoError(t, err)

	backupUsername := fmt.Sprintf("backup_user_%s", uuid.New().String()[:8])
	backupPassword := "backuppassword123"

	_, err = container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`,
		backupUsername,
		backupPassword,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		backupUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT USAGE ON SCHEMA public TO "%s"`,
		backupUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%s"`,
		backupUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "%s"`,
		backupUsername,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, backupUsername))
	}()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum(version),
		Host:     container.Host,
		Port:     container.Port,
		Username: backupUsername,
		Password: backupPassword,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = pgModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testShouldSuggestReadOnlyUserAdminUser(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	shouldSuggestReadOnlyUser, privileges, err := pgModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.True(t, shouldSuggestReadOnlyUser, "Admin user must be offered a read-only user")
	assert.NotEmpty(t, privileges, "Admin user should have privileges")
}

func Test_ShouldSuggestReadOnlyUser_ReadOnlyUser_ReturnsFalse(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	_, err := container.DB.Exec(`
		DROP TABLE IF EXISTS readonly_check_test CASCADE;
		CREATE TABLE readonly_check_test (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO readonly_check_test (data) VALUES ('test1');
	`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyModel := &PostgresqlLogicalDatabase{
		Version:  pgModel.Version,
		Host:     pgModel.Host,
		Port:     pgModel.Port,
		Username: username,
		Password: password,
		Database: pgModel.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Read-only user must not be offered another one")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}
	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func testCreateReadOnlyUserCanReadButNotWrite(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
			DROP TABLE IF EXISTS readonly_test CASCADE;
			DROP TABLE IF EXISTS hack_table CASCADE;
			DROP TABLE IF EXISTS future_table CASCADE;
			CREATE TABLE readonly_test (
				id SERIAL PRIMARY KEY,
				data TEXT NOT NULL
			);
			INSERT INTO readonly_test (data) VALUES ('test1'), ('test2');
		`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyModel := &PostgresqlLogicalDatabase{
		Version:  pgModel.Version,
		Host:     pgModel.Host,
		Port:     pgModel.Port,
		Username: username,
		Password: password,
		Database: pgModel.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Created user must not be offered another one")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	readOnlyDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		username,
		password,
		container.Database,
	)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM readonly_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec("INSERT INTO readonly_test (data) VALUES ('should-fail')")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("UPDATE readonly_test SET data = 'hacked' WHERE id = 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("DELETE FROM readonly_test WHERE id = 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("CREATE TABLE hack_table (id INT)")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_FutureTables_HaveSelectPermission(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`
		CREATE TABLE future_table (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO future_table (data) VALUES ('future_data');
	`)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, username, password, container.Database)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var data string
	err = readOnlyConn.Get(&data, "SELECT data FROM future_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "future_data", data)

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_MultipleSchemas_AllAccessible(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	_, err := container.DB.Exec(`
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;
		CREATE TABLE schema_a.table_a (id INT, data TEXT);
		CREATE TABLE schema_b.table_b (id INT, data TEXT);
		INSERT INTO schema_a.table_a VALUES (1, 'data_a');
		INSERT INTO schema_b.table_b VALUES (2, 'data_b');
	`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, username, password, container.Database)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var dataA string
	err = readOnlyConn.Get(&dataA, "SELECT data FROM schema_a.table_a LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "data_a", dataA)

	var dataB string
	err = readOnlyConn.Get(&dataB, "SELECT data FROM schema_b.table_b LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "data_b", dataB)

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
	_, err = container.DB.Exec(`DROP SCHEMA schema_a CASCADE; DROP SCHEMA schema_b CASCADE;`)
	assert.NoError(t, err)
}

func Test_CreateReadOnlyUser_DatabaseNameWithDash_Success(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	dashDbName := "test-db-with-dash"

	_, err := container.DB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dashDbName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dashDbName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dashDbName))
	}()

	dashDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, dashDbName)
	dashDB, err := sqlx.Connect("postgres", dashDSN)
	assert.NoError(t, err)
	defer dashDB.Close()

	_, err = dashDB.Exec(`
		CREATE TABLE dash_test (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO dash_test (data) VALUES ('test1'), ('test2');
	`)
	assert.NoError(t, err)

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum("16"),
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &dashDbName,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, username, password, dashDbName)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM dash_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec("INSERT INTO dash_test (data) VALUES ('should-fail')")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = dashDB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = dashDB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func testCreateReadOnlyUserWithPublicSchema(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
				DROP TABLE IF EXISTS public_schema_test CASCADE;
				CREATE TABLE public_schema_test (
					id SERIAL PRIMARY KEY,
					data TEXT NOT NULL
				);
				INSERT INTO public_schema_test (data) VALUES ('test1'), ('test2');
			`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyModel := &PostgresqlLogicalDatabase{
		Version:  pgModel.Version,
		Host:     pgModel.Host,
		Port:     pgModel.Port,
		Username: username,
		Password: password,
		Database: pgModel.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "User must not be offered a read-only user")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	readOnlyDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		username,
		password,
		container.Database,
	)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM public_schema_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec(
		"INSERT INTO public_schema_test (data) VALUES ('should-fail')",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("CREATE TABLE public.hack_table (id INT)")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}
	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func testCreateReadOnlyUserWithoutPublicSchema(t *testing.T, endpoint containers.Endpoint, version string) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
				DROP SCHEMA IF EXISTS public CASCADE;
				DROP SCHEMA IF EXISTS app_schema CASCADE;
				DROP SCHEMA IF EXISTS data_schema CASCADE;
				CREATE SCHEMA app_schema;
				CREATE SCHEMA data_schema;
				CREATE TABLE app_schema.users (
					id SERIAL PRIMARY KEY,
					username TEXT NOT NULL
				);
				CREATE TABLE data_schema.records (
					id SERIAL PRIMARY KEY,
					info TEXT NOT NULL
				);
				INSERT INTO app_schema.users (username) VALUES ('user1'), ('user2');
				INSERT INTO data_schema.records (info) VALUES ('record1'), ('record2');
			`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err, "CreateReadOnlyUser should succeed without public schema")
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyModel := &PostgresqlLogicalDatabase{
		Version:  pgModel.Version,
		Host:     pgModel.Host,
		Port:     pgModel.Port,
		Username: username,
		Password: password,
		Database: pgModel.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "User must not be offered a read-only user")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	readOnlyDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		username,
		password,
		container.Database,
	)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var userCount int
	err = readOnlyConn.Get(&userCount, "SELECT COUNT(*) FROM app_schema.users")
	assert.NoError(t, err)
	assert.Equal(t, 2, userCount)

	var recordCount int
	err = readOnlyConn.Get(&recordCount, "SELECT COUNT(*) FROM data_schema.records")
	assert.NoError(t, err)
	assert.Equal(t, 2, recordCount)

	_, err = readOnlyConn.Exec(
		"INSERT INTO app_schema.users (username) VALUES ('should-fail')",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("CREATE TABLE app_schema.hack_table (id INT)")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = readOnlyConn.Exec("CREATE TABLE data_schema.hack_table (id INT)")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}
	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)

	_, err = container.DB.Exec(`
				DROP SCHEMA IF EXISTS app_schema CASCADE;
				DROP SCHEMA IF EXISTS data_schema CASCADE;
				CREATE SCHEMA IF NOT EXISTS public;
			`)
	assert.NoError(t, err)
}

func testCreateReadOnlyUserPublicSchemaExistsButNoPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	container := connectToPostgresEndpoint(t, endpoint)
	defer container.DB.Close()

	limitedAdminUsername := fmt.Sprintf("limited_admin_%s", uuid.New().String()[:8])
	limitedAdminPassword := "limited_password_123"

	_, err := container.DB.Exec(`
				CREATE SCHEMA IF NOT EXISTS public;
				DROP TABLE IF EXISTS public.permission_test_table CASCADE;
				CREATE TABLE public.permission_test_table (
					id SERIAL PRIMARY KEY,
					data TEXT NOT NULL
				);
				INSERT INTO public.permission_test_table (data) VALUES ('test1');
			`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`GRANT CREATE ON SCHEMA public TO PUBLIC`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN CREATEROLE`,
		limitedAdminUsername,
		limitedAdminPassword,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		limitedAdminUsername,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, limitedAdminUsername),
		)
		_, _ = container.DB.Exec(
			fmt.Sprintf(`DROP USER IF EXISTS "%s"`, limitedAdminUsername),
		)
		_, _ = container.DB.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC`)
	}()

	limitedAdminDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		limitedAdminUsername,
		limitedAdminPassword,
		container.Database,
	)
	limitedAdminConn, err := sqlx.Connect("postgres", limitedAdminDSN)
	assert.NoError(t, err)
	defer limitedAdminConn.Close()

	pgModel := &PostgresqlLogicalDatabase{
		Version:  tools.GetPostgresqlVersionEnum(version),
		Host:     container.Host,
		Port:     container.Port,
		Username: limitedAdminUsername,
		Password: limitedAdminPassword,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.Error(
		t,
		err,
		"CreateReadOnlyUser should fail when admin lacks permissions to secure public schema",
	)
	if err != nil {
		errorMsg := err.Error()
		hasExpectedError := strings.Contains(
			errorMsg,
			"failed to revoke CREATE from PUBLIC on existing public schema",
		) ||
			strings.Contains(errorMsg, "permission denied for schema public") ||
			strings.Contains(errorMsg, "failed to grant")
		assert.True(
			t,
			hasExpectedError,
			"Error should indicate permission issues with public schema, got: %s",
			errorMsg,
		)
	}
	assert.Empty(t, username)
	assert.Empty(t, password)
}

func Test_Validate_WhenLocalhostAndDatabasus_ReturnsError(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		username string
		database string
	}{
		{
			name:     "localhost with databasus db",
			host:     "localhost",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.0.0.1 with databasus db",
			host:     "127.0.0.1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "172.17.0.1 with databasus db",
			host:     "172.17.0.1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "host.docker.internal with databasus db",
			host:     "host.docker.internal",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "LOCALHOST (uppercase) with DATABASUS db",
			host:     "LOCALHOST",
			username: "POSTGRES",
			database: "DATABASUS",
		},
		{
			name:     "LocalHost (mixed case) with DataBasus db",
			host:     "LocalHost",
			username: "anyuser",
			database: "DataBasus",
		},
		{
			name:     "localhost with databasus and any username",
			host:     "localhost",
			username: "myuser",
			database: "databasus",
		},
		{
			name:     "::1 (IPv6 loopback) with databasus db",
			host:     "::1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     ":: (IPv6 all interfaces) with databasus db",
			host:     "::",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "::1 (uppercase) with DATABASUS db",
			host:     "::1",
			username: "POSTGRES",
			database: "DATABASUS",
		},
		{
			name:     "0.0.0.0 (all IPv4 interfaces) with databasus db",
			host:     "0.0.0.0",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.0.0.2 (loopback range) with databasus db",
			host:     "127.0.0.2",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.255.255.255 (end of loopback range) with databasus db",
			host:     "127.255.255.255",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "Unix socket with databasus db",
			host:     "/tmp",
			username: "postgres",
			database: "databasus",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgModel := &PostgresqlLogicalDatabase{
				Host:     tc.host,
				Port:     5437,
				Username: tc.username,
				Password: "somepassword",
				Database: &tc.database,
				CpuCount: 1,
			}

			err := pgModel.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "backing up Databasus internal database is not allowed")
			assert.Contains(t, err.Error(), "https://databasus.com/faq#backup-databasus")
		})
	}
}

func Test_Validate_WhenNotLocalhostOrNotDatabasus_ValidatesSuccessfully(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		username string
		database string
	}{
		{
			name:     "different host (remote server) with databasus db",
			host:     "192.168.1.100",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "different database name on localhost",
			host:     "localhost",
			username: "postgres",
			database: "myapp",
		},
		{
			name:     "all different",
			host:     "db.example.com",
			username: "appuser",
			database: "production",
		},
		{
			name:     "localhost with postgres database",
			host:     "localhost",
			username: "postgres",
			database: "postgres",
		},
		{
			name:     "remote host with databasus db name (allowed)",
			host:     "db.example.com",
			username: "postgres",
			database: "databasus",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgModel := &PostgresqlLogicalDatabase{
				Host:     tc.host,
				Port:     5432,
				Username: tc.username,
				Password: "somepassword",
				Database: &tc.database,
				CpuCount: 1,
			}

			err := pgModel.Validate()
			assert.NoError(t, err)
		})
	}
}

func Test_Validate_WhenDatabaseIsNil_ValidatesSuccessfully(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Password: "somepassword",
		Database: nil,
		CpuCount: 1,
	}

	err := pgModel.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenDatabaseIsEmpty_ValidatesSuccessfully(t *testing.T) {
	emptyDb := ""
	pgModel := &PostgresqlLogicalDatabase{
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Password: "somepassword",
		Database: &emptyDb,
		CpuCount: 1,
	}

	err := pgModel.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenRequiredFieldsMissing_ReturnsError(t *testing.T) {
	testCases := []struct {
		name          string
		model         *PostgresqlLogicalDatabase
		expectedError string
	}{
		{
			name: "missing host",
			model: &PostgresqlLogicalDatabase{
				Host:     "",
				Port:     5432,
				Username: "user",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "host is required",
		},
		{
			name: "missing port",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     0,
				Username: "user",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "port is required",
		},
		{
			name: "missing username",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "username is required",
		},
		{
			name: "missing password",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "",
				CpuCount: 1,
			},
			expectedError: "password is required",
		},
		{
			name: "invalid cpu count",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "pass",
				CpuCount: 0,
			},
			expectedError: "cpu count must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.model.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

type PostgresContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	DB       *sqlx.DB
}

func Test_CreateReadOnlyUser_TablesCreatedByDifferentUser_ReadOnlyUserCanRead(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	// Step 1: Create a second database user who will create tables
	userCreatorUsername := fmt.Sprintf("user_creator_%s", uuid.New().String()[:8])
	userCreatorPassword := "creator_password_123"

	_, err := container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`,
		userCreatorUsername,
		userCreatorPassword,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, userCreatorUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, userCreatorUsername))
	}()

	// Step 2: Grant the user_creator privileges to connect and create tables
	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		userCreatorUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT USAGE ON SCHEMA public TO "%s"`,
		userCreatorUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CREATE ON SCHEMA public TO "%s"`,
		userCreatorUsername,
	))
	assert.NoError(t, err)

	// Step 2b: Create an initial table by user_creator so they become an object owner
	// This is important because our fix discovers existing object owners
	userCreatorDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		userCreatorUsername,
		userCreatorPassword,
		container.Database,
	)
	userCreatorConn, err := sqlx.Connect("postgres", userCreatorDSN)
	assert.NoError(t, err)
	defer userCreatorConn.Close()

	initialTableName := fmt.Sprintf(
		"public.initial_table_%s",
		strings.ReplaceAll(uuid.New().String()[:8], "-", ""),
	)
	_, err = userCreatorConn.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO %s (data) VALUES ('initial_data');
	`, initialTableName, initialTableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE`, initialTableName))
	}()

	// Step 3: NOW create read-only user via Databasus (as admin)
	// At this point, user_creator already owns objects, so ALTER DEFAULT PRIVILEGES FOR ROLE should apply
	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	readonlyUsername, readonlyPassword, err := pgModel.CreateReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, readonlyUsername)
	assert.NotEmpty(t, readonlyPassword)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, readonlyUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, readonlyUsername))
	}()

	// Step 4: user_creator creates a NEW table AFTER the read-only user was created
	// This table should automatically grant SELECT to the read-only user via ALTER DEFAULT PRIVILEGES FOR ROLE
	tableName := fmt.Sprintf(
		"public.future_table_%s",
		strings.ReplaceAll(uuid.New().String()[:8], "-", ""),
	)
	_, err = userCreatorConn.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO %s (data) VALUES ('test_data_1'), ('test_data_2');
	`, tableName, tableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE`, tableName))
	}()

	// Step 5: Connect as read-only user and verify it can SELECT from the new table
	readonlyDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		readonlyUsername,
		readonlyPassword,
		container.Database,
	)
	readonlyConn, err := sqlx.Connect("postgres", readonlyDSN)
	assert.NoError(t, err)
	defer readonlyConn.Close()

	var count int
	err = readonlyConn.Get(&count, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName))
	assert.NoError(t, err)
	assert.Equal(
		t,
		2,
		count,
		"Read-only user should be able to SELECT from table created by different user",
	)

	// Step 6: Verify read-only user cannot write to the table
	_, err = readonlyConn.Exec(
		fmt.Sprintf("INSERT INTO %s (data) VALUES ('should-fail')", tableName),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// Step 7: Verify pg_dump operations (LOCK TABLE) work
	// pg_dump needs to lock tables in ACCESS SHARE MODE for consistent backup
	tx, err := readonlyConn.Begin()
	assert.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.Exec(fmt.Sprintf("LOCK TABLE %s IN ACCESS SHARE MODE", tableName))
	assert.NoError(t, err, "Read-only user should be able to LOCK TABLE (needed for pg_dump)")

	err = tx.Commit()
	assert.NoError(t, err)
}

func Test_CreateReadOnlyUser_WithIncludeSchemas_OnlyGrantsAccessToSpecifiedSchemas(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	// Step 1: Create multiple schemas and tables
	_, err := container.DB.Exec(`
		DROP SCHEMA IF EXISTS included_schema CASCADE;
		DROP SCHEMA IF EXISTS excluded_schema CASCADE;
		CREATE SCHEMA included_schema;
		CREATE SCHEMA excluded_schema;

		CREATE TABLE public.public_table (id INT, data TEXT);
		INSERT INTO public.public_table VALUES (1, 'public_data');

		CREATE TABLE included_schema.included_table (id INT, data TEXT);
		INSERT INTO included_schema.included_table VALUES (2, 'included_data');

		CREATE TABLE excluded_schema.excluded_table (id INT, data TEXT);
		INSERT INTO excluded_schema.excluded_table VALUES (3, 'excluded_data');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`DROP SCHEMA IF EXISTS included_schema CASCADE`)
		_, _ = container.DB.Exec(`DROP SCHEMA IF EXISTS excluded_schema CASCADE`)
	}()

	// Step 2: Create a second user who owns tables in both included and excluded schemas
	userCreatorUsername := fmt.Sprintf("user_creator_%s", uuid.New().String()[:8])
	userCreatorPassword := "creator_password_123"

	_, err = container.DB.Exec(fmt.Sprintf(
		`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`,
		userCreatorUsername,
		userCreatorPassword,
	))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, userCreatorUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, userCreatorUsername))
	}()

	// Grant privileges to user_creator
	_, err = container.DB.Exec(fmt.Sprintf(
		`GRANT CONNECT ON DATABASE "%s" TO "%s"`,
		container.Database,
		userCreatorUsername,
	))
	assert.NoError(t, err)

	for _, schema := range []string{"public", "included_schema", "excluded_schema"} {
		_, err = container.DB.Exec(fmt.Sprintf(
			`GRANT USAGE, CREATE ON SCHEMA %s TO "%s"`,
			schema,
			userCreatorUsername,
		))
		assert.NoError(t, err)
	}

	// User_creator creates tables in included and excluded schemas
	userCreatorDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		userCreatorUsername,
		userCreatorPassword,
		container.Database,
	)
	userCreatorConn, err := sqlx.Connect("postgres", userCreatorDSN)
	assert.NoError(t, err)
	defer userCreatorConn.Close()

	_, err = userCreatorConn.Exec(`
		CREATE TABLE included_schema.user_table (id INT, data TEXT);
		INSERT INTO included_schema.user_table VALUES (4, 'user_included_data');

		CREATE TABLE excluded_schema.user_excluded_table (id INT, data TEXT);
		INSERT INTO excluded_schema.user_excluded_table VALUES (5, 'user_excluded_data');
	`)
	assert.NoError(t, err)

	// Step 3: Create read-only user with IncludeSchemas = ["public", "included_schema"]
	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{"public", "included_schema"}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	readonlyUsername, readonlyPassword, err := pgModel.CreateReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, readonlyUsername)
	assert.NotEmpty(t, readonlyPassword)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, readonlyUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, readonlyUsername))
	}()

	// Step 4: Connect as read-only user
	readonlyDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host,
		container.Port,
		readonlyUsername,
		readonlyPassword,
		container.Database,
	)
	readonlyConn, err := sqlx.Connect("postgres", readonlyDSN)
	assert.NoError(t, err)
	defer readonlyConn.Close()

	// Step 5: Verify read-only user CAN access included schemas
	var publicData string
	err = readonlyConn.Get(&publicData, "SELECT data FROM public.public_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "public_data", publicData)

	var includedData string
	err = readonlyConn.Get(&includedData, "SELECT data FROM included_schema.included_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "included_data", includedData)

	var userIncludedData string
	err = readonlyConn.Get(&userIncludedData, "SELECT data FROM included_schema.user_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "user_included_data", userIncludedData)

	// Step 6: Verify read-only user CANNOT access excluded schema
	var excludedData string
	err = readonlyConn.Get(&excludedData, "SELECT data FROM excluded_schema.excluded_table LIMIT 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	err = readonlyConn.Get(
		&excludedData,
		"SELECT data FROM excluded_schema.user_excluded_table LIMIT 1",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// Step 7: Verify future tables in included schemas are accessible
	_, err = userCreatorConn.Exec(`
		CREATE TABLE included_schema.future_table (id INT, data TEXT);
		INSERT INTO included_schema.future_table VALUES (6, 'future_data');
	`)
	assert.NoError(t, err)

	var futureData string
	err = readonlyConn.Get(&futureData, "SELECT data FROM included_schema.future_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(
		t,
		"future_data",
		futureData,
		"Read-only user should access future tables in included schemas via ALTER DEFAULT PRIVILEGES FOR ROLE",
	)

	// Step 8: Verify future tables in excluded schema are NOT accessible
	_, err = userCreatorConn.Exec(`
		CREATE TABLE excluded_schema.future_excluded_table (id INT, data TEXT);
		INSERT INTO excluded_schema.future_excluded_table VALUES (7, 'future_excluded_data');
	`)
	assert.NoError(t, err)

	var futureExcludedData string
	err = readonlyConn.Get(
		&futureExcludedData,
		"SELECT data FROM excluded_schema.future_excluded_table LIMIT 1",
	)
	assert.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"permission denied",
		"Read-only user should NOT access tables in excluded schemas",
	)
}

func Test_GetRawDbSizeMb_Postgresql_ReturnsPositiveSize(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	tableName := fmt.Sprintf("size_test_%s", uuid.New().String()[:8])
	_, err := container.DB.Exec(fmt.Sprintf(`
		CREATE TABLE %s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		INSERT INTO %s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 10000);
	`, tableName, tableName))
	assert.NoError(t, err)
	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	}()

	pgModel := createPostgresModel(container)
	assert.NotNil(t, pgModel)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sizeMB, err := pgModel.GetRawDbSizeMb(t.Context(), logger, nil)
	assert.NoError(t, err)
	assert.Greater(t, sizeMB, 0.0, "raw db size should be > 0 after inserting data")
}

func Test_GetRawDbSizeMb_WhenExcludeTablesSet_SkipsExcludedTableSize(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	seededSchema := seedSizeTestSchema(t, container)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{seededSchema.SchemaName}

	sizeWithHeavyTableMb := measureRawDbSizeMb(t, pgModel)

	pgModel.ExcludeTables = []string{seededSchema.SchemaName + "." + seededSchema.HeavyTableName}
	sizeWithoutHeavyTableMb := measureRawDbSizeMb(t, pgModel)

	assert.InDelta(
		t,
		measureRelationsSizeMb(
			t,
			container,
			relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.LightTableName},
		),
		sizeWithoutHeavyTableMb,
		0.5,
		"excluded table size must not be counted",
	)
	assert.Greater(t, sizeWithHeavyTableMb-sizeWithoutHeavyTableMb, heavyTableMinSizeMb)
}

func Test_GetRawDbSizeMb_WhenExcludeTableDataSet_SkipsTableSize(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	seededSchema := seedSizeTestSchema(t, container)

	_, err := container.DB.Exec(fmt.Sprintf(
		`CREATE INDEX ON %s.%s (payload)`,
		seededSchema.SchemaName, seededSchema.HeavyTableName,
	))
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{seededSchema.SchemaName}

	sizeWithHeavyTableDataMb := measureRawDbSizeMb(t, pgModel)

	pgModel.ExcludeTableData = []string{seededSchema.SchemaName + "." + seededSchema.HeavyTableName}
	sizeWithoutHeavyTableDataMb := measureRawDbSizeMb(t, pgModel)

	heavyTableTotalSizeMb := measureRelationsSizeMb(
		t, container,
		relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.HeavyTableName},
	)

	assert.GreaterOrEqual(
		t,
		sizeWithHeavyTableDataMb-sizeWithoutHeavyTableDataMb,
		heavyTableTotalSizeMb-0.01,
		"the data-excluded table must leave the estimate together with its indexes",
	)
}

func Test_GetRawDbSizeMb_WhenExcludeTablesUsesGlobPattern_SkipsMatchingTables(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	seededSchema := seedSizeTestSchema(t, container)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{seededSchema.SchemaName}
	pgModel.ExcludeTables = []string{seededSchema.SchemaName + ".heavy_*"}

	assert.InDelta(
		t,
		measureRelationsSizeMb(
			t,
			container,
			relationName{SchemaName: seededSchema.SchemaName, TableName: seededSchema.LightTableName},
		),
		measureRawDbSizeMb(t, pgModel),
		0.5,
		"glob-matched table size must not be counted",
	)
}

func Test_GetRawDbSizeMb_WhenUnqualifiedTableIsExcludedByEitherList_CountsSameNamedTableInAnotherSchema(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	otherSchema := seedSizeTestSchema(t, container)
	sameNamedTableName := otherSchema.HeavyTableName

	_, err := container.DB.Exec(fmt.Sprintf(`
		CREATE TABLE public.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		INSERT INTO public.%s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
	`, sameNamedTableName, sameNamedTableName))
	assert.NoError(t, err)

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS public.%s", sameNamedTableName))
	})

	publicTableSizeMb := measureRelationsSizeMb(
		t, container, relationName{SchemaName: "public", TableName: sameNamedTableName},
	)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{"public", otherSchema.SchemaName}

	sizeWithoutExclusionMb := measureRawDbSizeMb(t, pgModel)

	pgModel.ExcludeTables = []string{sameNamedTableName}
	sizeWithWholeTableExclusionMb := measureRawDbSizeMb(t, pgModel)

	pgModel.ExcludeTables = nil
	pgModel.ExcludeTableData = []string{sameNamedTableName}
	sizeWithDataExclusionMb := measureRawDbSizeMb(t, pgModel)

	assert.InDelta(
		t,
		publicTableSizeMb,
		sizeWithoutExclusionMb-sizeWithWholeTableExclusionMb,
		0.5,
		"a whole-table exclusion must leave out only the table visible on the default search path",
	)
	assert.InDelta(
		t,
		publicTableSizeMb,
		sizeWithoutExclusionMb-sizeWithDataExclusionMb,
		0.5,
		"a data exclusion must leave out only the table visible on the default search path",
	)
}

func Test_GetRawDbSizeMb_WhenIncludeSchemasSet_CountsOnlyIncludedSchema(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	includedSchema := seedSizeTestSchema(t, container)
	otherSchema := seedSizeTestSchema(t, container)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{includedSchema.SchemaName}

	sizeOfIncludedSchemaMb := measureRawDbSizeMb(t, pgModel)

	pgModel.IncludeSchemas = []string{includedSchema.SchemaName, otherSchema.SchemaName}
	sizeOfBothSchemasMb := measureRawDbSizeMb(t, pgModel)

	assert.InDelta(
		t,
		measureRelationsSizeMb(
			t, container,
			relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.HeavyTableName},
			relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.LightTableName},
		),
		sizeOfIncludedSchemaMb,
		0.5,
		"only the included schema must be counted",
	)
	assert.Greater(t, sizeOfBothSchemasMb-sizeOfIncludedSchemaMb, heavyTableMinSizeMb)
}

func Test_GetRawDbSizeMb_WhenIncludeSchemasAndExcludeTablesSet_AppliesBothFilters(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	includedSchema := seedSizeTestSchema(t, container)
	skippedSchema := seedSizeTestSchema(t, container)

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{includedSchema.SchemaName}
	pgModel.ExcludeTables = []string{
		includedSchema.SchemaName + "." + includedSchema.HeavyTableName,
		skippedSchema.SchemaName + "." + skippedSchema.LightTableName,
	}

	assert.InDelta(
		t,
		measureRelationsSizeMb(
			t, container,
			relationName{SchemaName: includedSchema.SchemaName, TableName: includedSchema.LightTableName},
		),
		measureRawDbSizeMb(t, pgModel),
		0.5,
		"schema filter and table exclusion must both apply",
	)
}

func Test_GetRawDbSizeMb_WhenNoFiltersSet_CountsEveryUserTable(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	firstSchema := seedSizeTestSchema(t, container)
	secondSchema := seedSizeTestSchema(t, container)

	pgModel := createPostgresModel(container)

	sizeOfWholeDatabaseMb := measureRawDbSizeMb(t, pgModel)

	pgModel.IncludeSchemas = []string{firstSchema.SchemaName}
	sizeOfFirstSchemaMb := measureRawDbSizeMb(t, pgModel)

	seededSchemasSizeMb := sizeOfFirstSchemaMb + measureRelationsSizeMb(
		t, container,
		relationName{SchemaName: secondSchema.SchemaName, TableName: secondSchema.HeavyTableName},
		relationName{SchemaName: secondSchema.SchemaName, TableName: secondSchema.LightTableName},
	)

	assert.GreaterOrEqual(t, sizeOfWholeDatabaseMb, seededSchemasSizeMb-0.5)
	assert.Greater(t, sizeOfWholeDatabaseMb-sizeOfFirstSchemaMb, heavyTableMinSizeMb)
}

func Test_GetRawDbSizeMb_WhenTableIsPartitioned_CountsEveryPartitionOnce(t *testing.T) {
	container := connectToPostgresContainer(t, "postgres:16")
	defer container.DB.Close()

	schemaName := fmt.Sprintf("partitioned_size_%s", uuid.New().String()[:8])

	_, err := container.DB.Exec(fmt.Sprintf(`
		CREATE SCHEMA %s;
		CREATE TABLE %s.events (id BIGINT, payload TEXT) PARTITION BY RANGE (id);
		CREATE TABLE %s.events_first PARTITION OF %s.events FOR VALUES FROM (0) TO (10000);
		CREATE TABLE %s.events_second PARTITION OF %s.events FOR VALUES FROM (10000) TO (20000);
		INSERT INTO %s.events
			SELECT generate_series(0, 19999), repeat('x', 1024);
	`, schemaName, schemaName, schemaName, schemaName, schemaName, schemaName, schemaName))
	assert.NoError(t, err)

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
	})

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{schemaName}

	assert.InDelta(
		t,
		measureRelationsSizeMb(
			t, container,
			relationName{SchemaName: schemaName, TableName: "events_first"},
			relationName{SchemaName: schemaName, TableName: "events_second"},
		),
		measureRawDbSizeMb(t, pgModel),
		0.5,
		"partitions must be counted once, without adding the partitioned parent",
	)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	pgModel := &PostgresqlLogicalDatabase{
		Version:        tools.GetPostgresqlVersionEnum("16"),
		Host:           "db.example.com",
		Port:           5432,
		Username:       "appuser",
		Password:       "supersecret",
		Database:       &databaseName,
		SslMode:        postgresql_shared.PostgresSslModeRequire,
		IncludeSchemas: []string{"public"},
		ExcludeTables:  []string{"audit_logs"},
		CpuCount:       4,
	}

	pgModel.HideSensitiveData()

	assert.Empty(t, pgModel.Password)
	assert.Equal(t, "db.example.com", pgModel.Host)
	assert.Equal(t, 5432, pgModel.Port)
	assert.Equal(t, "appuser", pgModel.Username)
	assert.Equal(t, &databaseName, pgModel.Database)
	assert.Equal(t, postgresql_shared.PostgresSslModeRequire, pgModel.SslMode)
	assert.Equal(t, []string{"public"}, pgModel.IncludeSchemas)
	assert.Equal(t, []string{"audit_logs"}, pgModel.ExcludeTables)
	assert.Equal(t, 4, pgModel.CpuCount)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var pgModel *PostgresqlLogicalDatabase

	assert.NotPanics(t, func() {
		pgModel.HideSensitiveData()
	})
}

type stubEncryptor struct{}

func (stubEncryptor) Encrypt(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, "enc:") {
		return value, nil
	}

	return "enc:" + value, nil
}

func (stubEncryptor) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "enc:"), nil
}

func Test_HideSensitiveData_WhenCertsSet_ClearsOnlyClientKey(t *testing.T) {
	databaseName := "appdb"
	pgModel := &PostgresqlLogicalDatabase{
		Host:          "db.example.com",
		Port:          5432,
		Username:      "appuser",
		Password:      "supersecret",
		Database:      &databaseName,
		SslMode:       postgresql_shared.PostgresSslModeVerifyFull,
		SslClientCert: "client-cert-pem",
		SslClientKey:  "client-key-pem",
		SslRootCert:   "root-cert-pem",
		CpuCount:      1,
	}

	pgModel.HideSensitiveData()

	assert.Empty(t, pgModel.Password)
	assert.Empty(t, pgModel.SslClientKey)
	assert.Equal(t, "client-cert-pem", pgModel.SslClientCert)
	assert.Equal(t, "root-cert-pem", pgModel.SslRootCert)
	assert.Equal(t, postgresql_shared.PostgresSslModeVerifyFull, pgModel.SslMode)
}

func Test_EncryptSensitiveFields_WhenCertsSet_EncryptsEverySecret(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{
		Password:      "supersecret",
		SslMode:       postgresql_shared.PostgresSslModeVerifyFull,
		SslClientCert: "client-cert-pem",
		SslClientKey:  "client-key-pem",
		SslRootCert:   "root-cert-pem",
	}

	err := pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Equal(t, "enc:client-cert-pem", pgModel.SslClientCert)
	assert.Equal(t, "enc:client-key-pem", pgModel.SslClientKey)
	assert.Equal(t, "enc:root-cert-pem", pgModel.SslRootCert)

	err = pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Equal(t, "enc:client-key-pem", pgModel.SslClientKey)
}

func Test_EncryptSensitiveFields_WhenCertsEmpty_LeavesThemEmpty(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{Password: "supersecret"}

	err := pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Empty(t, pgModel.SslClientCert)
	assert.Empty(t, pgModel.SslClientKey)
	assert.Empty(t, pgModel.SslRootCert)
}

func Test_Update_WhenIncomingClientKeyEmpty_PreservesExistingKey(t *testing.T) {
	databaseName := "appdb"
	existing := &PostgresqlLogicalDatabase{
		Host:          "old-host",
		Port:          5432,
		Username:      "olduser",
		Password:      "enc:oldpass",
		Database:      &databaseName,
		SslMode:       postgresql_shared.PostgresSslModeVerifyCA,
		SslClientCert: "enc:old-cert",
		SslClientKey:  "enc:old-key",
		SslRootCert:   "enc:old-root",
		CpuCount:      1,
	}

	incoming := &PostgresqlLogicalDatabase{
		Host:          "new-host",
		Port:          5433,
		Username:      "newuser",
		Database:      &databaseName,
		SslMode:       postgresql_shared.PostgresSslModeRequire,
		SslClientCert: "new-cert",
		SslClientKey:  "",
		SslRootCert:   "new-root",
		CpuCount:      2,
	}

	existing.Update(incoming)

	assert.Equal(t, postgresql_shared.PostgresSslModeRequire, existing.SslMode)
	assert.Equal(t, "new-cert", existing.SslClientCert)
	assert.Equal(t, "enc:old-key", existing.SslClientKey)
	assert.Equal(t, "new-root", existing.SslRootCert)
}

func Test_Validate_SslConfig_EnforcesCertRules(t *testing.T) {
	databaseName := "appdb"
	baseModel := func() *PostgresqlLogicalDatabase {
		return &PostgresqlLogicalDatabase{
			Host:     "db.example.com",
			Port:     5432,
			Username: "appuser",
			Password: "secret",
			Database: &databaseName,
			CpuCount: 1,
		}
	}

	testCases := []struct {
		name          string
		mutate        func(*PostgresqlLogicalDatabase)
		expectedError string
	}{
		{
			name: "client cert without key",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = postgresql_shared.PostgresSslModeRequire
				p.SslClientCert = "cert"
			},
			expectedError: "client certificate and client key must be provided together",
		},
		{
			name: "client key without cert",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = postgresql_shared.PostgresSslModeRequire
				p.SslClientKey = "key"
			},
			expectedError: "client certificate and client key must be provided together",
		},
		{
			name: "certificates with SSL disabled",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = postgresql_shared.PostgresSslModeDisable
				p.SslClientCert = "cert"
				p.SslClientKey = "key"
			},
			expectedError: "SSL certificates require SSL to be enabled",
		},
		{
			name: "unknown SSL mode",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = "bogus"
			},
			expectedError: "invalid SSL mode",
		},
		{
			name: "client certificate pair with verify-full",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = postgresql_shared.PostgresSslModeVerifyFull
				p.SslClientCert = "cert"
				p.SslClientKey = "key"
				p.SslRootCert = "root"
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := baseModel()
			tc.mutate(model)

			err := model.Validate()

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func Test_Validate_WhenSslModeEmpty_DefaultsToDisable(t *testing.T) {
	databaseName := "appdb"
	model := &PostgresqlLogicalDatabase{
		Host:     "db.example.com",
		Port:     5432,
		Username: "appuser",
		Password: "secret",
		Database: &databaseName,
		CpuCount: 1,
	}

	err := model.Validate()

	assert.NoError(t, err)
	assert.Equal(t, postgresql_shared.PostgresSslModeDisable, model.SslMode)
}

type sizeTestSchema struct {
	SchemaName     string
	HeavyTableName string
	LightTableName string
}

func seedSizeTestSchema(t *testing.T, container *PostgresContainer) sizeTestSchema {
	t.Helper()

	suffix := uuid.New().String()[:8]
	seeded := sizeTestSchema{
		SchemaName:     "size_" + suffix,
		HeavyTableName: "heavy_" + suffix,
		LightTableName: "light_" + suffix,
	}

	_, err := container.DB.Exec(fmt.Sprintf(`
		CREATE SCHEMA %s;
		CREATE TABLE %s.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		CREATE TABLE %s.%s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		INSERT INTO %s.%s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
		INSERT INTO %s.%s (payload) VALUES ('light');
	`,
		seeded.SchemaName,
		seeded.SchemaName, seeded.HeavyTableName,
		seeded.SchemaName, seeded.LightTableName,
		seeded.SchemaName, seeded.HeavyTableName,
		seeded.SchemaName, seeded.LightTableName,
	))
	assert.NoError(t, err)

	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", seeded.SchemaName))
	})

	return seeded
}

func measureRawDbSizeMb(t *testing.T, pgModel *PostgresqlLogicalDatabase) float64 {
	t.Helper()

	sizeMb, err := pgModel.GetRawDbSizeMb(
		t.Context(),
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		nil,
	)
	assert.NoError(t, err)

	return sizeMb
}

func measureRelationsSizeMb(
	t *testing.T,
	container *PostgresContainer,
	relations ...relationName,
) float64 {
	t.Helper()

	var totalSizeMb float64

	for _, relation := range relations {
		var relationSizeMb float64
		err := container.DB.Get(&relationSizeMb, `
			SELECT pg_total_relation_size(format('%I.%I', $1::text, $2::text)::regclass)
				/ (1024.0 * 1024.0)
		`, relation.SchemaName, relation.TableName)
		assert.NoError(t, err)

		totalSizeMb += relationSizeMb
	}

	return totalSizeMb
}

func connectToPostgresContainer(t *testing.T, image string) *PostgresContainer {
	t.Helper()

	endpoint := containers.StartPostgres(t, image)

	return connectToPostgresEndpoint(t, endpoint)
}

func connectToPostgresEndpoint(t *testing.T, endpoint containers.Endpoint) *PostgresContainer {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		endpoint.Host, endpoint.Port,
		containers.PostgresUsername, containers.PostgresPassword, containers.PostgresDatabase)

	db, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)

	var versionStr string
	err = db.Get(&versionStr, "SELECT version()")
	assert.NoError(t, err)

	return &PostgresContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
		Database: containers.PostgresDatabase,
		DB:       db,
	}
}

func createPostgresModel(container *PostgresContainer) *PostgresqlLogicalDatabase {
	var versionStr string
	err := container.DB.Get(&versionStr, "SELECT version()")
	if err != nil {
		return nil
	}

	version := extractPostgresVersion(versionStr)

	return &PostgresqlLogicalDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		SslMode:  postgresql_shared.PostgresSslModeDisable,
		CpuCount: 1,
	}
}

func extractPostgresVersion(versionStr string) tools.PostgresqlVersion {
	if strings.Contains(versionStr, "PostgreSQL 12") {
		return tools.GetPostgresqlVersionEnum("12")
	} else if strings.Contains(versionStr, "PostgreSQL 13") {
		return tools.GetPostgresqlVersionEnum("13")
	} else if strings.Contains(versionStr, "PostgreSQL 14") {
		return tools.GetPostgresqlVersionEnum("14")
	} else if strings.Contains(versionStr, "PostgreSQL 15") {
		return tools.GetPostgresqlVersionEnum("15")
	} else if strings.Contains(versionStr, "PostgreSQL 16") {
		return tools.GetPostgresqlVersionEnum("16")
	} else if strings.Contains(versionStr, "PostgreSQL 17") {
		return tools.GetPostgresqlVersionEnum("17")
	}

	return tools.GetPostgresqlVersionEnum("16")
}

func databasusOnLoopback() *PostgresqlLogicalDatabase {
	databaseName := "databasus"

	return &PostgresqlLogicalDatabase{
		Host:     "127.0.0.1",
		Port:     5432,
		Username: "postgres",
		Password: "password",
		Database: &databaseName,
		CpuCount: 1,
	}
}

func Test_Validate_WhenPointedAtDatabasusOnLoopback_IsRejected(t *testing.T) {
	assert.Error(t, databasusOnLoopback().Validate())
}

func Test_Validate_WhenDatabasusOnLoopbackIsBehindARemoteBastion_IsAllowed(t *testing.T) {
	database := databasusOnLoopback()
	database.SshTunnel = sshtunnel.Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  "tunnelpassword",
	}

	assert.NoError(t, database.Validate())
}

// A bastion on this machine forwards straight back to the instance the guard protects, so enabling
// a tunnel must not be a way around it.
func Test_Validate_WhenDatabasusOnLoopbackIsBehindALocalBastion_IsRejected(t *testing.T) {
	for _, bastionHost := range []string{"127.0.0.1", "localhost", "host.docker.internal", "127.5.5.5"} {
		database := databasusOnLoopback()
		database.SshTunnel = sshtunnel.Config{
			IsEnabled: true,
			Host:      bastionHost,
			Port:      22,
			Username:  "tunneluser",
			AuthType:  sshtunnel.AuthTypePassword,
			Password:  "tunnelpassword",
		}

		assert.Error(t, database.Validate(), "bastion host %q must not bypass the guard", bastionHost)
	}
}

func createRlsTable(t *testing.T, container *PostgresContainer, schema, table string) {
	t.Helper()

	if schema != "public" {
		_, err := container.DB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schema))
		require.NoError(t, err)
	}

	_, err := container.DB.Exec(fmt.Sprintf(`CREATE TABLE %q.%q (id int)`, schema, table))
	require.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(`ALTER TABLE %q.%q ENABLE ROW LEVEL SECURITY`, schema, table))
	require.NoError(t, err)

	// The container is shared by every subtest of this version, so a leaked RLS table would
	// silently flip the verdict for all of them.
	t.Cleanup(func() {
		if _, cleanupErr := container.DB.Exec(
			fmt.Sprintf(`DROP TABLE IF EXISTS %q.%q CASCADE`, schema, table),
		); cleanupErr != nil {
			t.Logf("failed to drop row-level security table: %v", cleanupErr)
		}
		if schema != "public" {
			if _, cleanupErr := container.DB.Exec(
				fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schema),
			); cleanupErr != nil {
				t.Logf("failed to drop row-level security schema: %v", cleanupErr)
			}
		}
	})
}

func testShouldSuggestReadOnlyUserWhenRlsTableInScope(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	container := connectToPostgresEndpoint(t, endpoint)
	t.Cleanup(func() {
		if closeErr := container.DB.Close(); closeErr != nil {
			t.Logf("failed to close connection: %v", closeErr)
		}
	})

	createRlsTable(t, container, "public", "rls_in_scope")

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shouldSuggestReadOnlyUser, privileges, err := pgModel.ShouldSuggestReadOnlyUser(t.Context(), logger, nil)
	require.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "row-level security must suppress the suggestion")
	assert.NotEmpty(t, privileges, "the admin user is still privileged, only the suggestion is suppressed")
}

func testShouldSuggestReadOnlyUserWhenRlsTableOutsideIncludeSchemas(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	container := connectToPostgresEndpoint(t, endpoint)
	t.Cleanup(func() {
		if closeErr := container.DB.Close(); closeErr != nil {
			t.Logf("failed to close connection: %v", closeErr)
		}
	})

	createRlsTable(t, container, "rls_excluded_schema", "rls_out_of_scope")

	pgModel := createPostgresModel(container)
	pgModel.IncludeSchemas = []string{"public"}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shouldSuggestReadOnlyUser, _, err := pgModel.ShouldSuggestReadOnlyUser(t.Context(), logger, nil)
	require.NoError(t, err)
	assert.True(t, shouldSuggestReadOnlyUser, "row-level security outside the backed-up schemas is irrelevant")
}

func testCreateReadOnlyUserWhenRlsTableInScope(
	t *testing.T,
	endpoint containers.Endpoint,
	version string,
) {
	container := connectToPostgresEndpoint(t, endpoint)
	t.Cleanup(func() {
		if closeErr := container.DB.Close(); closeErr != nil {
			t.Logf("failed to close connection: %v", closeErr)
		}
	})

	createRlsTable(t, container, "public", "rls_blocks_creation")

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	const countDatabasusRolesQuery = `SELECT COUNT(*) FROM pg_roles WHERE rolname LIKE 'databasus-%'`

	var roleCountBeforeRefusal int
	require.NoError(t, container.DB.Get(&roleCountBeforeRefusal, countDatabasusRolesQuery))

	createdUsername, createdPassword, err := pgModel.CreateReadOnlyUser(t.Context(), logger, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public.rls_blocks_creation")
	assert.Empty(t, createdUsername)
	assert.Empty(t, createdPassword)

	var roleCountAfterRefusal int
	require.NoError(t, container.DB.Get(&roleCountAfterRefusal, countDatabasusRolesQuery))
	assert.Equal(t, roleCountBeforeRefusal, roleCountAfterRefusal, "refusal must not leave a role behind")
}
