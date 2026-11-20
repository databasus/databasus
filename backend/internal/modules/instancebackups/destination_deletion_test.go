package instancebackups_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestDeleteSelectedDestinationStorage(t *testing.T) {
	t.Parallel()

	t.Run("refuses with databasus_backup_in_use before database backup references and cleanup", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		workspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Shared")
		storageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		databaseID := owner.App.InsertDatabase(t, workspace.Id)
		owner.App.InsertBackupConfigUsingStorage(t, apptest.StorageUse{DatabaseID: databaseID, StorageID: storageID})
		insertRetainedDatabaseBackup(t, owner, databaseID, storageID)
		pendingFileName := "pending-" + uuid.NewString()
		owner.App.WriteUnpublishedFile(t, storageID, pendingFileName)
		owner.selectDestinations(t, storageID)
		recordsBeforeDeletion := readStorageRecords(t, owner, storageID)

		selectedResponse := deleteStorage(t, owner.RootAdmin, storageID, true)
		owner.selectDestinations(t)
		referencedResponse := deleteStorage(t, owner.RootAdmin, storageID, true)

		requireProblemCode(t, selectedResponse.StatusCode(), selectedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeDatabasusBackupInUse)
		requireProblemCode(t, referencedResponse.StatusCode(), referencedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, http.StatusOK, getStorageStatus(t, owner.RootAdmin, storageID))
		assert.Equal(t, recordsBeforeDeletion, readStorageRecords(t, owner, storageID))
		assert.Equal(t, 1, countPendingFileDeletions(t, owner, pendingFileName))
		assert.True(t, owner.App.IsLocalStorageFilePresent(t, pendingFileName))
	})

	t.Run("deletes a deselected storage once its pending cleanup is abandoned", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.App.WriteUnpublishedFile(t, storageID, "pending-"+uuid.NewString())
		owner.selectDestinations(t, storageID)
		selectedResponse := deleteStorage(t, owner.RootAdmin, storageID, false)
		owner.selectDestinations(t)

		pendingResponse := deleteStorage(t, owner.RootAdmin, storageID, false)
		abandonedResponse := deleteStorage(t, owner.RootAdmin, storageID, true)

		requireProblemCode(t, selectedResponse.StatusCode(), selectedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeDatabasusBackupInUse)
		requireProblemCode(t, pendingResponse.StatusCode(), pendingResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeCleanupPending)
		assert.Equal(t, http.StatusNoContent, abandonedResponse.StatusCode(), string(abandonedResponse.Body))
		assert.Equal(t, http.StatusNotFound, getStorageStatus(t, owner.RootAdmin, storageID))
	})

	t.Run(
		"keeps a selected storage editable and transferable, and still selected after a transfer",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			storageID := owner.createLocalStorage(t, "Source")
			targetWorkspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Target")
			owner.selectDestinations(t, storageID)

			updateResponse, err := owner.RootAdmin.Client.UpdateStorageWithResponse(t.Context(), storageID,
				apiclient.StorageUpdate{Name: "Renamed archive", Settings: newLocalStorageSettings(t)})
			require.NoError(t, err)
			transferResponse, err := owner.RootAdmin.Client.TransferStorageWithResponse(t.Context(), storageID,
				apiclient.StorageTransfer{TargetWorkspaceId: targetWorkspace.Id})
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))
			assert.Equal(t, http.StatusNoContent, transferResponse.StatusCode(), string(transferResponse.Body))
			assert.Equal(t, []uuid.UUID{storageID}, owner.getSettings(t).StorageIds)
		},
	)

	t.Run("lets a workspace member change a selected storage and delivers the next backup there",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			workspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Shared")
			selectedNamespace := apptest.CreateStorageNamespace(t, owner.RootAdmin, workspace.Id)
			changedNamespace := apptest.CreateStorageNamespace(t, owner.RootAdmin,
				apptest.CreateWorkspace(t, owner.RootAdmin, "Other").Id)
			member := owner.App.SignUp(t, apptest.NewTestEmail("member"))
			apptest.AddWorkspaceMember(t, owner.RootAdmin, workspace.Id, member.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			owner.selectDestinations(t, selectedNamespace.StorageID)

			updateStatus := moveStorageToNamespace(t, member, selectedNamespace.StorageID, changedNamespace)
			backup := owner.backUp(t)

			require.Equal(t, http.StatusOK, updateStatus)
			assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED,
				getCopy(t, backup, selectedNamespace.StorageID).Status)
			storedFileName := owner.getStoredFileName(t, backup, selectedNamespace.StorageID)
			changedNamespace.RequireFile(t, storedFileName)
			selectedNamespace.RequireFileAbsent(t, storedFileName)
		})

	t.Run("either selects or deletes a storage when both race, never both", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		for range 5 {
			storageID := owner.createLocalStorage(t, "Racing "+uuid.NewString()[:8])
			var selectionStatus, deletionStatus int
			var waitGroup sync.WaitGroup

			waitGroup.Go(func() { selectionStatus = requestSelection(t, owner, storageID) })
			waitGroup.Go(func() { deletionStatus = deleteStorage(t, owner.RootAdmin, storageID, false).StatusCode() })
			waitGroup.Wait()

			isSelected := len(owner.getSettings(t).StorageIds) == 1
			isDeleted := getStorageStatus(t, owner.RootAdmin, storageID) == http.StatusNotFound
			assert.NotEqual(t, isSelected, isDeleted)
			if isSelected {
				assert.Equal(t, http.StatusOK, selectionStatus)
				assert.Equal(t, http.StatusConflict, deletionStatus)
				owner.selectDestinations(t)
			} else {
				assert.Equal(t, http.StatusBadRequest, selectionStatus)
				assert.Equal(t, http.StatusNoContent, deletionStatus)
			}
		}
	})
}

func TestDeleteWorkspaceWithSelectedDestination(t *testing.T) {
	t.Parallel()

	t.Run("refuses in the order of databases, destinations, references and cleanup", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		workspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Archive")
		otherWorkspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Other")
		selectedStorageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		referencedStorageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		pendingStorageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		ownDatabaseID := owner.App.InsertDatabase(t, workspace.Id)
		otherDatabaseID := owner.App.InsertDatabase(t, otherWorkspace.Id)
		owner.App.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: otherDatabaseID, StorageID: referencedStorageID,
		})
		owner.App.WriteUnpublishedFile(t, pendingStorageID, "pending-"+uuid.NewString())
		owner.selectDestinations(t, selectedStorageID)

		withDatabaseResponse := deleteWorkspace(t, owner.RootAdmin, workspace.Id)
		deleteDatabase(t, owner.RootAdmin, ownDatabaseID)
		selectedResponse := deleteWorkspace(t, owner.RootAdmin, workspace.Id)
		owner.selectDestinations(t)
		referencedResponse := deleteWorkspace(t, owner.RootAdmin, workspace.Id)
		deleteDatabase(t, owner.RootAdmin, otherDatabaseID)
		pendingResponse := deleteWorkspace(t, owner.RootAdmin, workspace.Id)
		require.Equal(t, http.StatusNoContent, deleteStorage(t, owner.RootAdmin, pendingStorageID, true).StatusCode())
		deletedResponse := deleteWorkspace(t, owner.RootAdmin, workspace.Id)

		requireProblemCode(t, withDatabaseResponse.StatusCode(), withDatabaseResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeWorkspaceNotEmpty)
		requireProblemCode(t, selectedResponse.StatusCode(), selectedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeDatabasusBackupInUse)
		requireProblemCode(t, referencedResponse.StatusCode(), referencedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeInUse)
		requireProblemCode(t, pendingResponse.StatusCode(), pendingResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeCleanupPending)
		assert.Equal(t, http.StatusNoContent, deletedResponse.StatusCode(), string(deletedResponse.Body))
	})

	t.Run("keeps every storage of the workspace when a selected destination refuses", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		workspace := apptest.CreateWorkspace(t, owner.RootAdmin, "Archive")
		unusedStorageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		selectedStorageID := owner.App.CreateLocalStorage(t, owner.RootAdmin, workspace.Id)
		workspaceNotifier := apptest.CreateNotifier(t, owner.RootAdmin, workspace.Id, newWebhookNotifierSettings(t))
		member := owner.App.SignUp(t, apptest.NewTestEmail("member"))
		apptest.AddWorkspaceMember(t, owner.RootAdmin, workspace.Id, member.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		owner.selectDestinations(t, selectedStorageID)

		response := deleteWorkspace(t, owner.RootAdmin, workspace.Id)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeDatabasusBackupInUse)
		assert.Equal(t, http.StatusOK, getStorageStatus(t, owner.RootAdmin, unusedStorageID))
		assert.Equal(t, http.StatusOK, getStorageStatus(t, owner.RootAdmin, selectedStorageID))
		workspaceResponse, err := owner.RootAdmin.Client.GetWorkspaceWithResponse(t.Context(), workspace.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, workspaceResponse.StatusCode(), string(workspaceResponse.Body))
		notifierResponse, err := owner.RootAdmin.Client.GetNotifierWithResponse(t.Context(), workspaceNotifier.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, notifierResponse.StatusCode(), string(notifierResponse.Body))
		assert.Contains(t, listMemberEmails(t, owner.RootAdmin, workspace.Id), member.Email)
	})
}

func deleteStorage(
	t *testing.T,
	session apptest.Session,
	storageID uuid.UUID,
	shouldAbandonCleanup bool,
) *apiclient.DeleteStorageResponse {
	t.Helper()

	response, err := session.Client.DeleteStorageWithResponse(t.Context(), storageID,
		&apiclient.DeleteStorageParams{ShouldAbandonCleanup: &shouldAbandonCleanup})
	require.NoError(t, err)

	return response
}

func deleteWorkspace(t *testing.T, session apptest.Session, workspaceID uuid.UUID) *apiclient.DeleteWorkspaceResponse {
	t.Helper()

	response, err := session.Client.DeleteWorkspaceWithResponse(t.Context(), workspaceID)
	require.NoError(t, err)

	return response
}

func deleteDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) {
	t.Helper()

	response, err := session.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
}

func getStorageStatus(t *testing.T, session apptest.Session, storageID uuid.UUID) int {
	t.Helper()

	response, err := session.Client.GetStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)

	return response.StatusCode()
}

func requestSelection(t *testing.T, owner instanceOwner, storageID uuid.UUID) int {
	t.Helper()

	settings := owner.getSettings(t)
	settings.StorageIds = []uuid.UUID{storageID}

	response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	require.NoError(t, err)

	return response.StatusCode()
}

func newLocalStorageSettings(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromLocalStorageSettings(apiclient.LocalStorageSettings{
		Type: apiclient.LocalStorageSettingsTypeLOCAL,
	}))

	return settings
}

// insertRetainedDatabaseBackup writes the record directly, because a completed database backup through
// the contract needs a reachable database.
func insertRetainedDatabaseBackup(t *testing.T, owner instanceOwner, databaseID, storageID uuid.UUID) {
	t.Helper()

	backupID := uuid.New()
	owner.App.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, 'COMPLETED', ?)`, backupID, databaseID, storageID, "backup-"+backupID.String())
}

// readStorageRecords reads every record that refers to the storage, which the API does not return
// as a whole: the storage, its database backups and configurations, and its queued file deletions.
func readStorageRecords(t *testing.T, owner instanceOwner, storageID uuid.UUID) string {
	t.Helper()

	var records string
	owner.App.QuerySQL(t, &records, `SELECT json_build_array(
		(SELECT row_to_json(storage) FROM storages storage WHERE id = ?),
		(SELECT json_agg(backup ORDER BY id) FROM backups_logical backup WHERE storage_id = ?),
		(SELECT json_agg(config ORDER BY database_id) FROM backups_configs config WHERE storage_id = ?),
		(SELECT json_agg(pending ORDER BY id) FROM storages_pending_file_deletions pending WHERE storage_id = ?)
	)::text`, storageID, storageID, storageID, storageID)

	return records
}

func countPendingFileDeletions(t *testing.T, owner instanceOwner, fileName string) int {
	t.Helper()

	var pendingCount int
	owner.App.QuerySQL(t, &pendingCount,
		"SELECT count(*) FROM storages_pending_file_deletions WHERE file_name = ?", fileName)

	return pendingCount
}

// moveStorageToNamespace points the S3 storage at the bucket of another namespace, as an editor
// changing the connection of a storage does, and returns the status of the update.
func moveStorageToNamespace(
	t *testing.T,
	editor apptest.Session,
	storageID uuid.UUID,
	target apptest.StorageNamespace,
) int {
	t.Helper()

	storageResponse, err := editor.Client.GetStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.NotNil(t, storageResponse.JSON200, string(storageResponse.Body))

	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromS3StorageSettings(apiclient.S3StorageSettings{
		Type:        apiclient.S3StorageSettingsTypeS3,
		S3Bucket:    target.Bucket,
		S3Region:    new(containers.MinioRegion),
		S3Endpoint:  new(target.Client.EndpointURL().String()),
		S3AccessKey: new(containers.MinioRootUser),
		S3SecretKey: new(containers.MinioRootPassword),
	}))

	updateResponse, err := editor.Client.UpdateStorageWithResponse(t.Context(), storageID,
		apiclient.StorageUpdate{Name: storageResponse.JSON200.Name, Settings: settings})
	require.NoError(t, err)

	return updateResponse.StatusCode()
}

func newWebhookNotifierSettings(t *testing.T) apiclient.NotifierSettings {
	t.Helper()

	var settings apiclient.NotifierSettings
	require.NoError(t, settings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         "https://hooks.example.com",
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		BodyTemplate:       nil,
		Headers:            []apiclient.WebhookHeader{},
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
	}))

	return settings
}

func listMemberEmails(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []string {
	t.Helper()

	limit := 100

	response, err := session.Client.ListWorkspaceMembersWithResponse(t.Context(),
		&apiclient.ListWorkspaceMembersParams{WorkspaceId: workspaceID, Limit: &limit, After: nil})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	memberEmails := make([]string, 0, len(response.JSON200.Items))
	for _, member := range response.JSON200.Items {
		memberEmails = append(memberEmails, member.Email)
	}

	return memberEmails
}
