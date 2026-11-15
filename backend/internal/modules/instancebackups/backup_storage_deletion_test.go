package instancebackups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestDeleteStorageWithInstanceBackupCopies(t *testing.T) {
	t.Parallel()

	t.Run("deletes a deselected storage and keeps its copies in history as unavailable", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)
		storedCopy := getCopy(t, backup, storageID)
		storedFileName := owner.getStoredFileName(t, backup, storageID)
		owner.selectDestinations(t)

		deletion := deleteStorage(t, owner.RootAdmin, storageID, false)
		owner.App.RunJobPass(t, storageFileDeletionJob)
		detachedBackup := owner.getBackup(t, backup.Id)
		tokenResponse, err := owner.RootAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(),
			backup.Id, apiclient.InstanceBackupDownloadTokenCreate{CopyId: storedCopy.Id})
		require.NoError(t, err)

		assert.Equal(t, http.StatusNoContent, deletion.StatusCode(), string(deletion.Body))
		require.Len(t, detachedBackup.Copies, 1)
		detachedCopy := detachedBackup.Copies[0]
		assert.Nil(t, detachedCopy.StorageId)
		assert.Equal(t, storedCopy.StorageName, detachedCopy.StorageName)
		assert.Equal(t, storedCopy.WorkspaceName, detachedCopy.WorkspaceName)
		assert.False(t, detachedCopy.IsDownloadable)
		requireProblemCode(t, tokenResponse.StatusCode(), tokenResponse.Body, http.StatusConflict,
			apiclient.ProblemCodeStorageUnavailable)
		assert.True(t, owner.App.IsLocalStorageFilePresent(t, storedFileName))
		assert.Zero(t, countPendingFileDeletions(t, owner, storedFileName))
	})

	t.Run("deletes a workspace whose deselected storage holds copies", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		workspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Archive")
		storageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)
		owner.selectDestinations(t)

		deletion := deleteWorkspace(t, owner.RootAdmin, workspace.Id)

		assert.Equal(t, http.StatusNoContent, deletion.StatusCode(), string(deletion.Body))
		detachedBackup := owner.getBackup(t, backup.Id)
		require.Len(t, detachedBackup.Copies, 1)
		assert.Nil(t, detachedBackup.Copies[0].StorageId)
	})

	t.Run("refuses deleting a storage that is still a destination and keeps its copies", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)

		deletion := deleteStorage(t, owner.RootAdmin, storageID, true)

		requireProblemCode(t, deletion.StatusCode(), deletion.Body, http.StatusConflict,
			apiclient.ProblemCodeDatabasusBackupInUse)
		keptCopy := getCopy(t, owner.getBackup(t, backup.Id), storageID)
		assert.True(t, keptCopy.IsDownloadable)
	})
}
