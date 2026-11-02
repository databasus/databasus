package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestLogicalBackupAccess(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("lets a member cancel a running backup and delete a completed one", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		source := newBackupSourceDatabase(t, tm, server)
		completedBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)

		cancelResponse := requestLogicalBackupCancel(t, member, runningBackup.Id)
		deleteResponse := requestLogicalBackupDeletion(t, member, completedBackup.Id)

		require.Equal(t, http.StatusAccepted, cancelResponse.StatusCode(), string(cancelResponse.Body))
		require.Equal(t, http.StatusNoContent, deleteResponse.StatusCode(), string(deleteResponse.Body))
		waitForRestorableBackupStatus(t, member, runningBackup.Id, apiclient.LogicalBackupStatusCANCELED)
		assert.Equal(t, []uuid.UUID{runningBackup.Id}, listLogicalBackupIDs(t, member, source.databaseID))
	})

	t.Run("lets a viewer start a backup, list and read backups and get a download token", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		source := newBackupSourceDatabase(t, tm, server)

		startResponse := requestLogicalBackupStart(t, viewer, source.databaseID)

		require.Equal(t, http.StatusAccepted, startResponse.StatusCode(), string(startResponse.Body))
		completedBackup := waitForRestorableBackupStatus(t, viewer, startResponse.JSON202.Id,
			apiclient.LogicalBackupStatusCOMPLETED)
		assert.Equal(t, []uuid.UUID{completedBackup.Id}, listLogicalBackupIDs(t, viewer, source.databaseID))
		issueDownloadToken(t, viewer, completedBackup.Id)
	})

	t.Run("refuses a viewer who cancels a running backup or deletes a completed one with 403", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		source := newBackupSourceDatabase(t, tm, server)
		completedBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)

		cancelResponse := requestLogicalBackupCancel(t, viewer, runningBackup.Id)
		deleteResponse := requestLogicalBackupDeletion(t, viewer, completedBackup.Id)

		requireProblem(t, cancelResponse.StatusCode(), cancelResponse.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, deleteResponse.StatusCode(), deleteResponse.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, getLogicalBackupStatus(t, tm.owner, runningBackup.Id))
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, source.databaseID), completedBackup.Id)
	})

	t.Run("answers 404 not_found to a user outside the workspace and changes nothing", func(t *testing.T) {
		tm := newTeam(t)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
		source := newBackupSourceDatabase(t, tm, server)
		completedBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)

		listResponse := requestLogicalBackupPage(t, stranger, newLogicalBackupListParams(source.databaseID))
		readResponse := requestLogicalBackup(t, stranger, completedBackup.Id)
		tokenResponse := requestDownloadToken(t, stranger, completedBackup.Id)
		startResponse := requestLogicalBackupStart(t, stranger, source.databaseID)
		cancelResponse := requestLogicalBackupCancel(t, stranger, runningBackup.Id)
		deleteResponse := requestLogicalBackupDeletion(t, stranger, completedBackup.Id)

		for _, response := range []struct {
			status int
			body   []byte
		}{
			{listResponse.StatusCode(), listResponse.Body},
			{readResponse.StatusCode(), readResponse.Body},
			{tokenResponse.StatusCode(), tokenResponse.Body},
			{startResponse.StatusCode(), startResponse.Body},
			{cancelResponse.StatusCode(), cancelResponse.Body},
			{deleteResponse.StatusCode(), deleteResponse.Body},
		} {
			requireProblem(t, response.status, response.body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		}

		assert.ElementsMatch(t, []uuid.UUID{runningBackup.Id, completedBackup.Id},
			listLogicalBackupIDs(t, tm.owner, source.databaseID))
		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, getLogicalBackupStatus(t, tm.owner, runningBackup.Id))
	})

	t.Run("answers a user outside the workspace the same as for a backup that does not exist", func(t *testing.T) {
		tm := newTeam(t)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
		database := newBackedUpDatabase(t, tm, server)

		existingResponse := requestLogicalBackup(t, stranger, database.backup.Id)
		missingResponse := requestLogicalBackup(t, stranger, uuid.New())

		requireProblem(t, existingResponse.StatusCode(), existingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(missingResponse.Body), string(existingResponse.Body))
	})

	t.Run("answers 401 unauthenticated to every backup operation without a session", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		anonymousClient := tm.app.NewClient(t)

		listResponse, err := anonymousClient.ListLogicalBackupsWithResponse(t.Context(),
			new(newLogicalBackupListParams(database.databaseID)))
		require.NoError(t, err)
		readResponse, err := anonymousClient.GetLogicalBackupWithResponse(t.Context(), database.backup.Id)
		require.NoError(t, err)
		cancelResponse, err := anonymousClient.CancelLogicalBackupWithResponse(t.Context(), database.backup.Id)
		require.NoError(t, err)
		deleteResponse, err := anonymousClient.DeleteLogicalBackupWithResponse(t.Context(), database.backup.Id)
		require.NoError(t, err)

		for _, response := range []struct {
			status int
			body   []byte
		}{
			{listResponse.StatusCode(), listResponse.Body},
			{readResponse.StatusCode(), readResponse.Body},
			{cancelResponse.StatusCode(), cancelResponse.Body},
			{deleteResponse.StatusCode(), deleteResponse.Body},
		} {
			requireProblem(t, response.status, response.body, http.StatusUnauthorized,
				apiclient.ProblemCodeUnauthenticated)
		}

		assert.Equal(t, []uuid.UUID{database.backup.Id}, listLogicalBackupIDs(t, tm.owner, database.databaseID))
	})
}
