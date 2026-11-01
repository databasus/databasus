package storages_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/modules/storages"
)

func TestStorageDeletionBatch(t *testing.T) {
	t.Parallel()
	t.Run("reports exhausted cleanup files when their source workspace is deleted", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		source := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
		reference := storages.StoredFileReference{StorageID: source.StorageID, FileName: "exhausted.dump"}
		tm.app.PublishStorageFile(t, reference, strings.NewReader("remaining"))
		batch := tm.app.RequestStorageDeletionBatch(t, []storages.StoredFileReference{reference})

		source.MakeUnavailable(t, tm.owner)
		for range 5 {
			tm.app.RunJobPass(t, "storage_file_deletion")
			tm.app.Clock.Advance(time.Second)
		}

		terminal := tm.app.GetStorageDeletionBatch(t, batch.ID)
		require.Equal(t, storages.CleanupWithRemainders, terminal.Status)
		require.EqualValues(t, 1, terminal.Remaining)

		response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)

		assert.Equal(t, terminal, tm.app.GetStorageDeletionBatch(t, batch.ID))
		assert.Contains(
			t,
			strings.Join(listAuditMessages(t, tm, tm.workspace.Id), "\n"),
			"deleted with 1 file(s) left behind: exhausted.dump",
		)
	})

	t.Run("retains confirmed deletion after restarting modules", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		storage := tm.createLocalStorage(t, "Old storage")
		reference := storages.StoredFileReference{StorageID: storage.Id, FileName: "old.dump"}
		tm.app.PublishStorageFile(t, reference, strings.NewReader("old backup"))
		batch := tm.app.RequestStorageDeletionBatch(t, []storages.StoredFileReference{reference})

		tm.app.RestartModules(t)
		tm.app.RunJobPass(t, "storage_file_deletion")
		completed := tm.app.GetStorageDeletionBatch(t, batch.ID)
		assert.Equal(t, storages.CleanupCompleted, completed.Status)
		assert.EqualValues(t, 1, completed.Deleted)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, reference.FileName))

		tm.app.RestartModules(t)
		assert.Equal(t, completed, tm.app.GetStorageDeletionBatch(t, batch.ID))
	})

	for _, isWorkspaceDeletion := range []bool{false, true} {
		scenario := "settles pending artifacts when the storage is deleted"
		if isWorkspaceDeletion {
			scenario = "settles pending artifacts when the workspace is deleted"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			source := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
			deletedFile := storages.StoredFileReference{StorageID: source.StorageID, FileName: "deleted.dump"}
			remainingFile := storages.StoredFileReference{StorageID: source.StorageID, FileName: "remaining.dump"}

			tm.app.PublishStorageFile(t, deletedFile, strings.NewReader("confirmed"))
			tm.app.PublishStorageFile(t, remainingFile, strings.NewReader("remaining"))
			batch := tm.app.RequestStorageDeletionBatch(t, []storages.StoredFileReference{deletedFile, remainingFile})
			tm.app.ExecuteSQL(
				t,
				"UPDATE storages_pending_deletions SET not_before=not_before+interval '1 hour' WHERE file_name=?",
				remainingFile.FileName,
			)

			tm.app.RunJobPass(t, "storage_file_deletion")
			assert.EqualValues(t, 1, tm.app.GetStorageDeletionBatch(t, batch.ID).Deleted)

			source.MakeUnavailable(t, tm.owner)
			if isWorkspaceDeletion {
				response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)
				require.NoError(t, err)
				require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
			} else {
				response := deleteStorage(t, tm.owner, source.StorageID)
				require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
			}

			settled := tm.app.GetStorageDeletionBatch(t, batch.ID)
			assert.Equal(t, storages.CleanupWithRemainders, settled.Status)
			assert.EqualValues(t, 1, settled.Deleted)
			assert.EqualValues(t, 1, settled.Remaining)
			assert.Zero(t, settled.Pending)
			assert.NotNil(t, settled.CompletedAt)
		})
	}
}
