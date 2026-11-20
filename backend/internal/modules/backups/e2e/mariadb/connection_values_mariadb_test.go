package mariadb

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one MariaDB server.
func TestMariadbConnectionValues(t *testing.T) {
	t.Parallel()

	server := enginetest.ConnectToMysqlFamily(
		t, containers.StartMariadb(t, "mariadb:11.8"), containers.MariadbRootPassword, "",
	)
	server.CreateTestDataTable(t)

	t.Run("fails the backup naming the user name when a stored one holds a line break", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, newMariadbSettings(server, apiclient.MariadbVersionN118))
		tm.EnableBackups(t, database.Id, tm.StorageID, apiclient.BackupEncryptionNONE)
		resultPath := filepath.Join(t.TempDir(), "out.sql")
		tm.App.ExecuteSQL(t, "UPDATE databases_mariadb SET username = ? WHERE database_id = ?",
			"x\nresult-file="+resultPath, database.Id)

		failedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, database.Id).Id)

		assert.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status)
		require.NotNil(t, failedBackup.FailMessage)
		assert.Contains(t, *failedBackup.FailMessage, "username must not contain a line break")
		assert.NoFileExists(t, resultPath)
	})

	t.Run("returns 400 naming the target's password before connecting when it holds a line break", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, newMariadbSettings(server, apiclient.MariadbVersionN118)).Id,
			apiclient.BackupEncryptionNONE)
		restoredDatabaseName, _ := server.CreateRestoreDatabase(t, "restored_mariadb_password")
		target := newMariadbTarget(server, restoredDatabaseName)
		target.Password = new(server.RootPassword + "\nresult-file=/tmp/out.sql")

		response, err := tm.Owner.Client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
			BackupId: backup.Id,
			Target:   toMariadbRestoreTarget(t, target),
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, response.StatusCode(), string(response.Body))
		assert.Contains(t, string(response.Body), string(apiclient.ProblemCodeInvalidRequest))
		assert.Contains(t, string(response.Body), `"target.password"`)
		assert.Zero(t, tm.CountRestores(t, backup.Id))
	})
}
