package mariadb_logical

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

func Test_BackupAndRestoreMariadbSSL_Succeeds(t *testing.T) {
	logicaltesting.RegisterSSLMysqlTLSConfig(t)

	endpoint := containers.StartMariadbSSL(t, "mariadb:11.8")
	host := endpoint.Host
	portInt := endpoint.Port

	dsn := fmt.Sprintf("root:rootpassword@tcp(%s:%d)/testdb?parseTime=true&tls=%s",
		host, portInt, logicaltesting.SSLMysqlTLSConfigName)
	originalDB, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to connect to SSL MariaDB: %v", err)
	}
	defer originalDB.Close()

	setupMariadbTestData(t, originalDB)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MariaDB SSL Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	dbName := "testdb"
	database := createMariadbSSLDatabaseViaAPI(
		t, router, "MariaDB SSL DB", workspace.ID,
		host, portInt, "root", "rootpassword", dbName,
		tools.MariadbVersion118, user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)
	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mariadb_ssl"
	_, err = originalDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)
	_, err = originalDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf("root:rootpassword@tcp(%s:%d)/%s?parseTime=true&tls=%s",
		host, portInt, newDBName, logicaltesting.SSLMysqlTLSConfigName)
	newDB, err := sqlx.Connect("mysql", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createMariadbSSLRestoreViaAPI(
		t, router, backup.ID,
		host, portInt, "root", "rootpassword", newDBName,
		tools.MariadbVersion118, user.Token,
	)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMariadbDataIntegrity(t, originalDB, newDB)

	_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token, http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createMariadbSSLDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username, password, database string,
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
			IsHttps:  true,
		},
	}

	return logicaltesting.SubmitCreateDatabase(t, router, "MariaDB SSL", request, token)
}

func createMariadbSSLRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username, password, database string,
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
			IsHttps:  true,
		},
	}
	logicaltesting.SubmitRestore(t, router, backupID, request, token)
}
