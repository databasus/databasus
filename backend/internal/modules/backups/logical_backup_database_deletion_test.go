package backups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestDeleteDatabaseWithLogicalBackups(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("removes the backups, their restore and token and their files with the database", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		secondBackup := completeRestorableBackup(t, tm.owner, database.databaseID)
		restoredBackupRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, restoredBackupRestore.Id, apiclient.RestoreStatusCOMPLETED)
		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)
		backupFileNames := []string{
			getBackupFileName(t, tm, database.backup.Id),
			getBackupFileName(t, tm, secondBackup.Id),
		}

		deleteResponse, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), database.databaseID)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleteResponse.StatusCode(), string(deleteResponse.Body))

		for _, backupID := range []apiclient.BackupId{database.backup.Id, secondBackup.Id} {
			readResponse := requestLogicalBackup(t, tm.owner, backupID)
			requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
		}

		restoreResponse, err := tm.owner.Client.GetRestoreWithResponse(t.Context(), restoredBackupRestore.Id)
		require.NoError(t, err)
		requireProblem(t, restoreResponse.StatusCode(), restoreResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)

		downloadResponse := requestBackupDownload(t, tm, database.backup.Id, issuedToken.Token)
		requireProblem(t, downloadResponse.StatusCode(), downloadResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)

		tm.app.RunJobPass(t, storageFileDeletionJobName)

		for _, backupFileName := range backupFileNames {
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, backupFileName))
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, backupFileName+".metadata"))
		}
	})
}
