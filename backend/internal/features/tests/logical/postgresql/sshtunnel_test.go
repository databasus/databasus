package postgresql_logical

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

func bastionedPostgresSettings(topology containers.BastionedDatabase) apiclient.PostgresqlLogicalSettings {
	settings := newPostgresqlSettings(
		topology.Database.Host,
		topology.Database.Port,
		containers.PostgresUsername,
		containers.PostgresPassword,
		containers.PostgresDatabase,
	)
	settings.SshTunnel = logicaltesting.NewSshTunnelSettings(bastion.GetTunnelSettings(topology))

	return settings
}

func bastionedPostgresTarget(
	topology containers.BastionedDatabase,
	databaseName string,
) *restores_core.PostgresqlTargetDatabase {
	return &restores_core.PostgresqlTargetDatabase{
		Host:      topology.Database.Host,
		Port:      topology.Database.Port,
		Username:  containers.PostgresUsername,
		Password:  containers.PostgresPassword,
		Database:  &databaseName,
		CpuCount:  1,
		SshTunnel: logicaltesting.NewTargetSshTunnel(bastion.GetTunnelSettings(topology)),
	}
}

func openTestTunnelTo(t *testing.T, topology containers.BastionedDatabase, databaseName string) *sqlx.DB {
	t.Helper()

	localEndpoint := logicaltesting.OpenForwardedEndpoint(t, topology)

	connection, err := sqlx.Connect("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		localEndpoint.Host, localEndpoint.Port,
		containers.PostgresUsername, containers.PostgresPassword, databaseName,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	return connection
}

func Test_PostgresqlBackupRestore_OverSshTunnel_RestoresData(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")
	sourceConnection := openTestTunnelTo(t, topology, containers.PostgresDatabase)

	tableName := "test_data_ssh_tunnel"
	_, err := sourceConnection.Exec(createAndFillTableQuery(tableName))
	require.NoError(t, err)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Cycle Workspace", user)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace) })

	storage := storages_bridge.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages_bridge.RemoveTestStorage(t.Context(), storage.ID) })

	database := logicaltesting.SubmitCreateDatabase(t, router, "Postgres cycle over SSH tunnel",
		logicaltesting.NewDatabaseCreate(t, workspace.ID, bastionedPostgresSettings(topology)),
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

	restoredDatabaseName := "restoreddb_ssh_tunnel"
	_, err = sourceConnection.Exec("CREATE DATABASE " + restoredDatabaseName)
	require.NoError(t, err)

	logicaltesting.SubmitRestore(t, router, backup.ID,
		restores_core.RestoreBackupRequest{
			PostgresqlLogicalDatabase: bastionedPostgresTarget(topology, restoredDatabaseName),
		}, user.Token)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyDataIntegrity(t, sourceConnection, openTestTunnelTo(t, topology, restoredDatabaseName), tableName)
}
