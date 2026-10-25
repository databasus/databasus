package mongodb_logical

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

const bastionedMongodbImage = "mongo:7.0"

func bastionedMongodbConfig(
	topology containers.BastionedDatabase,
	databaseName string,
) *mongodbtypes.MongodbDatabase {
	return &mongodbtypes.MongodbDatabase{
		Host:         topology.Database.Host,
		Port:         new(topology.Database.Port),
		Username:     containers.MongodbUsername,
		Password:     containers.MongodbPassword,
		Database:     databaseName,
		AuthDatabase: containers.MongodbAuthDatabase,
		CpuCount:     1,
		Version:      tools.MongodbVersion7,
		SshTunnel:    bastion.GetTunnelConfig(topology),
	}
}

// The forwarded endpoint stands in for the container's own address, so the shared connect helper
// works unchanged.
func openTestTunnelToMongodb(t *testing.T, topology containers.BastionedDatabase) *MongodbContainer {
	t.Helper()

	localEndpoint := logicaltesting.OpenForwardedEndpoint(t, topology)

	return connectToMongodbEndpoint(t, containers.Endpoint{
		Host: localEndpoint.Host,
		Port: localEndpoint.Port,
	}, tools.MongodbVersion7)
}

func Test_MongodbBackupRestore_OverSshTunnel_RestoresData(t *testing.T) {
	topology := containers.StartMongodbBehindSshBastion(t, bastionedMongodbImage)
	sourceContainer := openTestTunnelToMongodb(t, topology)

	setupMongodbTestData(t, sourceContainer)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MongoDB SSH Tunnel Workspace", user, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	storage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), storage.ID) })

	database := logicaltesting.SubmitCreateDatabase(t, router, "MongoDB cycle over SSH tunnel",
		databases.Database{
			Name:        "Bastioned MongoDB cycle",
			WorkspaceID: &workspace.ID,
			Type:        databases.DatabaseTypeMongodb,
			Mongodb:     bastionedMongodbConfig(topology, containers.MongodbDatabase),
		}, user.Token)

	t.Cleanup(func() {
		test_utils.MakeDeleteRequest(t, router, "/api/v1/databases/"+database.ID.String(),
			"Bearer "+user.Token, http.StatusNoContent)
	})

	logicaltesting.EnableBackupsViaAPI(t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token)
	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	require.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	})

	restoredDatabaseName := "restoreddb_mongodb_ssh_tunnel"

	logicaltesting.SubmitRestore(t, router, backup.ID,
		restores_core.RestoreBackupRequest{
			MongodbDatabase: bastionedMongodbConfig(topology, restoredDatabaseName),
		}, user.Token)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, sourceContainer, restoredDatabaseName)
}
