package postgresql_logical

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

func createAndFillTableQuery(tableName string) string {
	return fmt.Sprintf(`
DROP TABLE IF EXISTS %s;

CREATE TABLE %s (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    value INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO %s (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300);
`, tableName, tableName, tableName)
}

type PostgresContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  string
	DB       *sqlx.DB
}

type TestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

type postgresVersion struct {
	name  string
	tag   string
	image string
}

const skippedFromDumpTableMinSizeMb = 15.0

func measureWholeDatabaseSizeMb(t *testing.T, db *sqlx.DB) float64 {
	t.Helper()

	var sizeMb float64
	err := db.Get(&sizeMb, "SELECT pg_database_size(current_database()) / (1024.0 * 1024.0)")
	assert.NoError(t, err)

	return sizeMb
}

type relationRef struct {
	SchemaName string
	TableName  string
}

func measureTableSizeMb(t *testing.T, db *sqlx.DB, relation relationRef) float64 {
	t.Helper()

	var sizeMb float64
	err := db.Get(&sizeMb, `
		SELECT pg_total_relation_size(format('%I.%I', $1::text, $2::text)::regclass) / (1024.0 * 1024.0)
	`, relation.SchemaName, relation.TableName)
	assert.NoError(t, err)

	return sizeMb
}

var postgresVersions = []postgresVersion{
	{"PostgreSQL 12", "12", "postgres:12"},
	{"PostgreSQL 13", "13", "postgres:13"},
	{"PostgreSQL 14", "14", "postgres:14"},
	{"PostgreSQL 15", "15", "postgres:15"},
	{"PostgreSQL 16", "16", "postgres:16"},
	{"PostgreSQL 17", "17", "postgres:17"},
	{"PostgreSQL 18", "18", "postgres:18"},
}

// Test_PostgresqlBackupRestore_AcrossSupportedVersions boots each PostgreSQL version once, runs
// every backup/restore test function against it as a subtest, then shuts it down before the next
// version. Only one matrix container is alive per package at a time. See docs/testing.md.
func Test_PostgresqlBackupRestore_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range postgresVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartPostgres(t, dbVersion.image)

			t.Run("Test_BackupAndRestorePostgresql_RestoreIsSuccesful", func(t *testing.T) {
				t.Run(
					"CPU=1 streamed",
					func(t *testing.T) { testBackupRestoreForVersion(t, endpoint, dbVersion.tag, 1) },
				)
				t.Run(
					"CPU=4 directory",
					func(t *testing.T) { testBackupRestoreForVersion(t, endpoint, dbVersion.tag, 4) },
				)
			})

			t.Run("Test_BackupAndRestorePostgresqlWithEncryption_RestoreIsSuccessful", func(t *testing.T) {
				testBackupRestoreWithEncryptionForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupPostgresql_SchemaSelection_AllSchemasWhenNoneSpecified", func(t *testing.T) {
				testSchemaSelectionAllSchemasForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupAndRestorePostgresql_WithExcludeExtensions_RestoreIsSuccessful", func(t *testing.T) {
				testBackupRestoreWithExcludeExtensionsForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run(
				"Test_BackupAndRestorePostgresql_WithoutExcludeExtensions_ExtensionsAreRecovered",
				func(t *testing.T) {
					testBackupRestoreWithoutExcludeExtensionsForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run("Test_BackupAndRestorePostgresql_WithRestoreOwnership_OwnerIsRestored", func(t *testing.T) {
				testBackupRestoreWithRestoreOwnershipForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupAndRestorePostgresql_WithRestorePrivileges_GrantIsRestored", func(t *testing.T) {
				testBackupRestoreWithRestorePrivilegesForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupPostgresql_SchemaSelection_OnlySpecifiedSchemas", func(t *testing.T) {
				testSchemaSelectionOnlySpecifiedSchemasForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupAndRestorePostgresql_WithExcludeTables_ExcludedTablesNotRestored", func(t *testing.T) {
				testBackupRestoreWithExcludeTablesForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run(
				"Test_BackupAndRestorePostgresql_WithSchemasAndExcludeTables_OnlyIncludedSchemasMinusExcludedTablesRestored",
				func(t *testing.T) {
					testSchemasWithExcludeTablesForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run(
				"Test_BackupAndRestorePostgresql_WithExcludeTableData_TableRestoredEmptyWithIndexes",
				func(t *testing.T) {
					testBackupRestoreWithExcludeTableDataForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run(
				"Test_BackupAndRestorePostgresql_WithExclusionPatterns_MatchLikePgDump",
				func(t *testing.T) {
					testBackupRestoreWithExclusionPatternsForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run(
				"Test_BackupAndRestorePostgresql_WithExcludeTableDataOnPartitions_OnlyMatchedPartitionsEmptied",
				func(t *testing.T) {
					testBackupRestoreWithExcludeTableDataOnPartitionsForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run(
				"Test_BackupAndRestorePostgresql_WithSchemasAndExcludeTableData_DataExcludedOnlyInIncludedSchemas",
				func(t *testing.T) {
					testBackupRestoreWithSchemasAndExcludeTableDataForVersion(t, endpoint, dbVersion.tag)
				},
			)

			t.Run("Test_BackupAndRestorePostgresql_WithReadOnlyUser_RestoreIsSuccessful", func(t *testing.T) {
				testBackupRestoreWithReadOnlyUserForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupPostgresql_WhenUserCannotReadSequences_BackupFails", func(t *testing.T) {
				testBackupFailsWhenUserCannotReadSequencesForVersion(t, endpoint, dbVersion.tag)
			})

			t.Run("Test_BackupAndRestorePostgresql_WithSkipUserMappings_RestoreIsSuccessful", func(t *testing.T) {
				testBackupRestoreSkipUserMappingsForVersion(t, endpoint, dbVersion.tag)
			})
		})
	}
}

func testBackupRestoreForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string, cpuCount int) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	tableName := fmt.Sprintf("test_data_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithCpuCountViaAPI(
		t, router, "Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		cpuCount,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)
	logicaltesting.AssertBackupSizeMatchesDownloadedBytes(t, router, backup, user.Token)

	newDBName := fmt.Sprintf("restoreddb_%s_cpu%d_%s", pgVersion, cpuCount, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreWithCpuCountViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		cpuCount,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		fmt.Sprintf(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')",
			tableName,
		),
	)
	assert.NoError(t, err)
	assert.True(
		t,
		tableExists,
		fmt.Sprintf("Table '%s' should exist in restored database", tableName),
	)

	verifyDataIntegrity(t, container.DB, newDB, tableName)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testSchemaSelectionAllSchemasForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	_, err = container.DB.Exec(`
		DROP TABLE IF EXISTS public.public_table;
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;

		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);

		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS public.public_table;
			DROP SCHEMA IF EXISTS schema_a CASCADE;
			DROP SCHEMA IF EXISTS schema_b CASCADE;
		`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Schema Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithSchemasViaAPI(
		t, router, "All Schemas Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		nil,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_all_schemas_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var publicTableExists bool
	err = newDB.Get(&publicTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'public_table'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, publicTableExists, "public.public_table should exist in restored database")

	var schemaATableExists bool
	err = newDB.Get(&schemaATableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_a' AND table_name = 'table_a'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaATableExists, "schema_a.table_a should exist in restored database")

	var schemaBTableExists bool
	err = newDB.Get(&schemaBTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_b' AND table_name = 'table_b'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaBTableExists, "schema_b.table_b should exist in restored database")

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithExcludeExtensionsForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	// Create table with uuid-ossp extension and add a comment on the extension
	// The comment is important to test that COMMENT ON EXTENSION statements are also excluded
	_, err = container.DB.Exec(`
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		COMMENT ON EXTENSION "uuid-ossp" IS 'Test comment on uuid-ossp extension';

		DROP TABLE IF EXISTS test_extension_data;
		CREATE TABLE test_extension_data (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		INSERT INTO test_extension_data (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS test_extension_data;
			DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Extension Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Extension Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_exclude_ext_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Pre-install the extension in the target database (simulating managed service behavior)
	_, err = newDB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	assert.NoError(t, err)

	// Restore with isExcludeExtensions=true
	createRestoreWithOptionsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		true, // isExcludeExtensions
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	// Verify the table was restored
	var tableExists bool
	err = newDB.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'test_extension_data'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, tableExists, "test_extension_data should exist in restored database")

	// Verify data was restored
	var count int
	err = newDB.Get(&count, `SELECT COUNT(*) FROM test_extension_data`)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 rows after restore")

	// Verify extension still works (uuid_generate_v4 should work)
	var newUUID string
	err = newDB.Get(&newUUID, `SELECT uuid_generate_v4()::text`)
	assert.NoError(t, err)
	assert.NotEmpty(t, newUUID, "uuid_generate_v4 should work")

	// Cleanup
	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

// testBackupRestoreSkipUserMappingsForVersion backs up under a role that cannot read a user
// mapping's options (so pg_dump emits a bare CREATE USER MAPPING) with IsSkipUserMappings enabled,
// and asserts the restore succeeds with the mapping excluded. postgres_fdw stands in for
// oracle_fdw, which is not installable in CI.
func testBackupRestoreSkipUserMappingsForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	suffix := uuid.New().String()[:8]
	limitedUsername := fmt.Sprintf("um_backup_%s", suffix)
	limitedPassword := "limitedpassword123"
	serverName := fmt.Sprintf("um_srv_%s", suffix)

	setupStatements := []string{
		`CREATE TABLE skip_um_data (id INT PRIMARY KEY, name TEXT NOT NULL)`,
		`INSERT INTO skip_um_data (id, name) VALUES (1, 'row1'), (2, 'row2'), (3, 'row3')`,
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
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"`, limitedUsername),
		// The backup covers the whole database, and the version's subtests share this server
		// (docs/testing.md, "Isolation"), so the role needs every relation in public, not just this test's table.
		fmt.Sprintf(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%s"`, limitedUsername),
		fmt.Sprintf(`GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "%s"`, limitedUsername),
	}

	for _, statement := range setupStatements {
		_, err = container.DB.Exec(statement)
		assert.NoError(t, err)
	}

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP SERVER IF EXISTS %s CASCADE`, serverName))
		_, _ = container.DB.Exec(`DROP TABLE IF EXISTS skip_um_data CASCADE`)
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s"`, limitedUsername))
		_, _ = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, limitedUsername))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Skip User Mappings Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	// Created under the limited role with IsSkipUserMappings enabled, so TestConnection's
	// unreadable-user-mapping gate is bypassed and the backup runs as that role.
	database := createDatabaseWithSkipUserMappingsViaAPI(
		t, router, "Skip User Mappings Database", workspace.ID,
		container.Host, container.Port,
		limitedUsername, limitedPassword, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_skip_um_%s_%s", pgVersion, suffix)
	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Restore connects as the admin so the dumped extension/server can be created; the source
	// database's persisted IsSkipUserMappings flag drives the user-mapping exclusion.
	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var rowCount int
	err = newDB.Get(&rowCount, `SELECT COUNT(*) FROM skip_um_data`)
	assert.NoError(t, err)
	assert.Equal(t, 3, rowCount, "table data should be restored")

	var userMappingCount int
	err = newDB.Get(&userMappingCount,
		`SELECT COUNT(*) FROM pg_user_mappings um
		 JOIN pg_foreign_server s ON s.oid = um.srvid
		 WHERE s.srvname = $1`, serverName)
	assert.NoError(t, err)
	assert.Equal(t, 0, userMappingCount, "user mapping should be excluded from the restore")

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithoutExcludeExtensionsForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	// Create table with uuid-ossp extension
	_, err = container.DB.Exec(`
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		DROP TABLE IF EXISTS test_extension_recovery;
		CREATE TABLE test_extension_recovery (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		INSERT INTO test_extension_recovery (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS test_extension_recovery;
			DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Extension Recovery Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Extension Recovery Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_with_ext_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Verify extension does NOT exist before restore
	var extensionExistsBefore bool
	err = newDB.Get(&extensionExistsBefore, `
		SELECT EXISTS (
			SELECT FROM pg_extension WHERE extname = 'uuid-ossp'
		)
	`)
	assert.NoError(t, err)
	assert.False(t, extensionExistsBefore, "Extension should NOT exist before restore")

	// Restore with isExcludeExtensions=false (extensions should be recovered)
	createRestoreWithOptionsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		false, // isExcludeExtensions = false means extensions ARE included
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	// Verify the extension was recovered
	var extensionExists bool
	err = newDB.Get(&extensionExists, `
		SELECT EXISTS (
			SELECT FROM pg_extension WHERE extname = 'uuid-ossp'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, extensionExists, "Extension 'uuid-ossp' should be recovered during restore")

	// Verify the table was restored
	var tableExists bool
	err = newDB.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'test_extension_recovery'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, tableExists, "test_extension_recovery should exist in restored database")

	// Verify data was restored
	var count int
	err = newDB.Get(&count, `SELECT COUNT(*) FROM test_extension_recovery`)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Should have 3 rows after restore")

	// Verify extension works (uuid_generate_v4 should work)
	var newUUID string
	err = newDB.Get(&newUUID, `SELECT uuid_generate_v4()::text`)
	assert.NoError(t, err)
	assert.NotEmpty(t, newUUID, "uuid_generate_v4 should work after extension recovery")

	// Cleanup
	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithRestoreOwnershipForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	suffix := uuid.New().String()[:8]
	ownerRole := fmt.Sprintf("test_owner_%s", suffix)
	tableName := fmt.Sprintf("test_ownership_data_%s", suffix)

	_, err = container.DB.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS %s;
		DROP ROLE IF EXISTS %s;
		CREATE ROLE %s LOGIN PASSWORD 'placeholder';
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		);
		ALTER TABLE %s OWNER TO %s;
		INSERT INTO %s (name, value) VALUES ('a', 1), ('b', 2), ('c', 3);
	`, tableName, ownerRole, ownerRole, tableName, tableName, ownerRole, tableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
		_, _ = container.DB.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s;", ownerRole))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Ownership Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Ownership Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_ownership_%s_%s", pgVersion, suffix)
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreWithRestoreFlagsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		true,  // isRestoreOwnership
		false, // isRestorePrivileges
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var restoredOwner string
	err = newDB.Get(&restoredOwner, `
		SELECT tableowner FROM pg_tables
		WHERE schemaname = 'public' AND tablename = $1
	`, tableName)
	assert.NoError(t, err)
	assert.Equal(
		t,
		ownerRole,
		restoredOwner,
		"Restored table should retain its original owner when isRestoreOwnership=true",
	)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithRestorePrivilegesForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	suffix := uuid.New().String()[:8]
	granteeRole := fmt.Sprintf("test_grantee_%s", suffix)
	tableName := fmt.Sprintf("test_privileges_data_%s", suffix)

	_, err = container.DB.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS %s;
		DROP ROLE IF EXISTS %s;
		CREATE ROLE %s LOGIN PASSWORD 'placeholder';
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		);
		INSERT INTO %s (name, value) VALUES ('a', 1), ('b', 2), ('c', 3);
		GRANT SELECT, INSERT ON %s TO %s;
	`, tableName, granteeRole, granteeRole, tableName, tableName, tableName, granteeRole))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
		_, _ = container.DB.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s;", granteeRole))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Privileges Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Privileges Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_privileges_%s_%s", pgVersion, suffix)
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreWithRestoreFlagsViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		false, // isRestoreOwnership
		true,  // isRestorePrivileges
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var grantedPrivileges []string
	err = newDB.Select(&grantedPrivileges, `
		SELECT privilege_type FROM information_schema.role_table_grants
		WHERE grantee = $1
		  AND table_schema = 'public'
		  AND table_name = $2
		ORDER BY privilege_type
	`, granteeRole, tableName)
	assert.NoError(t, err)
	assert.Contains(
		t,
		grantedPrivileges,
		"SELECT",
		"Restored table should retain SELECT grant when isRestorePrivileges=true",
	)
	assert.Contains(
		t,
		grantedPrivileges,
		"INSERT",
		"Restored table should retain INSERT grant when isRestorePrivileges=true",
	)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithReadOnlyUserForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	tableName := fmt.Sprintf("test_data_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ReadOnly Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	readOnlyUser := createReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateDatabaseCredentialsViaAPI(
		t, router, database,
		readOnlyUser.Username, readOnlyUser.Password,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, updatedDatabase.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, updatedDatabase.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, updatedDatabase.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restoreddb_readonly_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		fmt.Sprintf(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')",
			tableName,
		),
	)
	assert.NoError(t, err)
	assert.True(
		t,
		tableExists,
		fmt.Sprintf("Table '%s' should exist in restored database", tableName),
	)

	verifyDataIntegrity(t, container.DB, newDB, tableName)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+updatedDatabase.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

// pg_dump 18.0 and 18.1 emitted wrong sequence values instead of failing when the
// dumping role could not read a sequence, so a backup completed while silently
// losing every sequence position (issue #725, fixed upstream in PostgreSQL 18.2).
func testBackupFailsWhenUserCannotReadSequencesForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	uniqueSuffix := uuid.New().String()[:8]
	tableName := fmt.Sprintf("test_data_%s", uniqueSuffix)
	sequenceName := tableName + "_id_seq"
	restrictedUsername := fmt.Sprintf("no_sequence_reader_%s", uniqueSuffix)
	restrictedPassword := uuid.New().String()

	_, err = container.DB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(`
		CREATE ROLE %[1]s LOGIN PASSWORD '%[2]s';
		GRANT CONNECT ON DATABASE %[3]s TO %[1]s;
		GRANT USAGE ON SCHEMA public TO %[1]s;
		GRANT SELECT ON ALL TABLES IN SCHEMA public TO %[1]s;
		GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO %[1]s;
	`, restrictedUsername, restrictedPassword, container.Database))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf(`
			DROP TABLE IF EXISTS %s;
			REASSIGN OWNED BY %s TO %s;
			DROP OWNED BY %s;
			DROP ROLE IF EXISTS %s;
		`, tableName, restrictedUsername, container.Username, restrictedUsername, restrictedUsername))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Sequence Privileges Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Sequence Privileges Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	restrictedDatabase := updateDatabaseCredentialsViaAPI(
		t, router, database,
		restrictedUsername, restrictedPassword,
		user.Token,
	)

	// The sequence is revoked only after the database is saved, because privileges drift on a
	// live server and a backup must fail loudly when they do, rather than emit a partial dump.
	_, err = container.DB.Exec(fmt.Sprintf(
		`REVOKE ALL ON SEQUENCE %s FROM PUBLIC, %s`,
		sequenceName,
		restrictedUsername,
	))
	assert.NoError(t, err)

	logicaltesting.EnableBackupsViaAPI(
		t, router, restrictedDatabase.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, restrictedDatabase.ID, user.Token)

	backup := logicaltesting.WaitForBackupTerminalStatus(
		t, router, restrictedDatabase.ID, user.Token, 5*time.Minute,
	)

	assert.Equal(
		t,
		backups_core_logical.BackupStatusFailed,
		backup.Status,
		"PostgreSQL %s: backup must fail when the dumping role cannot read sequences", pgVersion,
	)

	// PG 18 says "failed to get data for sequence ..." where earlier majors say
	// "permission denied for sequence ..."; both name the sequence they choked on.
	if assert.NotNil(t, backup.FailMessage) {
		assert.Contains(t, *backup.FailMessage, sequenceName)
	}

	_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+restrictedDatabase.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testSchemaSelectionOnlySpecifiedSchemasForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	_, err = container.DB.Exec(`
		DROP TABLE IF EXISTS public.public_table;
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;

		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);

		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
		INSERT INTO schema_b.table_b (data)
			SELECT repeat('x', 1024) FROM generate_series(1, 20000);
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS public.public_table;
			DROP SCHEMA IF EXISTS schema_a CASCADE;
			DROP SCHEMA IF EXISTS schema_b CASCADE;
		`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Schema Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithSchemasViaAPI(
		t, router, "Specific Schemas Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		[]string{"public", "schema_a"},
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	skippedSchemaTableSizeMb := measureTableSizeMb(
		t, container.DB,
		relationRef{SchemaName: "schema_b", TableName: "table_b"},
	)
	assert.Greater(t, skippedSchemaTableSizeMb, skippedFromDumpTableMinSizeMb)
	assert.Greater(t, backup.BackupRawDbSizeMb, 0.0)
	assert.LessOrEqual(
		t,
		backup.BackupRawDbSizeMb+skippedSchemaTableSizeMb,
		measureWholeDatabaseSizeMb(t, container.DB),
		"recorded raw db size must not count schemas left out of the dump",
	)

	newDBName := fmt.Sprintf("restored_specific_schemas_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var publicTableExists bool
	err = newDB.Get(&publicTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'public_table'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, publicTableExists, "public.public_table should exist (was included)")

	var schemaATableExists bool
	err = newDB.Get(&schemaATableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_a' AND table_name = 'table_a'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaATableExists, "schema_a.table_a should exist (was included)")

	var schemaBTableExists bool
	err = newDB.Get(&schemaBTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_b' AND table_name = 'table_b'
		)
	`)
	assert.NoError(t, err)
	assert.False(t, schemaBTableExists, "schema_b.table_b should NOT exist (was excluded)")

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithExcludeTablesForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	suffix := uuid.New().String()[:8]
	keepTable := fmt.Sprintf("keep_me_%s", suffix)
	dropTable := fmt.Sprintf("drop_me_%s", suffix)

	_, err = container.DB.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS public.%s;
		DROP TABLE IF EXISTS public.%s;
		CREATE TABLE public.%s (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE public.%s (id SERIAL PRIMARY KEY, data TEXT);
		INSERT INTO public.%s (data) VALUES ('keep_value');
		INSERT INTO public.%s (data) VALUES ('drop_value');
		INSERT INTO public.%s (data) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
	`, keepTable, dropTable, keepTable, dropTable, keepTable, dropTable, dropTable))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP TABLE IF EXISTS public.%s; DROP TABLE IF EXISTS public.%s;",
				keepTable, dropTable),
		)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Exclude Tables Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Exclude Tables Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	database.PostgresqlLogical.ExcludeTables = []string{dropTable}
	updateDatabaseViaAPI(t, router, database, user.Token)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	excludedTableSizeMb := measureTableSizeMb(
		t, container.DB,
		relationRef{SchemaName: "public", TableName: dropTable},
	)
	assert.Greater(t, excludedTableSizeMb, skippedFromDumpTableMinSizeMb)
	assert.Greater(t, backup.BackupRawDbSizeMb, 0.0)
	assert.LessOrEqual(
		t,
		backup.BackupRawDbSizeMb+excludedTableSizeMb,
		measureWholeDatabaseSizeMb(t, container.DB),
		"recorded raw db size must not count the excluded table",
	)

	newDBName := fmt.Sprintf("restored_excl_tables_%s_%s", pgVersion, suffix)
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var keepExists bool
	err = newDB.Get(&keepExists, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = '%s'
		)
	`, keepTable))
	assert.NoError(t, err)
	assert.True(t, keepExists, "kept table should exist in restored database")

	var dropExists bool
	err = newDB.Get(&dropExists, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = '%s'
		)
	`, dropTable))
	assert.NoError(t, err)
	assert.False(t, dropExists, "excluded table should NOT exist in restored database")

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testSchemasWithExcludeTablesForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL container: %v", err)
	}
	defer container.DB.Close()

	_, err = container.DB.Exec(`
		DROP TABLE IF EXISTS public.public_table;
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;

		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.skip_me (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);

		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_a.skip_me (data) VALUES ('skipped_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
	`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`
			DROP TABLE IF EXISTS public.public_table;
			DROP SCHEMA IF EXISTS schema_a CASCADE;
			DROP SCHEMA IF EXISTS schema_b CASCADE;
		`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Schemas+Exclude Tables Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseWithSchemasViaAPI(
		t, router, "Schemas+Exclude Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		[]string{"public", "schema_a"},
		user.Token,
	)

	database.PostgresqlLogical.IncludeSchemas = []string{"public", "schema_a"}
	database.PostgresqlLogical.ExcludeTables = []string{"schema_a.skip_me"}
	updateDatabaseViaAPI(t, router, database, user.Token)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := fmt.Sprintf("restored_schemas_excl_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var publicTableExists bool
	err = newDB.Get(&publicTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'public_table'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, publicTableExists, "public.public_table should exist (schema included)")

	var schemaATableExists bool
	err = newDB.Get(&schemaATableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_a' AND table_name = 'table_a'
		)
	`)
	assert.NoError(t, err)
	assert.True(t, schemaATableExists, "schema_a.table_a should exist (schema included, not excluded)")

	var skipMeExists bool
	err = newDB.Get(&skipMeExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_a' AND table_name = 'skip_me'
		)
	`)
	assert.NoError(t, err)
	assert.False(
		t,
		skipMeExists,
		"schema_a.skip_me should NOT exist (excluded even though schema included)",
	)

	var schemaBTableExists bool
	err = newDB.Get(&schemaBTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'schema_b' AND table_name = 'table_b'
		)
	`)
	assert.NoError(t, err)
	assert.False(t, schemaBTableExists, "schema_b.table_b should NOT exist (schema not included)")

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func testBackupRestoreWithEncryptionForVersion(t *testing.T, endpoint containers.Endpoint, pgVersion string) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	tableName := fmt.Sprintf("test_data_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionEncrypted, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)
	assert.Equal(t, backups_core_enums.BackupEncryptionEncrypted, backup.Encryption)

	newDBName := fmt.Sprintf("restoreddb_encrypted_%s", uuid.New().String()[:8])
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	}()

	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists bool
	err = newDB.Get(
		&tableExists,
		fmt.Sprintf(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')",
			tableName,
		),
	)
	assert.NoError(t, err)
	assert.True(
		t,
		tableExists,
		fmt.Sprintf("Table '%s' should exist in restored database", tableName),
	)

	verifyDataIntegrity(t, container.DB, newDB, tableName)

	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func waitForRestoreCompletion(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	token string,
	timeout time.Duration,
) *restores_core.Restore {
	startTime := time.Now()
	pollInterval := 500 * time.Millisecond

	for {
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout waiting for restore completion after %v", timeout)
		}

		var restores []*restores_core.Restore
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/restores/%s", backupID.String()),
			"Bearer "+token,
			http.StatusOK,
			&restores,
		)

		for _, restore := range restores {
			if restore.Status == restores_core.RestoreStatusCompleted {
				return restore
			}
			if restore.Status == restores_core.RestoreStatusFailed {
				failMsg := "unknown error"
				if restore.FailMessage != nil {
					failMsg = *restore.FailMessage
				}
				t.Fatalf("Restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func createDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	token string,
) *databases.Database {
	return createDatabaseWithCpuCountViaAPI(
		t, router, name, workspaceID,
		host, port, username, password, database,
		1,
		token,
	)
}

func createDatabaseWithCpuCountViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	cpuCount int,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			CpuCount: cpuCount,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createDatabaseWithSkipUserMappingsViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{
			Host:               host,
			Port:               port,
			Username:           username,
			Password:           password,
			Database:           &database,
			CpuCount:           1,
			IsSkipUserMappings: true,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	token string,
) {
	createRestoreWithCpuCountViaAPI(
		t,
		router,
		backupID,
		host,
		port,
		username,
		password,
		database,
		1,
		token,
	)
}

func createRestoreWithCpuCountViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	cpuCount int,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &pgtypes.PostgresqlLogicalDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			CpuCount: cpuCount,
		},
	}

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
		"Bearer "+token,
		request,
		http.StatusOK,
	)
}

func createRestoreWithOptionsViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	isExcludeExtensions bool,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &pgtypes.PostgresqlLogicalDatabase{
			Host:                host,
			Port:                port,
			Username:            username,
			Password:            password,
			Database:            &database,
			IsExcludeExtensions: isExcludeExtensions,
			CpuCount:            1,
		},
	}

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
		"Bearer "+token,
		request,
		http.StatusOK,
	)
}

func createRestoreWithRestoreFlagsViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	isRestoreOwnership bool,
	isRestorePrivileges bool,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &pgtypes.PostgresqlLogicalDatabase{
			Host:                host,
			Port:                port,
			Username:            username,
			Password:            password,
			Database:            &database,
			IsRestoreOwnership:  isRestoreOwnership,
			IsRestorePrivileges: isRestorePrivileges,
			CpuCount:            1,
		},
	}

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
		"Bearer "+token,
		request,
		http.StatusOK,
	)
}

func createDatabaseWithSchemasViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	includeSchemas []string,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{
			Host:           host,
			Port:           port,
			Username:       username,
			Password:       password,
			Database:       &database,
			IncludeSchemas: includeSchemas,
			CpuCount:       1,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf(
			"Failed to create database with schemas. Status: %d, Body: %s",
			w.Code,
			w.Body.String(),
		)
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func verifyDataIntegrity(t *testing.T, originalDB, restoredDB *sqlx.DB, tableName string) {
	var originalData []TestDataItem
	var restoredData []TestDataItem

	err := originalDB.Select(&originalData, fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName))
	assert.NoError(t, err)

	err = restoredDB.Select(&restoredData, fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName))
	assert.NoError(t, err)

	assert.Equal(t, len(originalData), len(restoredData), "Should have same number of rows")

	if len(originalData) > 0 && len(restoredData) > 0 {
		for i := range originalData {
			assert.Equal(t, originalData[i].ID, restoredData[i].ID, "ID should match")
			assert.Equal(t, originalData[i].Name, restoredData[i].Name, "Name should match")
			assert.Equal(t, originalData[i].Value, restoredData[i].Value, "Value should match")
		}
	}
}

func createReadOnlyUserViaAPI(
	t *testing.T,
	router *gin.Engine,
	databaseID uuid.UUID,
	token string,
) *databases.CreateReadOnlyUserResponse {
	var database databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/databases/%s", databaseID.String()),
		"Bearer "+token,
		http.StatusOK,
		&database,
	)

	var response databases.CreateReadOnlyUserResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create-readonly-user",
		"Bearer "+token,
		database,
		http.StatusOK,
		&response,
	)

	return &response
}

func updateDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	token string,
) *databases.Database {
	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}

func updateDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.PostgresqlLogical.Username = username
	database.PostgresqlLogical.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}

func connectToPostgresContainer(t *testing.T, image string) (*PostgresContainer, error) {
	t.Helper()

	endpoint := containers.StartPostgres(t, image)

	return connectToPostgresEndpoint(t, endpoint)
}

func connectToPostgresEndpoint(t *testing.T, endpoint containers.Endpoint) (*PostgresContainer, error) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		endpoint.Host, endpoint.Port,
		containers.PostgresUsername, containers.PostgresPassword, containers.PostgresDatabase)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
		Database: containers.PostgresDatabase,
		DB:       db,
	}, nil
}
