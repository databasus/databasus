package backups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/storages"
)

func TestBackupCleanupSummary(t *testing.T) {
	t.Parallel()

	t.Run(
		"returns computed pending and failed counts when configuration input supplies fabricated counts",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageA, storageB := tm.createLocalStorage(t), tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageA)
			insertLogicalBackupInStorage(t, tm, databaseID, storageA, "COMPLETED")
			accepted := acceptLogicalStorageSwitch(
				t,
				tm,
				logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageB},
			)
			require.EqualValues(t, 2, accepted.Cleanup.PendingFileCount)
			tm.app.ExecuteSQL(
				t,
				"UPDATE storages_pending_file_deletions SET last_error='provider unavailable', not_before=not_before+interval '48 hours' WHERE cleanup_database_id=?",
				databaseID,
			)

			for _, supplied := range []int64{0, 999999} {
				body := newLogicalBackupConfigBody(t, &storageB)
				body["cleanup"] = map[string]any{"pendingFileCount": supplied, "failedFileCount": supplied}
				response := updateBackupConfigWithBody(t, tm.owner, databaseID, body)
				require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
				require.NotNil(t, response.JSON200.Cleanup)
				assert.EqualValues(t, 2, response.JSON200.Cleanup.PendingFileCount)
				assert.EqualValues(t, 2, response.JSON200.Cleanup.FailedFileCount)
				read := apptest.GetBackupConfig(t, tm.owner, databaseID)
				assert.Equal(t, response.JSON200.Cleanup, read.Cleanup)
			}

			listed, err := tm.owner.Client.ListBackupConfigsWithResponse(
				t.Context(),
				&apiclient.ListBackupConfigsParams{WorkspaceId: &tm.workspace.Id},
			)
			require.NoError(t, err)
			require.NotNil(t, listed.JSON200)
			require.Len(t, listed.JSON200.Items, 1)
			assert.EqualValues(t, 2, listed.JSON200.Items[0].Cleanup.PendingFileCount)
			assert.EqualValues(t, 2, listed.JSON200.Items[0].Cleanup.FailedFileCount)
		},
	)

	t.Run("returns zero without saving a default configuration or counting ordinary uploads", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		tm.app.WriteUnpublishedFile(t, storageID, databaseID.String()+"-upload")
		read := apptest.GetBackupConfig(t, tm.owner, databaseID)
		require.NotNil(t, read.Cleanup)
		assert.Zero(t, read.Cleanup.PendingFileCount)
		assert.Zero(t, read.Cleanup.FailedFileCount)
		assert.Zero(t, countRows(t, tm, "SELECT count(*) FROM backups_configs WHERE database_id=?", databaseID))
	})
}

func TestCleanupSummaryStorageAccess(t *testing.T) {
	t.Parallel()

	t.Run("refuses counts when storage access does not grant access to the database workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		outsider := tm.app.CreateNonAdminUser(t, tm.admin)
		privateWorkspace := apptest.CreateWorkspace(t, outsider, "Private database")
		tm.app.ExecuteSQL(t, "UPDATE databases SET workspace_id=? WHERE id=?", privateWorkspace.Id, databaseID)
		tm.app.RequestStorageFileDeletions(t, storages.FileDeletionRequest{
			References: []storages.StoredFileReference{{StorageID: storageID, FileName: "private.dump"}},
			Scope:      &storages.CleanupScope{DatabaseID: databaseID, WorkspaceID: privateWorkspace.Id},
		})

		listed, err := tm.owner.Client.ListBackupConfigsWithResponse(
			t.Context(),
			&apiclient.ListBackupConfigsParams{StorageId: &storageID},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, listed.StatusCode(), "%s", listed.Body)
		adminListed, err := tm.admin.Client.ListBackupConfigsWithResponse(
			t.Context(),
			&apiclient.ListBackupConfigsParams{StorageId: &storageID},
		)
		require.NoError(t, err)
		require.NotNil(t, adminListed.JSON200)
		require.Len(t, adminListed.JSON200.Items, 1)
		assert.EqualValues(t, 1, adminListed.JSON200.Items[0].Cleanup.PendingFileCount)
	})
}
