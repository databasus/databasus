package postgresql_logical

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
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

func Test_BackupAndRestorePostgresqlSSL_Succeeds(t *testing.T) {
	contextDir, err := filepath.Abs(filepath.Join("testdata", "ssl"))
	if err != nil {
		t.Fatalf("failed to resolve ssl testdata dir: %v", err)
	}

	endpoint := containers.StartPostgresSSL(t, contextDir)
	host := endpoint.Host
	portInt := endpoint.Port

	dsn := fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=testdb sslmode=require",
		host, portInt,
	)
	originalDB, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to SSL Postgres: %v", err)
	}
	defer originalDB.Close()

	tableName := "test_data_ssl"
	_, err = originalDB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Postgres SSL Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	dbName := "testdb"
	database := createPostgresqlSSLDatabaseViaAPI(
		t, router, "Postgres SSL DB", workspace.ID,
		host, portInt, "testuser", "testpassword", dbName, user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)
	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_pg_ssl"
	_, err = originalDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)
	_, err = originalDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDSN := fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=%s sslmode=require",
		host, portInt, newDBName,
	)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	createPostgresqlSSLRestoreViaAPI(
		t, router, backup.ID,
		host, portInt, "testuser", "testpassword", newDBName, user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyDataIntegrity(t, originalDB, newDB, tableName)

	_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token, http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createPostgresqlSSLDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username, password, database, token string,
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
			SslMode:  postgresql_shared.PostgresSslModeRequire,
			CpuCount: 1,
		},
	}

	return logicaltesting.SubmitCreateDatabase(t, router, "Postgres SSL", request, token)
}

func createPostgresqlSSLRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username, password, database, token string,
) {
	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &pgtypes.PostgresqlLogicalDatabase{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: &database,
			SslMode:  postgresql_shared.PostgresSslModeRequire,
			CpuCount: 1,
		},
	}
	logicaltesting.SubmitRestore(t, router, backupID, request, token)
}
