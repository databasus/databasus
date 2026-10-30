package backups_test

import (
	"net/http"
	"sort"
	"testing"
	"time"

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

func acceptLogicalStorageSwitch(t *testing.T, tm team, selection logicalStorageSwitchSpec) apiclient.StorageSwitch {
	t.Helper()
	response := updateBackupConfig(
		t,
		tm.owner,
		selection.DatabaseID,
		newLogicalBackupConfigInput(t, &selection.StorageID, newLogicalBackupSettings()),
	)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), "%s", response.Body)
	require.NotNil(t, response.JSON202)
	assert.Equal(
		t,
		"/api/v1/storage-switches/"+response.JSON202.Id.String(),
		response.HTTPResponse.Header.Get("Location"),
	)
	return *response.JSON202
}

func getStorageSwitch(t *testing.T, session apptest.Session, operationID uuid.UUID) apiclient.StorageSwitch {
	t.Helper()
	response, err := session.Client.GetStorageSwitchWithResponse(t.Context(), operationID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	require.NotNil(t, response.JSON200)
	operation := *response.JSON200
	assert.Equal(t, operation.Total, operation.Pending+operation.Deleted+operation.Remaining)
	if operation.Status == apiclient.StorageSwitchStatusCLEANUPPENDING {
		assert.Nil(t, operation.CompletedAt)
		assert.Positive(t, operation.Pending)
	} else {
		assert.Zero(t, operation.Pending)
		require.NotNil(t, operation.CompletedAt)
		assert.False(t, operation.CompletedAt.Before(operation.CreatedAt))
	}
	return operation
}

func TestReadStorageSwitch(t *testing.T) {
	t.Parallel()
	t.Run(
		"shows worker outcomes directly and preserves history after restart and resource deletion",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			oldStorageID, targetStorageID := tm.createLocalStorage(t), tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorageID)
			backup := insertLogicalBackupInStorage(t, tm, databaseID, oldStorageID, "COMPLETED")
			operation := acceptLogicalStorageSwitch(
				t,
				tm,
				logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetStorageID},
			)
			assert.Equal(t, apiclient.StorageSwitchStatusCLEANUPPENDING, operation.Status)
			assert.Equal(t, &oldStorageID, operation.PreviousStorageId)
			require.Positive(t, operation.Total)
			otherDatabaseID := tm.insertLogicalDatabase(t)
			saveLogicalBackupConfig(t, tm.owner, otherDatabaseID, oldStorageID)
			otherOperation := acceptLogicalStorageSwitch(t, tm, logicalStorageSwitchSpec{
				DatabaseID: otherDatabaseID, StorageID: targetStorageID,
			})
			tm.app.RestartModules(t)
			assert.Equal(t, operation.Total, getStorageSwitch(t, tm.owner, operation.Id).Pending)
			deleted, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, deleted.StatusCode(), "%s", deleted.Body)
			pending := getStorageSwitch(t, tm.owner, operation.Id)
			assert.Equal(t, apiclient.StorageSwitchStatusCLEANUPPENDING, pending.Status)
			pendingHistory, err := tm.owner.Client.ListStorageSwitchesWithResponse(t.Context(),
				&apiclient.ListStorageSwitchesParams{WorkspaceId: tm.workspace.Id, DatabaseId: &databaseID})
			require.NoError(t, err)
			require.NotNil(t, pendingHistory.JSON200)
			require.Len(t, pendingHistory.JSON200.Items, 1)
			assert.Equal(t, pending, pendingHistory.JSON200.Items[0])
			assert.NotEqual(t, otherOperation.Id, pendingHistory.JSON200.Items[0].Id)
			tm.app.RunJobPass(t, storageFileDeletionJobName)
			completed := getStorageSwitch(t, tm.owner, operation.Id)
			assert.Equal(t, apiclient.StorageSwitchStatusCOMPLETED, completed.Status)
			assert.Equal(t, operation.Total, completed.Deleted)
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
			removed, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), oldStorageID)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, removed.StatusCode(), "%s", removed.Body)
			tm.app.RestartModules(t)
			assert.Equal(t, completed, getStorageSwitch(t, tm.owner, operation.Id))
			page, err := tm.owner.Client.ListStorageSwitchesWithResponse(
				t.Context(),
				&apiclient.ListStorageSwitchesParams{WorkspaceId: tm.workspace.Id, DatabaseId: &databaseID},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, page.StatusCode(), "%s", page.Body)
			require.Len(t, page.JSON200.Items, 1)
			assert.Equal(t, completed, page.JSON200.Items[0])
		},
	)
	t.Run("returns a completed zero-artifact operation for a transfer with a default config", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		target := tm.createWorkspace(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, target.Id)
		response := requestDatabaseTransfer(
			t,
			tm.owner,
			databaseID,
			newDatabaseTransfer(t, target.Id, newStorageSwitchedOnTransfer(t, storageID)),
		)
		require.Equal(t, http.StatusAccepted, response.StatusCode(), "%s", response.Body)
		require.NotNil(t, response.JSON202)
		operation := getStorageSwitch(t, tm.owner, response.JSON202.Id)
		assert.Equal(t, target.Id, operation.WorkspaceId)
		assert.Nil(t, operation.PreviousStorageId)
		assert.Zero(t, operation.Total)
		assert.Equal(t, apiclient.StorageSwitchStatusCOMPLETED, operation.Status)
		assert.WithinDuration(t, operation.CreatedAt.Add(24*time.Hour), operation.CleanupDeadline, time.Microsecond)
	})
	t.Run("keeps acceptance workspace authorization after later transfers", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		outsider := tm.app.CreateNonAdminUser(t, tm.admin)
		databaseID := tm.insertLogicalDatabase(t)
		oldStorageID, targetStorageID := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorageID)
		operation := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetStorageID},
		)
		assert.Equal(t, operation.Id, getStorageSwitch(t, tm.admin, operation.Id).Id)
		assert.Equal(t, operation.Id, getStorageSwitch(t, viewer, operation.Id).Id)
		for _, client := range []*apiclient.ClientWithResponses{outsider.Client, tm.app.NewClient(t)} {
			status := http.StatusNotFound
			if client != outsider.Client {
				status = http.StatusUnauthorized
			}
			read, err := client.GetStorageSwitchWithResponse(t.Context(), operation.Id)
			require.NoError(t, err)
			assert.Equal(t, status, read.StatusCode(), "%s", read.Body)
			page, err := client.ListStorageSwitchesWithResponse(
				t.Context(),
				&apiclient.ListStorageSwitchesParams{WorkspaceId: tm.workspace.Id},
			)
			require.NoError(t, err)
			assert.Equal(t, status, page.StatusCode(), "%s", page.Body)
		}
		adminPage, err := tm.admin.Client.ListStorageSwitchesWithResponse(
			t.Context(),
			&apiclient.ListStorageSwitchesParams{WorkspaceId: tm.workspace.Id},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, adminPage.StatusCode(), "%s", adminPage.Body)
		require.Len(t, adminPage.JSON200.Items, 1)
		nextWorkspace := tm.createWorkspace(t)
		destinationViewer := tm.app.CreateNonAdminUser(t, tm.admin)
		apptest.AddWorkspaceMember(
			t,
			tm.owner,
			nextWorkspace.Id,
			destinationViewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		)
		response := requestDatabaseTransfer(
			t,
			tm.owner,
			databaseID,
			newDatabaseTransfer(t, nextWorkspace.Id, newStorageMovedWithDatabase(t)),
		)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
		assert.Equal(t, operation.Id, getStorageSwitch(t, viewer, operation.Id).Id)
		read, err := destinationViewer.Client.GetStorageSwitchWithResponse(t.Context(), operation.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, read.StatusCode(), "%s", read.Body)
		destinationHistory, err := destinationViewer.Client.ListStorageSwitchesWithResponse(t.Context(),
			&apiclient.ListStorageSwitchesParams{WorkspaceId: nextWorkspace.Id})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, destinationHistory.StatusCode(), "%s", destinationHistory.Body)
		require.NotNil(t, destinationHistory.JSON200)
		assert.Empty(t, destinationHistory.JSON200.Items)
	})
	t.Run("returns an error when durable cleanup outcomes cannot be read", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		oldStorageID, targetStorageID := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorageID)
		operation := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: targetStorageID},
		)
		tm.app.ExecuteSQL(t, "ALTER TABLE storages_deletion_batches RENAME TO unavailable_deletion_batches")
		read, err := tm.owner.Client.GetStorageSwitchWithResponse(t.Context(), operation.Id)
		page, pageErr := tm.owner.Client.ListStorageSwitchesWithResponse(
			t.Context(),
			&apiclient.ListStorageSwitchesParams{WorkspaceId: tm.workspace.Id},
		)
		tm.app.ExecuteSQL(t, "ALTER TABLE unavailable_deletion_batches RENAME TO storages_deletion_batches")
		require.NoError(t, err)
		require.NoError(t, pageErr)
		assert.Equal(t, http.StatusInternalServerError, read.StatusCode())
		assert.Equal(t, http.StatusInternalServerError, page.StatusCode())
		assert.Equal(t, operation.Id, getStorageSwitch(t, tm.owner, operation.Id).Id)
	})
}

func TestListStorageSwitches(t *testing.T) {
	t.Parallel()
	t.Run("pages by descending acceptance time and ID including equal times", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storages := []uuid.UUID{
			tm.createLocalStorage(t),
			tm.createLocalStorage(t),
			tm.createLocalStorage(t),
			tm.createLocalStorage(t),
		}
		saveLogicalBackupConfig(t, tm.owner, databaseID, storages[0])
		operations := []apiclient.StorageSwitch{}
		for _, storageID := range storages[1:] {
			operations = append(
				operations,
				acceptLogicalStorageSwitch(
					t,
					tm,
					logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageID},
				),
			)
		}
		sort.Slice(operations, func(i, j int) bool { return operations[i].Id.String() > operations[j].Id.String() })
		var after *string
		for _, operation := range operations {
			page, err := tm.owner.Client.ListStorageSwitchesWithResponse(
				t.Context(),
				&apiclient.ListStorageSwitchesParams{
					WorkspaceId: tm.workspace.Id,
					DatabaseId:  &databaseID,
					Limit:       new(1),
					After:       after,
				},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, page.StatusCode(), "%s", page.Body)
			require.Len(t, page.JSON200.Items, 1)
			assert.Equal(t, operation.Id, page.JSON200.Items[0].Id)
			after = page.JSON200.NextCursor
		}
		assert.Nil(t, after)
		for _, params := range []apiclient.ListStorageSwitchesParams{{WorkspaceId: tm.workspace.Id, After: new("broken")}, {WorkspaceId: tm.workspace.Id, Limit: new(0)}} {
			response, err := tm.owner.Client.ListStorageSwitchesWithResponse(t.Context(), &params)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode(), "%s", response.Body)
		}
	})
}

func TestStorageSwitchAttribution(t *testing.T) {
	t.Parallel()
	t.Run("keeps consecutive cleanup sets separate and exposes terminal remainders", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageA, storageB, storageC := tm.createLocalStorage(t), tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageA)
		insertLogicalBackupInStorage(t, tm, databaseID, storageA, "COMPLETED")
		first := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageB},
		)
		tm.app.ExecuteSQL(
			t,
			"UPDATE storages_pending_deletions SET not_before=not_before+interval '48 hours' WHERE storage_id=?",
			storageA,
		)
		insertLogicalBackupInStorage(t, tm, databaseID, storageB, "COMPLETED")
		second := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageC},
		)
		assert.Equal(t, &storageB, second.PreviousStorageId)
		assert.Equal(t, first.Total, second.Total)
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.Equal(t, apiclient.StorageSwitchStatusCLEANUPPENDING, getStorageSwitch(t, tm.owner, first.Id).Status)
		assert.Equal(t, apiclient.StorageSwitchStatusCOMPLETED, getStorageSwitch(t, tm.owner, second.Id).Status)
		tm.app.Clock.Advance(24 * time.Hour)
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		remaining := getStorageSwitch(t, tm.owner, first.Id)
		assert.Equal(t, apiclient.StorageSwitchStatusCOMPLETEDWITHREMAINDERS, remaining.Status)
		assert.Equal(t, first.Total, remaining.Remaining)
		removed, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), storageA)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, removed.StatusCode(), "%s", removed.Body)
		assert.Equal(t, remaining, getStorageSwitch(t, tm.owner, first.Id))
	})
	t.Run("omits previous storage and captures retained artifacts after storage was cleared", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		insertLogicalBackupInStorage(t, tm, databaseID, storageID, "COMPLETED")
		cleared := updateBackupConfig(
			t,
			tm.owner,
			databaseID,
			newLogicalBackupConfigInput(t, nil, newLogicalBackupSettings()),
		)
		require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
		operation := acceptLogicalStorageSwitch(
			t,
			tm,
			logicalStorageSwitchSpec{DatabaseID: databaseID, StorageID: storageID},
		)
		assert.Nil(t, operation.PreviousStorageId)
		assert.Positive(t, operation.Total)
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.Equal(t, operation.Total, getStorageSwitch(t, tm.owner, operation.Id).Deleted)
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
			if switchOutcome.Status == http.StatusAccepted {
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
