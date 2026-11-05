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
	t.Parallel()
	t.Run(
		"removes unneeded WAL and history when a running full is removed and a target full survives",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			running := catalog.insertFull(t, newInProgressFull(inProgressFullSpec{
				DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
			}))
			oldWal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1, TimelineID: 1})
			oldHistory := catalog.seedHistory(t, 1)

			secondStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
			secondCatalog := *catalog
			secondCatalog.StorageID = secondStorageID
			surviving := secondCatalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4, TimelineID: 2, Offset: 0x28})
			unneededTargetWal := secondCatalog.seedCascadeWal(t, cascadeWalSpec{Segment: 2, TimelineID: 2})
			boundaryWal := secondCatalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4, TimelineID: 2})
			targetHistory := secondCatalog.seedHistory(t, 2)

			response := catalog.saveStorageSelection(t, &secondStorageID)
			require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
			require.NotNil(t, response.JSON200)

			catalog.assertFullGone(t, running.ID)
			catalog.assertFullPresent(t, surviving.ID)
			catalog.assertWalGone(t, oldWal.ID)
			catalog.assertWalGone(t, unneededTargetWal.ID)
			catalog.assertHistoryPresent(t, oldHistory.ID, false)
			catalog.assertWalPresent(t, boundaryWal.ID)
			catalog.assertHistoryPresent(t, targetHistory.ID, true)

			var capturedFiles []string
			catalog.App.QuerySQL(
				t,
				&capturedFiles,
				`SELECT file_name FROM storages_pending_file_deletions WHERE cleanup_database_id=?`,
				catalog.DatabaseID,
			)
			for _, name := range []string{*oldWal.FileName, *unneededTargetWal.FileName, oldHistory.FileName} {
				assert.Contains(t, capturedFiles, name)
				assert.Contains(t, capturedFiles, name+".metadata")
			}
			assert.NotContains(t, capturedFiles, *boundaryWal.FileName)
			assert.NotContains(t, capturedFiles, targetHistory.FileName)
		},
	)

	t.Run("removes the old chain and schedules its files for deletion when storage changes", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)

		catalog.saveStorageConfig(t, catalog.StorageID)

		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalPresent(t, incremental.ID)
		catalog.assertWalPresent(t, wal.ID)
		assert.Empty(t, catalog.pendingFiles(t, catalog.StorageID))
	})
	t.Run("keeps the config and chain unchanged when recording cleanup fails", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)
		targetStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)

		catalog.App.ExecuteSQL(
			t,
			`CREATE FUNCTION refuse_switch_cleanup() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'cleanup refused'; END $$ LANGUAGE plpgsql`,
		)
		catalog.App.ExecuteSQL(
			t,
			`CREATE TRIGGER refuse_switch_cleanup BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION refuse_switch_cleanup()`,
		)

		backupConfig := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
		response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
			t.Context(),
			catalog.DatabaseID,
			apiclient.BackupConfigInput{
				IsBackupsEnabled: true,
				StorageId:        &targetStorageID,
				Encryption:       backupConfig.Encryption,
				Settings:         backupConfig.Settings,
			},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, response.StatusCode(), "%s", response.Body)

		assert.Equal(t, &catalog.StorageID, apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID).StorageId)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalPresent(t, incremental.ID)
		catalog.assertWalPresent(t, wal.ID)
		assert.Empty(t, catalog.pendingFiles(t, catalog.StorageID))
	})
}

func (catalog *catalogFixture) saveStorageConfig(t *testing.T, storageID uuid.UUID) {
	t.Helper()
	response := catalog.saveStorageSelection(t, &storageID)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

func (catalog *catalogFixture) saveStorageSelection(
	t *testing.T,
	storageID *uuid.UUID,
) *apiclient.UpdateBackupConfigResponse {
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
			StorageId:        storageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	return response
}

func TestReassignUnfinishedPhysicalFiles(t *testing.T) {
	t.Parallel()
	for _, isTransfer := range []bool{false, true} {
		scenario := "captures only the last selected storage after clearing and restarting"
		if isTransfer {
			scenario = "captures only the last selected storage after clearing and restarting before transfer"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)
			secondStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
			catalog.saveStorageConfig(t, secondStorageID)

			oldFile := catalog.DatabaseID.String() + "-late-first-storage"
			unfinishedFile := catalog.DatabaseID.String() + "-unfinished-second-storage"
			catalog.App.WriteUnpublishedFile(t, catalog.StorageID, oldFile)
			catalog.App.WriteUnpublishedFile(t, secondStorageID, unfinishedFile)

			cleared := catalog.saveStorageSelection(t, nil)
			require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
			catalog.App.RestartModules(t)
			assert.Nil(t, apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID).StorageId)

			var operation apiclient.BackupConfig
			if isTransfer {
				workspace := apptest.CreateWorkspace(t, catalog.Owner, "Reassigned unfinished physical files")
				targetStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, workspace.Id)
				operation = apptest.TransferToStorage(t, catalog.Owner, apptest.StorageSwitchTransfer{
					DatabaseID: catalog.DatabaseID, WorkspaceID: workspace.Id, StorageID: targetStorageID,
				})
			} else {
				targetStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
				assigned := catalog.saveStorageSelection(t, &targetStorageID)
				require.Equal(t, http.StatusOK, assigned.StatusCode(), "%s", assigned.Body)
				require.NotNil(t, assigned.JSON200)
				operation = *assigned.JSON200
			}

			assert.EqualValues(t, 1, operation.Cleanup.PendingFileCount)

			var capturedFiles []struct {
				StorageID uuid.UUID
				FileName  string
			}
			catalog.App.QuerySQL(
				t,
				&capturedFiles,
				`SELECT storage_id,file_name FROM storages_pending_file_deletions WHERE cleanup_database_id=?`,
				catalog.DatabaseID,
			)
			require.Len(t, capturedFiles, 1)
			assert.Equal(t, secondStorageID, capturedFiles[0].StorageID)
			assert.Equal(t, unfinishedFile, capturedFiles[0].FileName)
			assert.Contains(t, catalog.pendingFiles(t, catalog.StorageID), oldFile)
		})
	}
}

func (catalog *catalogFixture) pendingFiles(t *testing.T, storageID uuid.UUID) []string {
	t.Helper()
	var names []string
	catalog.App.QuerySQL(
		t,
		&names,
		"SELECT file_name FROM storages_pending_file_deletions WHERE storage_id = ?",
		storageID,
	)
	return names
}

func TestRollbackReassignedPhysicalStorage(t *testing.T) {
	t.Parallel()
	t.Run("preserves all retained history when reassignment cleanup cannot be saved", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full, incremental, wal := seedOneOfEachType(t, catalog)

		backupConfig := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
		cleared, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
			t.Context(),
			catalog.DatabaseID,
			apiclient.BackupConfigInput{
				IsBackupsEnabled: true,
				Encryption:       backupConfig.Encryption,
				Settings:         backupConfig.Settings,
			},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)

		catalog.App.RefuseStorageFileDeletions(t)
		targetStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		assigned, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
			t.Context(),
			catalog.DatabaseID,
			apiclient.BackupConfigInput{
				IsBackupsEnabled: true,
				StorageId:        &targetStorageID,
				Encryption:       backupConfig.Encryption,
				Settings:         backupConfig.Settings,
			},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, assigned.StatusCode(), "%s", assigned.Body)

		assert.Nil(t, apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID).StorageId)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalPresent(t, incremental.ID)
		catalog.assertWalPresent(t, wal.ID)
		assert.Empty(t, catalog.pendingFiles(t, catalog.StorageID))
	})
}
