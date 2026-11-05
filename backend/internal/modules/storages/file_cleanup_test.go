package storages_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/storages"
)

func TestStorageFileCleanup(t *testing.T) {
	t.Parallel()

	t.Run("keeps retrying beyond five failures and a day after a restart", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		source := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
		reference := storages.StoredFileReference{StorageID: source.StorageID, FileName: "outstanding.dump"}
		tm.app.PublishStorageFile(t, reference, strings.NewReader("remaining"))
		scope := &storages.CleanupScope{DatabaseID: uuid.New(), WorkspaceID: tm.workspace.Id}
		tm.app.RequestStorageFileDeletions(
			t,
			storages.FileDeletionRequest{
				References: []storages.StoredFileReference{reference, reference},
				Scope:      scope,
			},
		)
		source.MakeUnavailable(t, tm.owner)

		for range 7 {
			tm.app.RunJobPass(t, "storage_file_deletion")
			tm.app.Clock.Advance(25 * time.Hour)
		}

		tm.app.RestartModules(t)
		tm.app.RunJobPass(t, "storage_file_deletion")

		assert.Equal(t, 1, countPendingDeletions(t, tm.app, source.StorageID))
		var attempts int
		tm.app.QuerySQL(
			t,
			&attempts,
			"SELECT attempt_count FROM storages_pending_file_deletions WHERE storage_id=?",
			source.StorageID,
		)
		assert.Equal(t, 8, attempts)

		response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeCleanupPending,
		)
		assert.Equal(t, 1, countPendingDeletions(t, tm.app, source.StorageID))

		removed, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), source.StorageID,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, removed.StatusCode(), "%s", removed.Body)

		response, err = tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
	})

	t.Run(
		"retains original attribution and removes confirmed files when deletion requests are duplicated",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			storage := tm.createLocalStorage(t, "Old storage")
			reference := storages.StoredFileReference{StorageID: storage.Id, FileName: "old.dump"}
			tm.app.PublishStorageFile(t, reference, strings.NewReader("old backup"))
			scope := &storages.CleanupScope{DatabaseID: uuid.New(), WorkspaceID: tm.workspace.Id}
			tm.app.RequestStorageFileDeletions(
				t,
				storages.FileDeletionRequest{References: []storages.StoredFileReference{reference}, Scope: scope},
			)
			tm.app.RequestStorageFileDeletions(
				t,
				storages.FileDeletionRequest{
					References: []storages.StoredFileReference{reference},
					Scope:      &storages.CleanupScope{DatabaseID: uuid.New(), WorkspaceID: uuid.New()},
				},
			)
			var scopeIDs []uuid.UUID
			tm.app.QuerySQL(
				t,
				&scopeIDs,
				"SELECT cleanup_workspace_id FROM storages_pending_file_deletions WHERE storage_id=?",
				storage.Id,
			)
			assert.Equal(t, []uuid.UUID{tm.workspace.Id}, scopeIDs)

			tm.app.RestartModules(t)
			tm.app.RunJobPass(t, "storage_file_deletion")

			assert.Zero(t, countPendingDeletions(t, tm.app, storage.Id))
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, reference.FileName))
		},
	)
}
