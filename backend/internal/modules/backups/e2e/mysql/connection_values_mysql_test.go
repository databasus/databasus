package mysql

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one MySQL server.
func TestMysqlConnectionValues(t *testing.T) {
	t.Parallel()

	server := enginetest.ConnectToMysqlFamily(
		t,
		containers.StartMysql(t, "mysql:8.0"),
		containers.MysqlRootPassword,
		"",
	)
	server.CreateTestDataTable(t)

	t.Run("fails the backup naming the database name when a stored one starts with a dash", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, newMysqlSettings(server, apiclient.MysqlVersionN80))
		tm.EnableBackups(t, database.Id, tm.StorageID, apiclient.BackupEncryptionNONE)
		resultPath := filepath.Join(t.TempDir(), "out.sql")
		tm.App.ExecuteSQL(t, `UPDATE databases_mysql SET "database" = ? WHERE database_id = ?`,
			"--result-file="+resultPath, database.Id)

		failedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, database.Id).Id)

		assert.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status)
		require.NotNil(t, failedBackup.FailMessage)
		assert.Contains(t, *failedBackup.FailMessage, `database must not start with "-"`)
		assert.NoFileExists(t, resultPath)
	})

	t.Run("returns 400 naming the target's database name to a viewer when it starts with a dash", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, newMysqlSettings(server, apiclient.MysqlVersionN80)).Id,
			apiclient.BackupEncryptionNONE)
		viewer := tm.App.SignUp(t, apptest.NewTestEmail("viewer"))
		apptest.AddWorkspaceMember(t, tm.Owner, tm.Workspace.Id, viewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		teePath := filepath.Join(t.TempDir(), "restore.log")

		response, err := viewer.Client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
			BackupId: backup.Id,
			Target:   toMysqlRestoreTarget(t, newMysqlTarget(server, "--tee="+teePath)),
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, response.StatusCode(), string(response.Body))
		assert.Contains(t, string(response.Body), string(apiclient.ProblemCodeInvalidRequest))
		assert.Contains(t, string(response.Body), `"target.database"`)
		assert.Zero(t, tm.CountRestores(t, backup.Id))
		assert.NoFileExists(t, teePath)
	})

	t.Run("restores the rows into a target whose database name has a dash inside it", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, newMysqlSettings(server, apiclient.MysqlVersionN80)).Id,
			apiclient.BackupEncryptionNONE)
		restoredDatabaseName := "orders-archive-" + enginetest.NewUniqueSuffix()
		server.Execute(t, "CREATE DATABASE `"+restoredDatabaseName+"`")
		server.CleanUp(t, "DROP DATABASE IF EXISTS `"+restoredDatabaseName+"`")

		restore := tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, newMysqlTarget(server, restoredDatabaseName)))

		assert.Equal(t, apiclient.RestoreStatusCOMPLETED, restore.Status)
		enginetest.VerifyMysqlFamilyRows(t, server.Admin, server.Open(t, restoredDatabaseName))
	})
}
