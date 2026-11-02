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
func TestDeleteLogicalBackup(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("removes the backup from the list and deletes its file and metadata file", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		keptBackup := completeRestorableBackup(t, tm.owner, database.databaseID)
		deletedFileName := getBackupFileName(t, tm, database.backup.Id)
		keptFileName := getBackupFileName(t, tm, keptBackup.Id)
		require.True(t, tm.app.IsLocalStorageFilePresent(t, deletedFileName))
		require.True(t, tm.app.IsLocalStorageFilePresent(t, deletedFileName+".metadata"))

		response := requestLogicalBackupDeletion(t, tm.owner, database.backup.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{keptBackup.Id}, listLogicalBackupIDs(t, tm.owner, database.databaseID))

		readResponse := requestLogicalBackup(t, tm.owner, database.backup.Id)
		requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, deletedFileName))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, deletedFileName+".metadata"))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, keptFileName))
	})

	t.Run("returns 409 already_running for a running backup and keeps it running", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)

		response := requestLogicalBackupDeletion(t, tm.owner, runningBackup.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, getLogicalBackupStatus(t, tm.owner, runningBackup.Id))
	})

	t.Run("returns 404 not_found for a backup that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestLogicalBackupDeletion(t, tm.owner, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}
