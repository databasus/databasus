package mysql_logical

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

const dropMysqlTestTableQuery = `DROP TABLE IF EXISTS test_data`

const createMysqlTestTableQuery = `
CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

const insertMysqlTestDataQuery = `
INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300)`

type MysqlContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MysqlVersion
	DB       *sqlx.DB
}

type MysqlTestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

type mysqlVersion struct {
	name    string
	version tools.MysqlVersion
	image   string
}

var mysqlVersions = []mysqlVersion{
	{"MySQL 5.7", tools.MysqlVersion57, "mysql:5.7"},
	{"MySQL 8.0", tools.MysqlVersion80, "mysql:8.0"},
	{"MySQL 8.4", tools.MysqlVersion84, "mysql:8.4"},
	{"MySQL 9", tools.MysqlVersion9, "mysql:9"},
	{"MySQL 26", tools.MysqlVersion26, "mysql:26.7"},
}

// Test_MysqlBackupRestore_AcrossSupportedVersions boots each MySQL version once, runs every
// backup/restore test function against it as a subtest, then shuts it down before the next version.
// Only one matrix container is alive per package at a time. See docs/testing.md.
func Test_MysqlBackupRestore_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mysqlVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMysql(t, dbVersion.image)

			t.Run("Test_BackupAndRestoreMysql_RestoreIsSuccessful", func(t *testing.T) {
				testMysqlBackupRestoreForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMysqlWithEncryption_RestoreIsSuccessful", func(t *testing.T) {
				testMysqlBackupRestoreWithEncryptionForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMysql_WithExcludeTables_ExcludedTablesNotRestored", func(t *testing.T) {
				testMysqlBackupRestoreWithExcludeTablesForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMysql_WithReadOnlyUser_RestoreIsSuccessful", func(t *testing.T) {
				testMysqlBackupRestoreWithReadOnlyUserForVersion(t, endpoint, dbVersion.version)
			})
		})
	}
}

func testMysqlBackupRestoreForVersion(t *testing.T, endpoint containers.Endpoint, mysqlVersion tools.MysqlVersion) {
	container, err := connectToMysqlEndpoint(t, endpoint, mysqlVersion)
	if err != nil {
		t.Fatalf("MySQL %s test failed: %v", mysqlVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MySQL Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
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

	newDBName := "restoreddb_mysql"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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

func testMysqlBackupRestoreWithEncryptionForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mysqlVersion tools.MysqlVersion,
) {
	container, err := connectToMysqlEndpoint(t, endpoint, mysqlVersion)
	if err != nil {
		t.Fatalf("MySQL %s test failed: %v", mysqlVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MySQL Encrypted Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL Encrypted Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
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
	logicaltesting.AssertBackupSizeMatchesDownloadedBytes(t, router, backup, user.Token)

	newDBName := "restoreddb_mysql_encrypted"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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

func testMysqlBackupRestoreWithReadOnlyUserForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mysqlVersion tools.MysqlVersion,
) {
	container, err := connectToMysqlEndpoint(t, endpoint, mysqlVersion)
	if err != nil {
		t.Fatalf("MySQL %s test failed: %v", mysqlVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MySQL ReadOnly Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	readOnlyUser := createMysqlReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateMysqlDatabaseCredentialsViaAPI(
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

	newDBName := "restoreddb_mysql_readonly"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMysqlDataIntegrity(t, container.DB, newDB)

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

func testMysqlBackupRestoreWithExcludeTablesForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mysqlVersion tools.MysqlVersion,
) {
	container, err := connectToMysqlEndpoint(t, endpoint, mysqlVersion)
	if err != nil {
		t.Fatalf("MySQL %s test failed: %v", mysqlVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMysqlTestData(t, container.DB)

	_, err = container.DB.Exec(`DROP TABLE IF EXISTS extra_table`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`
		CREATE TABLE extra_table (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`INSERT INTO extra_table (data) VALUES ('drop_me')`)
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(`DROP TABLE IF EXISTS extra_table`)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MySQL Exclude Tables Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMysqlDatabaseViaAPI(
		t, router, "MySQL Exclude Tables Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	database.Mysql.ExcludeTables = []string{"\nextra_table"}
	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+user.Token,
		database,
	)
	if w.Code != http.StatusOK {
		t.Fatalf(
			"Failed to update database with ExcludeTables. Status: %d, Body: %s",
			w.Code,
			w.Body.String(),
		)
	}

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mysql_excl_tables"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMysqlRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var keepExists int
	err = newDB.Get(
		&keepExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, keepExists, "test_data table should exist in restored database")

	var dropExists int
	err = newDB.Get(
		&dropExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'extra_table'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		0,
		dropExists,
		"extra_table should NOT exist in restored database (was excluded)",
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

func createMysqlDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	version tools.MysqlVersion,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypeMysql,
		Mysql: &mysqltypes.MysqlDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			Version:  version,
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
		t.Fatalf("Failed to create MySQL database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createMysqlRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	version tools.MysqlVersion,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		MysqlDatabase: &mysqltypes.MysqlDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			Version:  version,
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

func waitForMysqlRestoreCompletion(
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
			t.Fatalf("Timeout waiting for MySQL restore completion after %v", timeout)
		}

		var restoresList []*restores_core.Restore
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/restores/%s", backupID.String()),
			"Bearer "+token,
			http.StatusOK,
			&restoresList,
		)

		for _, restore := range restoresList {
			if restore.Status == restores_core.RestoreStatusCompleted {
				return restore
			}
			if restore.Status == restores_core.RestoreStatusFailed {
				failMsg := "unknown error"
				if restore.FailMessage != nil {
					failMsg = *restore.FailMessage
				}
				t.Fatalf("MySQL restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func verifyMysqlDataIntegrity(t *testing.T, originalDB, restoredDB *sqlx.DB) {
	var originalData []MysqlTestDataItem
	var restoredData []MysqlTestDataItem

	err := originalDB.Select(
		&originalData,
		"SELECT id, name, value, created_at FROM test_data ORDER BY id",
	)
	assert.NoError(t, err)

	err = restoredDB.Select(
		&restoredData,
		"SELECT id, name, value, created_at FROM test_data ORDER BY id",
	)
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

func connectToMysqlContainer(
	t *testing.T,
	image string,
	version tools.MysqlVersion,
) (*MysqlContainer, error) {
	endpoint := containers.StartMysql(t, image)

	return connectToMysqlEndpoint(t, endpoint, version)
}

func connectToMysqlEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) (*MysqlContainer, error) {
	dbName := containers.MysqlDatabase
	username := "root"
	password := containers.MysqlRootPassword

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, endpoint.Host, endpoint.Port, dbName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	return &MysqlContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}, nil
}

func setupMysqlTestData(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec(dropMysqlTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(createMysqlTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(insertMysqlTestDataQuery)
	assert.NoError(t, err)
}

func createMysqlReadOnlyUserViaAPI(
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

func updateMysqlDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.Mysql.Username = username
	database.Mysql.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update MySQL database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}
