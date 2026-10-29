package catalog_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestChangePhysicalBackupStorage(t *testing.T) {
	t.Run("removes the old chain and schedules its files for deletion when storage changes", func(t *testing.T) {
		catalog := newCatalog(t)
		newStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)
		catalog.saveStorageConfig(t, newStorageID)
		assert.Empty(t, getBackups(t, catalog, "").Backups)
		pending := catalog.pendingFiles(t, catalog.StorageID)
		for _, name := range []string{*full.FileName, *incremental.FileName, *wal.FileName} {
			assert.Contains(t, pending, name)
		}
	})
	t.Run("releases and removes a running full when storage changes", func(t *testing.T) {
		catalog := newCatalog(t)
		newStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.saveStorageConfig(t, newStorageID)
		assert.Nil(t, catalog.getRunningBackupID(t))
		catalog.assertFullGone(t, full.ID)
		assert.Empty(t, getBackups(t, catalog, "").Backups)
	})
	t.Run("keeps the chain when storage stays the same", func(t *testing.T) {
		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalPresent(t, incremental.ID)
		catalog.assertWalPresent(t, wal.ID)
		assert.Empty(t, catalog.pendingFiles(t, catalog.StorageID))
	})
	t.Run("removes the old chain on the next save after its first removal failed", func(t *testing.T) {
		catalog := newCatalog(t)
		newStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		catalog.saveStorageConfig(t, newStorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)
		catalog.saveStorageConfig(t, newStorageID)
		catalog.assertFullGone(t, full.ID)
		catalog.assertIncrementalGone(t, incremental.ID)
		catalog.assertWalGone(t, wal.ID)
	})
}

func (catalog *catalogFixture) saveStorageConfig(t *testing.T, storageID uuid.UUID) {
	t.Helper()
	config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	configureRetentionSchedule(&settings)
	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))
	response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		catalog.DatabaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &storageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

func (catalog *catalogFixture) pendingFiles(t *testing.T, storageID uuid.UUID) []string {
	t.Helper()
	var names []string
	catalog.App.QuerySQL(t, &names, "SELECT file_name FROM storages_pending_deletions WHERE storage_id = ?", storageID)
	return names
}
