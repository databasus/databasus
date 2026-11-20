package instancebackups_test

import (
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

const storageFileDeletionJob = "storage_file_deletion"

//nolint:tparallel // Each scenario moves its own application's clock between backups.
func TestApplyInstanceBackupRetention(t *testing.T) {
	t.Parallel()

	t.Run("deletes expired copies but keeps the newest stored copy of a destination", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectRetainedDestinations(t, 1, storageID)
		oldestBackup := owner.backUp(t)
		oldestFileName := owner.getStoredFileName(t, oldestBackup, storageID)
		owner.App.Clock.Advance(time.Hour)
		newestBackup := owner.backUp(t)
		newestFileName := owner.getStoredFileName(t, newestBackup, storageID)

		owner.App.Clock.Advance(3 * 24 * time.Hour)
		owner.App.RunJobPass(t, retentionJob)
		owner.App.RunJobPass(t, storageFileDeletionJob)

		assert.Equal(t, apiclient.InstanceBackupCopyStatusDELETED,
			getCopy(t, owner.getBackup(t, oldestBackup.Id), storageID).Status)
		assert.False(t, owner.App.IsLocalStorageFilePresent(t, oldestFileName))
		keptCopy := getCopy(t, owner.getBackup(t, newestBackup.Id), storageID)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, keptCopy.Status)
		assert.True(t, keptCopy.IsDownloadable)
		assert.True(t, owner.App.IsLocalStorageFilePresent(t, newestFileName))
	})

	t.Run("keeps the last stored copy while newer deliveries fail", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		owner.selectRetainedDestinations(t, 1, namespace.StorageID)
		storedBackup := owner.backUp(t)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.App.Clock.Advance(3 * 24 * time.Hour)
		failingBackup := owner.backUp(t)

		owner.App.RunJobPass(t, retentionJob)

		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusRETRYING,
			getCopy(t, failingBackup, namespace.StorageID).Status,
		)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED,
			getCopy(t, owner.getBackup(t, storedBackup.Id), namespace.StorageID).Status)
	})

	t.Run("leaves database backup files and copies in deselected storages untouched", func(t *testing.T) {
		owner := newInstanceOwner(t)
		selectedStorageID := owner.createLocalStorage(t, "Selected")
		deselectedStorageID := owner.createLocalStorage(t, "Deselected")
		databaseBackupFileName := "database-backup-" + uuid.NewString() + ".dump"
		owner.App.PublishStorageFile(t, storages.StoredFileReference{
			StorageID: selectedStorageID, FileName: databaseBackupFileName,
		}, strings.NewReader("a database backup"))
		owner.selectRetainedDestinations(t, 1, selectedStorageID, deselectedStorageID)
		firstBackup := owner.backUp(t)
		owner.App.Clock.Advance(time.Hour)
		owner.backUp(t)
		owner.selectRetainedDestinations(t, 1, selectedStorageID)

		owner.App.Clock.Advance(3 * 24 * time.Hour)
		owner.App.RunJobPass(t, retentionJob)
		owner.App.RunJobPass(t, storageFileDeletionJob)

		retainedBackup := owner.getBackup(t, firstBackup.Id)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusDELETED, getCopy(t, retainedBackup, selectedStorageID).Status)
		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusSTORED,
			getCopy(t, retainedBackup, deselectedStorageID).Status,
		)
		assert.True(t, owner.App.IsLocalStorageFilePresent(t, databaseBackupFileName))
	})
}

func (o instanceOwner) selectRetainedDestinations(t *testing.T, retentionDays int, storageIDs ...uuid.UUID) {
	t.Helper()

	settings := o.getSettings(t)
	settings.RetentionDays = retentionDays
	settings.StorageIds = append([]uuid.UUID{}, storageIDs...)

	response, err := o.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
}

func (o instanceOwner) getStoredFileName(t *testing.T, backup apiclient.InstanceBackup, storageID uuid.UUID) string {
	t.Helper()

	var storedFileName string
	o.App.QuerySQL(
		t,
		&storedFileName,
		`SELECT file_name FROM instancebackups_copies WHERE id = ?`,
		getCopy(t, backup, storageID).Id,
	)

	return storedFileName
}
