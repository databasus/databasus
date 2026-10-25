package mariadb_logical

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
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

const dropMariadbTestTableQuery = `DROP TABLE IF EXISTS test_data`

const createMariadbTestTableQuery = `
CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

const insertMariadbTestDataQuery = `
INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300)`

type MariadbContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MariadbVersion
	DB       *sqlx.DB
}

type MariadbTestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

type mariadbVersion struct {
	name             string
	version          tools.MariadbVersion
	image            string
	runsExcludeTests bool // exclude-tables/exclude-events run only on 10.11 and 11.4
}

// 5.5 and 10.1 are here because they are the only servers the legacy client
// tier serves: without them nothing dumps through that tier.
var mariadbVersions = []mariadbVersion{
	{"MariaDB 5.5", tools.MariadbVersion55, "mariadb:5.5", false},
	{"MariaDB 10.1", tools.MariadbVersion101, "mariadb:10.1", false},
	{"MariaDB 10.6", tools.MariadbVersion106, "mariadb:10.6", false},
	{"MariaDB 10.11", tools.MariadbVersion1011, "mariadb:10.11", true},
	{"MariaDB 11.4", tools.MariadbVersion114, "mariadb:11.4", true},
	{"MariaDB 11.8", tools.MariadbVersion118, "mariadb:11.8", false},
	{"MariaDB 12.0", tools.MariadbVersion120, "mariadb:12.0", false},
	{"MariaDB 13.0", tools.MariadbVersion130, "mariadb:13.0", true},
}

// Test_MariadbBackupRestore_AcrossSupportedVersions boots each MariaDB version once, runs every
// backup/restore test function against it as a subtest, then shuts it down before the next version.
// Only one matrix container is alive per package at a time.
// Test_MariadbBackupRestore_AcrossSupportedVersions boots each MariaDB version once, runs every
// backup/restore test function against it as a subtest, then shuts it down before the next version.
// Only one matrix container is alive per package at a time. See docs/testing.md.
func Test_MariadbBackupRestore_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mariadbVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMariadb(t, dbVersion.image)

			t.Run("Test_BackupAndRestoreMariadb_RestoreIsSuccessful", func(t *testing.T) {
				testMariadbBackupRestoreForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMariadbWithEncryption_RestoreIsSuccessful", func(t *testing.T) {
				testMariadbBackupRestoreWithEncryptionForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMariadb_WithReadOnlyUser_RestoreIsSuccessful", func(t *testing.T) {
				testMariadbBackupRestoreWithReadOnlyUserForVersion(t, endpoint, dbVersion.version)
			})

			if dbVersion.runsExcludeTests {
				t.Run("Test_BackupAndRestoreMariadb_WithExcludeTables_ExcludedTablesNotRestored", func(t *testing.T) {
					testMariadbBackupRestoreWithExcludeTablesForVersion(t, endpoint, dbVersion.version)
				})
				t.Run("Test_BackupAndRestoreMariadb_WithExcludeEvents_EventsNotRestored", func(t *testing.T) {
					testMariadbBackupRestoreWithExcludeEventsForVersion(t, endpoint, dbVersion.version)
				})
			}
		})
	}
}

func testMariadbBackupRestoreForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mariadbVersion tools.MariadbVersion,
) {
	container, err := connectToMariadbEndpoint(t, endpoint, mariadbVersion)
	if err != nil {
		t.Fatalf("MariaDB %s test failed: %v", mariadbVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMariadbTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MariaDB Test Workspace", user, router)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMariadbDatabaseViaAPI(
		t, router, "MariaDB Test Database", workspace.ID,
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

	newDBName := "restoreddb_mariadb"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMariadbDataIntegrity(t, container.DB, newDB)

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

func testMariadbBackupRestoreWithEncryptionForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mariadbVersion tools.MariadbVersion,
) {
	container, err := connectToMariadbEndpoint(t, endpoint, mariadbVersion)
	if err != nil {
		t.Fatalf("MariaDB %s test failed: %v", mariadbVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMariadbTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MariaDB Encrypted Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMariadbDatabaseViaAPI(
		t, router, "MariaDB Encrypted Test Database", workspace.ID,
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

	newDBName := "restoreddb_mariadb_encrypted"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMariadbDataIntegrity(t, container.DB, newDB)

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

func testMariadbBackupRestoreWithReadOnlyUserForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mariadbVersion tools.MariadbVersion,
) {
	container, err := connectToMariadbEndpoint(t, endpoint, mariadbVersion)
	if err != nil {
		t.Fatalf("MariaDB %s test failed: %v", mariadbVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMariadbTestData(t, container.DB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MariaDB ReadOnly Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMariadbDatabaseViaAPI(
		t, router, "MariaDB ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	readOnlyUser := createMariadbReadOnlyUserViaAPI(t, router, database.ID, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateMariadbDatabaseCredentialsViaAPI(
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

	newDBName := "restoreddb_mariadb_readonly"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMariadbDataIntegrity(t, container.DB, newDB)

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

func createMariadbDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	version tools.MariadbVersion,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypeMariadb,
		Mariadb: &mariadbtypes.MariadbDatabase{
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
		t.Fatalf("Failed to create MariaDB database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var createdDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &createdDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &createdDatabase
}

func createMariadbRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	version tools.MariadbVersion,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		MariadbDatabase: &mariadbtypes.MariadbDatabase{
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

func waitForMariadbRestoreCompletion(
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
			t.Fatalf("Timeout waiting for MariaDB restore completion after %v", timeout)
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
				t.Fatalf("MariaDB restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func verifyMariadbDataIntegrity(t *testing.T, originalDB, restoredDB *sqlx.DB) {
	var originalData []MariadbTestDataItem
	var restoredData []MariadbTestDataItem

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

func connectToMariadbContainer(
	t *testing.T,
	image string,
	version tools.MariadbVersion,
) (*MariadbContainer, error) {
	endpoint := containers.StartMariadb(t, image)

	return connectToMariadbEndpoint(t, endpoint, version)
}

func connectToMariadbEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) (*MariadbContainer, error) {
	dbName := containers.MariadbDatabase
	username := "root"
	password := containers.MariadbRootPassword

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, endpoint.Host, endpoint.Port, dbName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MariaDB database: %w", err)
	}

	return &MariadbContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}, nil
}

func setupMariadbTestData(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec(dropMariadbTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(createMariadbTestTableQuery)
	assert.NoError(t, err)

	_, err = db.Exec(insertMariadbTestDataQuery)
	assert.NoError(t, err)
}

func createMariadbReadOnlyUserViaAPI(
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

func updateMariadbDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database *databases.Database,
	username string,
	password string,
	token string,
) *databases.Database {
	database.Mariadb.Username = username
	database.Mariadb.Password = password

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+token,
		database,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update MariaDB database. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var updatedDatabase databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &updatedDatabase); err != nil {
		t.Fatalf("Failed to unmarshal database response: %v", err)
	}

	return &updatedDatabase
}

func testMariadbBackupRestoreWithExcludeTablesForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mariadbVersion tools.MariadbVersion,
) {
	container, err := connectToMariadbEndpoint(t, endpoint, mariadbVersion)
	if err != nil {
		t.Fatalf("MariaDB %s test failed: %v", mariadbVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMariadbTestData(t, container.DB)

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
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MariaDB Exclude Tables Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMariadbDatabaseViaAPI(
		t, router, "MariaDB Exclude Tables Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	database.Mariadb.ExcludeTables = []string{"\nextra_table"}
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

	newDBName := "restoreddb_mariadb_excl_tables"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
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

func testMariadbBackupRestoreWithExcludeEventsForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mariadbVersion tools.MariadbVersion,
) {
	container, err := connectToMariadbEndpoint(t, endpoint, mariadbVersion)
	if err != nil {
		t.Fatalf("MariaDB %s test failed: %v", mariadbVersion, err)
	}
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	setupMariadbTestData(t, container.DB)

	_, err = container.DB.Exec(`
		CREATE EVENT IF NOT EXISTS test_event
		ON SCHEDULE EVERY 1 DAY
		DO BEGIN
			INSERT INTO test_data (name, value) VALUES ('event_test', 999);
		END
	`)
	if err != nil {
		t.Fatalf(
			"MariaDB version doesn't support events or event scheduler disabled: %v",
			err,
		)
	}

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MariaDB Exclude Events Test Workspace",
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)

	database := createMariadbDatabaseViaAPI(
		t, router, "MariaDB Exclude Events Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		user.Token,
	)

	database.Mariadb.IsExcludeEvents = true
	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/update",
		"Bearer "+user.Token,
		database,
	)
	if w.Code != http.StatusOK {
		t.Fatalf(
			"Failed to update database with IsExcludeEvents. Status: %d, Body: %s",
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

	newDBName := "restoreddb_mariadb_no_events"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, newDBName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, newDBName,
		container.Version,
		user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	var tableExists int
	err = newDB.Get(
		&tableExists,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = 'test_data'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, tableExists, "Table 'test_data' should exist in restored database")

	verifyMariadbDataIntegrity(t, container.DB, newDB)

	var eventCount int
	err = newDB.Get(
		&eventCount,
		"SELECT COUNT(*) FROM information_schema.events WHERE event_schema = ? AND event_name = 'test_event'",
		newDBName,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		0,
		eventCount,
		"Event 'test_event' should NOT exist in restored database when IsExcludeEvents is true",
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
