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
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
)

func bastionedPostgresConfig(
	topology containers.BastionedDatabase,
) *postgresql_logical.PostgresqlLogicalDatabase {
	databaseName := containers.PostgresDatabase

	return &postgresql_logical.PostgresqlLogicalDatabase{
		Host:      topology.Database.Host,
		Port:      topology.Database.Port,
		Username:  containers.PostgresUsername,
		Password:  containers.PostgresPassword,
		Database:  &databaseName,
		CpuCount:  1,
		SshTunnel: bastion.GetTunnelConfig(topology),
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
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Cycle Workspace", user, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	storage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), storage.ID) })

	database := logicaltesting.SubmitCreateDatabase(t, router, "Postgres cycle over SSH tunnel",
		databases.Database{
			Name:              "Bastioned PG cycle",
			WorkspaceID:       &workspace.ID,
			Type:              databases.DatabaseTypePostgresLogical,
			PostgresqlLogical: bastionedPostgresConfig(topology),
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

	restoredDatabaseName := "restoreddb_ssh_tunnel"
	_, err = sourceConnection.Exec("CREATE DATABASE " + restoredDatabaseName)
	require.NoError(t, err)

	restoreTarget := bastionedPostgresConfig(topology)
	restoreTarget.Database = &restoredDatabaseName

	logicaltesting.SubmitRestore(t, router, backup.ID,
		restores_core.RestoreBackupRequest{PostgresqlLogicalDatabase: restoreTarget}, user.Token)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyDataIntegrity(t, sourceConnection, openTestTunnelTo(t, topology, restoredDatabaseName), tableName)
}
