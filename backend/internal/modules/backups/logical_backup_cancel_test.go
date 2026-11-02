package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestCancelLogicalBackup(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 202, records the running backup as canceled and deletes its file", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)
		cancelledFileName := getBackupFileName(t, tm, runningBackup.Id)

		response := requestLogicalBackupCancel(t, tm.owner, runningBackup.Id)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		assert.Equal(t, runningBackup.Id, response.JSON202.Id)
		waitForRestorableBackupStatus(t, tm.owner, runningBackup.Id, apiclient.LogicalBackupStatusCANCELED)
		assert.Zero(t, countRunningLogicalBackups(t, tm.owner, source.databaseID))

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, cancelledFileName))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, cancelledFileName+".metadata"))
		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM storages_pending_deletions WHERE file_name IN (?, ?)",
			cancelledFileName, cancelledFileName+".metadata"))
	})

	t.Run("returns 409 not_running when the backup has completed and keeps it completed", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		response := requestLogicalBackupCancel(t, tm.owner, database.backup.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeNotRunning)
		assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, getLogicalBackupStatus(t, tm.owner, database.backup.Id))
	})

	t.Run("returns 404 not_found for a backup that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestLogicalBackupCancel(t, tm.owner, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}
