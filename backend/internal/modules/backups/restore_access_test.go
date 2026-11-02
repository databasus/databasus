package backups_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestLogicalBackupAndRestoreAccess(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("lets a viewer back up, get a download token and restore", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		viewerBackup := completeRestorableBackup(t, viewer, database.databaseID)

		tokenResponse, err := viewer.Client.CreateDownloadTokenWithResponse(t.Context(),
			apiclient.DownloadTokenCreate{BackupId: viewerBackup.Id})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, tokenResponse.StatusCode(), string(tokenResponse.Body))

		viewerRestore := requireStartedRestore(t, viewer, viewerBackup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, viewer, viewerRestore.Id, apiclient.RestoreStatusCOMPLETED)
	})

	t.Run("returns 404 not_found for every read of another team's backups", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))

		listResponse, err := stranger.Client.ListLogicalBackupsWithResponse(t.Context(),
			&apiclient.ListLogicalBackupsParams{
				DatabaseId:    database.databaseID,
				Status:        nil,
				CreatedBefore: nil,
				Limit:         nil,
				After:         nil,
			})
		require.NoError(t, err)
		requireProblem(
			t,
			listResponse.StatusCode(),
			listResponse.Body,
			http.StatusNotFound,
			apiclient.ProblemCodeNotFound,
		)

		readResponse, err := stranger.Client.GetLogicalBackupWithResponse(t.Context(), database.backup.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			readResponse.StatusCode(),
			readResponse.Body,
			http.StatusNotFound,
			apiclient.ProblemCodeNotFound,
		)

		tokenResponse, err := stranger.Client.CreateDownloadTokenWithResponse(t.Context(),
			apiclient.DownloadTokenCreate{BackupId: database.backup.Id})
		require.NoError(t, err)
		requireProblem(t, tokenResponse.StatusCode(), tokenResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)

		restoresResponse, err := stranger.Client.ListRestoresWithResponse(t.Context(),
			&apiclient.ListRestoresParams{BackupId: database.backup.Id, Limit: nil, After: nil})
		require.NoError(t, err)
		requireProblem(t, restoresResponse.StatusCode(), restoresResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
	})

	t.Run("returns the same 404 for an existing and a missing backup and restore to another team", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		existingRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, existingRestore.Id, apiclient.RestoreStatusCOMPLETED)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
		missingID := uuid.New()

		existingBackupResponse, err := stranger.Client.GetLogicalBackupWithResponse(t.Context(), database.backup.Id)
		require.NoError(t, err)
		missingBackupResponse, err := stranger.Client.GetLogicalBackupWithResponse(t.Context(), missingID)
		require.NoError(t, err)
		existingRestoreResponse, err := stranger.Client.GetRestoreWithResponse(t.Context(), existingRestore.Id)
		require.NoError(t, err)
		missingRestoreResponse, err := stranger.Client.GetRestoreWithResponse(t.Context(), missingID)
		require.NoError(t, err)

		for _, response := range []struct {
			status int
			body   []byte
		}{
			{status: existingBackupResponse.StatusCode(), body: existingBackupResponse.Body},
			{status: missingBackupResponse.StatusCode(), body: missingBackupResponse.Body},
			{status: existingRestoreResponse.StatusCode(), body: existingRestoreResponse.Body},
			{status: missingRestoreResponse.StatusCode(), body: missingRestoreResponse.Body},
		} {
			requireProblem(t, response.status, response.body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		}

		assert.Equal(t,
			strings.ReplaceAll(string(missingBackupResponse.Body), missingID.String(), ""),
			strings.ReplaceAll(string(existingBackupResponse.Body), database.backup.Id.String(), ""))
		assert.Equal(t,
			strings.ReplaceAll(string(missingRestoreResponse.Body), missingID.String(), ""),
			strings.ReplaceAll(string(existingRestoreResponse.Body), existingRestore.Id.String(), ""))
	})

	t.Run("returns 404 not_found and changes nothing when a user of another team changes backups and restores",
		func(t *testing.T) {
			tm := newTeam(t)
			work := startRunningWork(t, tm, server)
			stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))

			startBackupResponse, err := stranger.Client.StartLogicalBackupWithResponse(t.Context(),
				apiclient.LogicalBackupStart{DatabaseId: work.database.databaseID})
			require.NoError(t, err)
			requireProblem(t, startBackupResponse.StatusCode(), startBackupResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			cancelBackupResponse, err := stranger.Client.CancelLogicalBackupWithResponse(t.Context(),
				work.runningBackup.Id)
			require.NoError(t, err)
			requireProblem(t, cancelBackupResponse.StatusCode(), cancelBackupResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			deleteBackupResponse, err := stranger.Client.DeleteLogicalBackupWithResponse(t.Context(),
				work.database.backup.Id)
			require.NoError(t, err)
			requireProblem(t, deleteBackupResponse.StatusCode(), deleteBackupResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			startRestoreResponse := startRestore(t, stranger, work.database.backup.Id,
				server.newRestoreTarget(t, server.newTargetSpec(t)))
			requireProblem(t, startRestoreResponse.StatusCode(), startRestoreResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			cancelRestoreResponse := cancelRestore(t, stranger, work.runningRestore.Id)
			requireProblem(t, cancelRestoreResponse.StatusCode(), cancelRestoreResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			assert.Equal(t, 2, countRows(t, tm, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?",
				work.database.databaseID))
			assert.Equal(t, 1, countRestoresOfBackup(t, tm, work.database.backup.Id))
			requireWorkStillRunning(t, tm, work)
		})

	t.Run("returns 403 permission_denied and changes nothing when a viewer cancels and deletes", func(t *testing.T) {
		tm := newTeam(t)
		work := startRunningWork(t, tm, server)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		cancelBackupResponse, err := viewer.Client.CancelLogicalBackupWithResponse(t.Context(), work.runningBackup.Id)
		require.NoError(t, err)
		requireProblem(t, cancelBackupResponse.StatusCode(), cancelBackupResponse.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)

		deleteBackupResponse, err := viewer.Client.DeleteLogicalBackupWithResponse(t.Context(), work.database.backup.Id)
		require.NoError(t, err)
		requireProblem(t, deleteBackupResponse.StatusCode(), deleteBackupResponse.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)

		cancelRestoreResponse := cancelRestore(t, viewer, work.runningRestore.Id)
		requireProblem(t, cancelRestoreResponse.StatusCode(), cancelRestoreResponse.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)

		requireWorkStillRunning(t, tm, work)
	})
}

// requireWorkStillRunning checks that the running backup and restore still run and the completed
// backup is still listed.
func requireWorkStillRunning(t *testing.T, tm team, work runningWork) {
	t.Helper()

	assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS,
		getLogicalBackupStatus(t, tm.owner, work.runningBackup.Id))
	assert.Equal(t, apiclient.RestoreStatusINPROGRESS, getRestore(t, tm.owner, work.runningRestore.Id).Status)
	assert.Contains(t, listLogicalBackupIDs(t, tm.owner, work.database.databaseID), work.database.backup.Id)
}
