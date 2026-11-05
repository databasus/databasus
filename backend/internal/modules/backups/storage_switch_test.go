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

type logicalStorageSwitchSpec struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
}

func acceptLogicalStorageSwitch(t *testing.T, tm team, selection logicalStorageSwitchSpec) apiclient.BackupConfig {
	t.Helper()
	response := updateBackupConfig(t, tm.owner, selection.DatabaseID,
		newLogicalBackupConfigInput(t, &selection.StorageID, newLogicalBackupSettings()))
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	require.NotNil(t, response.JSON200)
	assert.Empty(t, response.HTTPResponse.Header.Get("Location"))
	return *response.JSON200
}

func TestCurrentStorageCleanup(t *testing.T) {
	t.Parallel()

	t.Run("retains aggregate cleanup when a later storage change adds no files", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageA, storageB, storageC := tm.createLocalStorage(t), tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageA)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, storageA, "COMPLETED")

		first := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageB},
		)
		require.Positive(t, first.Cleanup.PendingFileCount)
		tm.app.RestartModules(t)
		second := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageC},
		)
		assert.Equal(t, first.Cleanup, second.Cleanup)
		assert.Equal(t, second.Cleanup, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup)

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.Zero(t, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup.PendingFileCount)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
	})

	t.Run(
		"deletes old files from both original storages when two changes overlap with files published in the selected storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageA := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
			storageB := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
			storageC := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageA.StorageID)
			backupA := insertLogicalBackupInStorage(t, tm, databaseID, storageA.StorageID, "COMPLETED")
			first := acceptLogicalStorageSwitch(
				t,
				tm,
				logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageB.StorageID},
			)
			require.EqualValues(t, 2, first.Cleanup.PendingFileCount)
			backupB := insertLogicalBackupInStorage(t, tm, databaseID, storageB.StorageID, "COMPLETED")

			second := acceptLogicalStorageSwitch(
				t,
				tm,
				logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageC.StorageID},
			)
			backupC := insertLogicalBackupInStorage(t, tm, databaseID, storageC.StorageID, "COMPLETED")

			require.EqualValues(t, 4, second.Cleanup.PendingFileCount)
			assert.Equal(t, second.Cleanup, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup)
			storageA.RequireFile(t, backupA.fileName)
			storageB.RequireFile(t, backupB.fileName)
			storageC.RequireFile(t, backupC.fileName)

			tm.app.RunJobPass(t, storageFileDeletionJobName)

			storageA.RequireFileAbsent(t, backupA.fileName)
			storageB.RequireFileAbsent(t, backupB.fileName)
			storageC.RequireFile(t, backupC.fileName)
			assert.Zero(t, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup.PendingFileCount)
			listed, err := tm.owner.Client.ListLogicalBackupsWithResponse(
				t.Context(),
				&apiclient.ListLogicalBackupsParams{DatabaseId: databaseID},
			)
			require.NoError(t, err)
			require.NotNil(t, listed.JSON200)
			require.Len(t, listed.JSON200.Items, 1)
			assert.Equal(t, backupC.id, listed.JSON200.Items[0].Id)
		},
	)

	t.Run("hides earlier workspace cleanup after another transfer", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		oldStorage, newStorage := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorage)
		insertLogicalBackupInStorage(t, tm, databaseID, oldStorage, "COMPLETED")
		accepted := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: newStorage},
		)
		require.Positive(t, accepted.Cleanup.PendingFileCount)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		assert.Equal(t, accepted.Cleanup, apptest.GetBackupConfig(t, viewer, databaseID).Cleanup)
		assert.Equal(t, accepted.Cleanup, apptest.GetBackupConfig(t, tm.admin, databaseID).Cleanup)
		outsider := tm.app.CreateNonAdminUser(t, tm.admin)
		for _, client := range []*apiclient.ClientWithResponses{outsider.Client, tm.app.NewClient(t)} {
			response, err := client.GetBackupConfigWithResponse(t.Context(), databaseID)
			require.NoError(t, err)
			if client == outsider.Client {
				assert.Equal(t, http.StatusNotFound, response.StatusCode())
			} else {
				assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
			}
		}

		target := tm.createWorkspace(t)
		response := requestDatabaseTransfer(t, tm.owner, databaseID,
			newDatabaseTransfer(t, target.Id, newStorageMovedWithDatabase(t)))
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
		assert.Zero(t, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup.PendingFileCount)
		assert.Positive(
			t,
			countRows(
				t,
				tm,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
				databaseID,
			),
		)
	})
}

func TestTransferDatabaseRacingTargetStorageTransfer(t *testing.T) {
	t.Parallel()

	t.Run("commits only one move when the target storage moves concurrently", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		for range 4 {
			databaseID := tm.insertLogicalDatabase(t)
			sourceStorageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, sourceStorageID)
			oldBackup := insertLogicalBackupInStorage(t, tm, databaseID, sourceStorageID, "COMPLETED")
			targetWorkspace, otherWorkspace := tm.createWorkspace(t), tm.createWorkspace(t)
			targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)

			type moveOutcome struct {
				Status int
				Err    error
			}
			switched, moved := make(chan moveOutcome, 1), make(chan moveOutcome, 1)
			start := make(chan struct{})
			transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID))

			go func() {
				<-start
				response, err := tm.owner.Client.TransferDatabaseWithResponse(
					t.Context(),
					databaseID,
					transfer,
				)
				if err != nil {
					switched <- moveOutcome{Err: err}
					return
				}
				switched <- moveOutcome{Status: response.StatusCode()}
			}()
			go func() {
				<-start
				response, err := tm.owner.Client.TransferStorageWithResponse(
					t.Context(),
					targetStorageID,
					apiclient.StorageTransfer{TargetWorkspaceId: otherWorkspace.Id},
				)
				if err != nil {
					moved <- moveOutcome{Err: err}
					return
				}
				moved <- moveOutcome{Status: response.StatusCode()}
			}()

			close(start)
			switchOutcome, storageMove := <-switched, <-moved
			require.NoError(t, switchOutcome.Err)
			require.NoError(t, storageMove.Err)

			if switchOutcome.Status == http.StatusNoContent {
				assert.Equal(t, http.StatusConflict, storageMove.Status)
				requireConfigStorage(t, tm, databaseID, targetStorageID)
				assert.Zero(t, countLogicalBackups(t, tm, databaseID))
			} else {
				assert.Equal(t, http.StatusBadRequest, switchOutcome.Status)
				assert.Equal(t, http.StatusNoContent, storageMove.Status)
				requireConfigStorage(t, tm, databaseID, sourceStorageID)
				assert.Equal(t, 1, countLogicalBackups(t, tm, databaseID))
				assert.True(t, tm.app.IsLocalStorageFilePresent(t, oldBackup.fileName))
			}
		}
	})
}

func TestStorageCleanupLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("keeps a claimed file pending until the provider confirms deletion", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		source := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
		targetID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, source.StorageID)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, source.StorageID, "COMPLETED")
		accepted := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetID},
		)
		require.Positive(t, accepted.Cleanup.PendingFileCount)
		deletion := source.HoldNextDeletion(t, tm.owner, false)
		completed := make(chan struct{})
		go func() {
			defer close(completed)
			tm.app.RunJobPass(t, storageFileDeletionJobName)
		}()
		deletion.WaitUntilStarted(t)

		duringClaim := apptest.GetBackupConfig(t, tm.owner, databaseID)

		assert.Equal(t, accepted.Cleanup, duringClaim.Cleanup)
		source.RequireFile(t, backup.fileName)
		assert.Positive(
			t,
			countRows(
				t,
				tm,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND attempt_count>0",
				source.StorageID,
			),
		)
		deletion.Release()
		<-completed
		source.RequireFileAbsent(t, backup.fileName)
		assert.Zero(t, apptest.GetBackupConfig(t, tm.owner, databaseID).Cleanup.PendingFileCount)
	})

	t.Run("omits abandoned work from configuration reads after deleting unused source storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		sourceID, targetID := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, sourceID)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, sourceID, "COMPLETED")
		accepted := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetID},
		)
		require.Positive(t, accepted.Cleanup.PendingFileCount)

		deleted, err := tm.owner.Client.DeleteStorageWithResponse(
			t.Context(),
			sourceID,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode(), "%s", deleted.Body)
		currentConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)
		assert.Equal(t, &targetID, currentConfig.StorageId)
		assert.Zero(t, currentConfig.Cleanup.PendingFileCount)
		assert.Zero(t, currentConfig.Cleanup.FailedFileCount)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
	})

	t.Run("removes old files after the database is deleted during scoped cleanup", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		sourceID, targetID := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, sourceID)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, sourceID, "COMPLETED")
		accepted := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetID},
		)
		require.Positive(t, accepted.Cleanup.PendingFileCount)

		deleted, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode(), "%s", deleted.Body)
		assert.EqualValues(t, accepted.Cleanup.PendingFileCount, countRows(
			t,
			tm,
			"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND cleanup_workspace_id=?",
			databaseID,
			tm.workspace.Id,
		))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.Zero(
			t,
			countRows(
				t,
				tm,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
				databaseID,
			),
		)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
	})
}
