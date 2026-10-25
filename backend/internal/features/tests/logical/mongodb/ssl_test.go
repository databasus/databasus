package mongodb_logical

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
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

func Test_BackupAndRestoreMongodbSSL_Succeeds(t *testing.T) {
	endpoint := containers.StartMongodbSSL(t, "testdata/ssl/server.pem", "testdata/ssl/server.crt")
	host := endpoint.Host
	portInt := endpoint.Port

	uri := fmt.Sprintf(
		"mongodb://root:rootpassword@%s:%d/testdb?authSource=admin&tls=true&tlsInsecure=true&serverSelectionTimeoutMS=5000",
		host,
		portInt,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("failed to connect to SSL MongoDB: %v", err)
	}
	defer client.Disconnect(t.Context())

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping SSL MongoDB: %v", err)
	}

	container := &MongodbContainer{
		Host:         host,
		Port:         portInt,
		Username:     "root",
		Password:     "rootpassword",
		Database:     "testdb",
		AuthDatabase: "admin",
		Version:      tools.MongodbVersion8,
		Client:       client,
	}
	setupMongodbTestData(t, container)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MongoDB SSL Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	database := createMongodbSSLDatabaseViaAPI(
		t, router, "MongoDB SSL DB", workspace.ID,
		host, portInt, "root", "rootpassword",
		"testdb", "admin", tools.MongodbVersion8, user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)
	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mongo_ssl_" + uuid.New().String()[:8]
	createMongodbSSLRestoreViaAPI(
		t, router, backup.ID,
		host, portInt, "root", "rootpassword",
		newDBName, "admin", tools.MongodbVersion8, user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	_ = container.Client.Database(newDBName).Drop(t.Context())
	_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token, http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createMongodbSSLDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username, password, database, authDatabase string,
	version tools.MongodbVersion,
	token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypeMongodb,
		Mongodb: &mongodbtypes.MongodbDatabase{
			Host:         host,
			Port:         &port,
			Username:     username,
			Password:     password,
			Database:     database,
			AuthDatabase: authDatabase,
			Version:      version,
			IsHttps:      true,
			IsSrv:        false,
			CpuCount:     1,
		},
	}

	return logicaltesting.SubmitCreateDatabase(t, router, "MongoDB SSL", request, token)
}

func createMongodbSSLRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username, password, database, authDatabase string,
	version tools.MongodbVersion,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		MongodbDatabase: &mongodbtypes.MongodbDatabase{
			Host:         host,
			Port:         &port,
			Username:     username,
			Password:     password,
			Database:     database,
			AuthDatabase: authDatabase,
			Version:      version,
			IsHttps:      true,
			IsSrv:        false,
			CpuCount:     1,
		},
	}
	logicaltesting.SubmitRestore(t, router, backupID, request, token)
}
