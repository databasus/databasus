package mysql_logical

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
	mysql_engine "databasus-backend/internal/engines/mysql"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	storages_bridge "databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
)

const bastionedMysqlImage = "mysql:8.4"

func bastionedMysqlSettings(topology containers.BastionedDatabase, databaseName string) apiclient.MysqlSettings {
	password := containers.MysqlRootPassword
	version := apiclient.MysqlVersionN84

	return apiclient.MysqlSettings{
		Host:          topology.Database.Host,
		Port:          topology.Database.Port,
		Username:      "root",
		Password:      &password,
		Database:      &databaseName,
		Version:       &version,
		ExcludeTables: []string{},
		SshTunnel:     logicaltesting.NewSshTunnelSettings(bastion.GetTunnelSettings(topology)),
	}
}

func bastionedMysqlTarget(
	topology containers.BastionedDatabase,
	databaseName string,
) *restores_core.MysqlTargetDatabase {
	return &restores_core.MysqlTargetDatabase{
		Host:      topology.Database.Host,
		Port:      topology.Database.Port,
		Username:  "root",
		Password:  containers.MysqlRootPassword,
		Database:  &databaseName,
		Version:   mysql_engine.Version84,
		SshTunnel: logicaltesting.NewTargetSshTunnel(bastion.GetTunnelSettings(topology)),
	}
}

func openTestTunnelToMysql(t *testing.T, topology containers.BastionedDatabase, databaseName string) *sqlx.DB {
	t.Helper()

	localEndpoint := logicaltesting.OpenForwardedEndpoint(t, topology)

	connection, err := sqlx.Connect("mysql", fmt.Sprintf(
		"root:%s@tcp(%s:%d)/%s?parseTime=true",
		containers.MysqlRootPassword, localEndpoint.Host, localEndpoint.Port, databaseName,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	return connection
}

func Test_MysqlBackupRestore_OverSshTunnel_RestoresData(t *testing.T) {
	topology := containers.StartMysqlBehindSshBastion(t, bastionedMysqlImage)
	sourceConnection := openTestTunnelToMysql(t, topology, containers.MysqlDatabase)

	setupMysqlTestData(t, sourceConnection)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MySQL SSH Tunnel Workspace", user)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace) })

	storage := storages_bridge.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages_bridge.RemoveTestStorage(t.Context(), storage.ID) })

	database := logicaltesting.SubmitCreateDatabase(t, router, "MySQL cycle over SSH tunnel",
		logicaltesting.NewDatabaseCreate(t, workspace.ID, bastionedMysqlSettings(topology, containers.MysqlDatabase)),
		user.Token)

	t.Cleanup(func() {
		test_utils.MakeDeleteRequest(t, router, "/api/v1/databases/"+database.Id.String(),
			"Bearer "+user.Token, http.StatusNoContent)
	})

	logicaltesting.EnableBackupsViaAPI(t, router, database.Id, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token)
	logicaltesting.CreateBackupViaAPI(t, router, database.Id, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.Id, user.Token, 5*time.Minute)
	require.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	})

	restoredDatabaseName := "restoreddb_mysql_ssh_tunnel"
	_, err := sourceConnection.Exec("CREATE DATABASE " + restoredDatabaseName)
	require.NoError(t, err)

	logicaltesting.SubmitRestore(t, router, backup.ID,
		restores_core.RestoreBackupRequest{
			MysqlDatabase: bastionedMysqlTarget(topology, restoredDatabaseName),
		}, user.Token)

	restore := waitForMysqlRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMysqlDataIntegrity(t, sourceConnection, openTestTunnelToMysql(t, topology, restoredDatabaseName))
}
