package mariadb_logical

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

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
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

const bastionedMariadbImage = "mariadb:11.4"

func bastionedMariadbConfig(
	topology containers.BastionedDatabase,
	databaseName string,
) *mariadbtypes.MariadbDatabase {
	return &mariadbtypes.MariadbDatabase{
		Host:      topology.Database.Host,
		Port:      topology.Database.Port,
		Username:  "root",
		Password:  containers.MariadbRootPassword,
		Database:  &databaseName,
		Version:   tools.MariadbVersion114,
		SshTunnel: bastion.GetTunnelConfig(topology),
	}
}

func openTestTunnelToMariadb(t *testing.T, topology containers.BastionedDatabase, databaseName string) *sqlx.DB {
	t.Helper()

	localEndpoint := logicaltesting.OpenForwardedEndpoint(t, topology)

	connection, err := sqlx.Connect("mysql", fmt.Sprintf(
		"root:%s@tcp(%s:%d)/%s?parseTime=true",
		containers.MariadbRootPassword, localEndpoint.Host, localEndpoint.Port, databaseName,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	return connection
}

func Test_MariadbBackupRestore_OverSshTunnel_RestoresData(t *testing.T) {
	topology := containers.StartMariadbBehindSshBastion(t, bastionedMariadbImage)
	sourceConnection := openTestTunnelToMariadb(t, topology, containers.MariadbDatabase)

	setupMariadbTestData(t, sourceConnection)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MariaDB SSH Tunnel Workspace", user, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	storage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), storage.ID) })

	database := logicaltesting.SubmitCreateDatabase(t, router, "MariaDB cycle over SSH tunnel",
		databases.Database{
			Name:        "Bastioned MariaDB cycle",
			WorkspaceID: &workspace.ID,
			Type:        databases.DatabaseTypeMariadb,
			Mariadb:     bastionedMariadbConfig(topology, containers.MariadbDatabase),
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

	restoredDatabaseName := "restoreddb_mariadb_ssh_tunnel"
	_, err := sourceConnection.Exec("CREATE DATABASE " + restoredDatabaseName)
	require.NoError(t, err)

	logicaltesting.SubmitRestore(t, router, backup.ID,
		restores_core.RestoreBackupRequest{
			MariadbDatabase: bastionedMariadbConfig(topology, restoredDatabaseName),
		}, user.Token)

	restore := waitForMariadbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMariadbDataIntegrity(t, sourceConnection, openTestTunnelToMariadb(t, topology, restoredDatabaseName))
}
