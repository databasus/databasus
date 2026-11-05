package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestBackupConfigOperationsWithoutSession(t *testing.T) {
	t.Parallel()

	t.Run("returns 401 unauthenticated for a list without a session", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.app.NewClient(t).ListBackupConfigsWithResponse(t.Context(),
			&apiclient.ListBackupConfigsParams{
				WorkspaceId: &tm.workspace.Id,
				StorageId:   nil,
				Limit:       nil,
				After:       nil,
			})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 401 unauthenticated for a read without a session", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)

		response, err := tm.app.NewClient(t).GetBackupConfigWithResponse(t.Context(), databaseID)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 401 unauthenticated for a save without a session and saves nothing", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)

		response, err := tm.app.NewClient(t).UpdateBackupConfigWithResponse(t.Context(), databaseID,
			newLogicalBackupConfigInput(t, &storageID, newLogicalBackupSettings()))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
	})

	t.Run("returns 401 unauthenticated for a transfer without a session and moves nothing", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

		response, err := tm.app.NewClient(t).TransferDatabaseWithResponse(t.Context(), databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
	})
}
