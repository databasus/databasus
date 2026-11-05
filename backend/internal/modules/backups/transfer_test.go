package backups_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestTransferDatabase(t *testing.T) {
	t.Parallel()

	t.Run("moves the database and its storage when no other database uses the storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, storageID, "COMPLETED")

		response := requestDatabaseTransfer(t, tm.owner, databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))
		assert.Equal(t, []uuid.UUID{storageID}, listWorkspaceStorageIDs(t, tm.owner, targetWorkspace.Id))
		assert.Empty(t, listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
		assert.Empty(t, listWorkspaceStorageIDs(t, tm.owner, tm.workspace.Id))
		assert.Equal(t, &storageID, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, databaseID), backup.id)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))

		tm.app.Clock.Advance(time.Hour)
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, databaseID), backup.id)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
	})

	t.Run("removes the old catalog and files when switching to the target storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)
			sourceStorageID := tm.createLocalStorage(t)
			targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)
			saveLogicalBackupConfig(t, tm.owner, databaseID, sourceStorageID)
			removedBackup := insertLogicalBackupInStorage(t, tm, databaseID, sourceStorageID, "COMPLETED")

			response := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID)))

			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))
			assert.Equal(t, []uuid.UUID{sourceStorageID}, listWorkspaceStorageIDs(t, tm.owner, tm.workspace.Id))
			assert.Equal(t, &targetStorageID, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
			assert.Zero(t, countRows(
				t,
				tm,
				"SELECT COUNT(*) FROM backups_logical WHERE id = ? AND storage_id = ?",
				removedBackup.id,
				sourceStorageID,
			))
			tm.app.RunJobPass(t, storageFileDeletionJobName)
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, removedBackup.fileName))
		})

	t.Run("saves the default config with the target storage when the config was never saved", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)
		defaultConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)

		response := requestDatabaseTransfer(t, tm.owner, databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID)))

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))

		switchedConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)
		assert.Equal(t, &targetStorageID, switchedConfig.StorageId)
		assert.Equal(t, string(apiclient.LogicalBackupSettingsTypeLOGICAL), getSettingsType(t, switchedConfig))
		assert.Equal(t, getLogicalSettings(t, defaultConfig), getLogicalSettings(t, switchedConfig))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceConfigDatabaseIDs(t, tm.owner, targetWorkspace.Id))
	})

	t.Run("moves only the notifiers that no other database uses when moving notifiers", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		neighbourDatabaseID := tm.insertLogicalDatabase(t)
		targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)
		ownNotifier := createTransferWebhookNotifier(t, tm.owner, tm.workspace.Id)
		sharedNotifier := createTransferWebhookNotifier(t, tm.owner, tm.workspace.Id)
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: databaseID,
			NotifierID: ownNotifier.Id,
		})
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: databaseID,
			NotifierID: sharedNotifier.Id,
		})
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: neighbourDatabaseID,
			NotifierID: sharedNotifier.Id,
		})

		transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID))
		transfer.IsMovingNotifiers = true
		response := requestDatabaseTransfer(t, tm.owner, databaseID, transfer)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{ownNotifier.Id}, listWorkspaceNotifierIDs(t, tm.owner, targetWorkspace.Id))
		assert.Equal(t, []uuid.UUID{sharedNotifier.Id}, listWorkspaceNotifierIDs(t, tm.owner, tm.workspace.Id))
		assert.Equal(t, []uuid.UUID{neighbourDatabaseID}, listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("replaces the database's notifiers with notifiers of the target workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)
		sourceNotifier := createTransferWebhookNotifier(t, tm.owner, tm.workspace.Id)
		targetNotifier := createTransferWebhookNotifier(t, tm.owner, targetWorkspace.Id)
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: databaseID,
			NotifierID: sourceNotifier.Id,
		})

		transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID))
		transfer.TargetNotifierIds = &[]uuid.UUID{targetNotifier.Id}
		response := requestDatabaseTransfer(t, tm.owner, databaseID, transfer)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{targetNotifier.Id}, listAttachedNotifierIDsOfDatabase(t, tm, databaseID))
		assert.Equal(t, []uuid.UUID{sourceNotifier.Id}, listWorkspaceNotifierIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request and moves nothing when a replacing notifier does not exist",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

			transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t))
			transfer.TargetNotifierIds = &[]uuid.UUID{uuid.New()}
			response := requestDatabaseTransfer(t, tm.owner, databaseID, transfer)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
		})

	t.Run("returns 400 invalid_request and moves nothing when a replacing notifier is outside the target",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
			sourceNotifier := createTransferWebhookNotifier(t, tm.owner, tm.workspace.Id)

			transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t))
			transfer.TargetNotifierIds = &[]uuid.UUID{sourceNotifier.Id}
			response := requestDatabaseTransfer(t, tm.owner, databaseID, transfer)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
		})

	t.Run("returns 409 in_use and moves nothing when another database backs up into the storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)
			neighbourDatabaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
			saveLogicalBackupConfig(t, tm.owner, neighbourDatabaseID, storageID)

			response := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
			assert.ElementsMatch(t, []uuid.UUID{databaseID, neighbourDatabaseID},
				listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
			assert.Equal(t, []uuid.UUID{storageID}, listWorkspaceStorageIDs(t, tm.owner, tm.workspace.Id))
			assert.Empty(t, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))
			assert.Empty(t, listWorkspaceStorageIDs(t, tm.owner, targetWorkspace.Id))
		})

	t.Run("returns 400 invalid_request and moves nothing when moving the storage of a config without one",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)

			response := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
			assert.Empty(t, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))
		})

	t.Run("returns 400 invalid_request and moves nothing when the target storage is outside the target",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetWorkspace := tm.createWorkspace(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

			response := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, storageID)))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
		})
}

func TestTransferDatabasePermissions(t *testing.T) {
	t.Parallel()

	t.Run("moves the database when the user is a member of both workspaces", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		targetWorkspace := tm.createWorkspace(t)
		apptest.AddWorkspaceMember(t, tm.owner, targetWorkspace.Id, member.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, member, databaseID, storageID)

		response := requestDatabaseTransfer(t, member, databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, targetWorkspace.Id))
	})

	t.Run("returns 403 permission_denied when the user is a viewer of the source workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		targetWorkspace := tm.createWorkspace(t)
		apptest.AddWorkspaceMember(t, tm.owner, targetWorkspace.Id, viewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

		response := requestDatabaseTransfer(t, viewer, databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
	})

	t.Run("returns 403 permission_denied when the user is a viewer of the target workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		targetWorkspace := tm.createWorkspace(t)
		apptest.AddWorkspaceMember(t, tm.owner, targetWorkspace.Id, member.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

		response := requestDatabaseTransfer(t, member, databaseID,
			newDatabaseTransfer(t, targetWorkspace.Id, newStorageMovedWithDatabase(t)))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireNothingTransferred(t, tm, targetWorkspace.Id, databaseID, storageID)
	})

	t.Run("answers a target workspace of another team exactly as a workspace that does not exist",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			otherOwner := tm.app.CreateNonAdminUser(t, tm.admin)
			otherTeamWorkspace := apptest.CreateWorkspace(t, otherOwner, "Other team "+uuid.NewString()[:8])
			missingWorkspaceID := uuid.New()
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

			otherTeamResponse := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, otherTeamWorkspace.Id, newStorageMovedWithDatabase(t)))
			missingResponse := requestDatabaseTransfer(t, tm.owner, databaseID,
				newDatabaseTransfer(t, missingWorkspaceID, newStorageMovedWithDatabase(t)))

			requireProblem(t, otherTeamResponse.StatusCode(), otherTeamResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t,
				normalizeTransferProblem(t, missingResponse.Body, missingWorkspaceID),
				normalizeTransferProblem(t, otherTeamResponse.Body, otherTeamWorkspace.Id))
			requireNothingTransferred(t, tm, otherTeamWorkspace.Id, databaseID, storageID)
		})
}

func TestTransferDatabaseOutsideSourceWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("answers a database of another team exactly as a database that does not exist", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)
		strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Stranger "+uuid.NewString()[:8])
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		missingDatabaseID := uuid.New()

		strangerResponse := requestDatabaseTransfer(t, stranger, databaseID,
			newDatabaseTransfer(t, strangerWorkspace.Id, newStorageMovedWithDatabase(t)))
		missingResponse := requestDatabaseTransfer(t, stranger, missingDatabaseID,
			newDatabaseTransfer(t, strangerWorkspace.Id, newStorageMovedWithDatabase(t)))

		requireProblem(t, strangerResponse.StatusCode(), strangerResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(missingResponse.Body), string(strangerResponse.Body))
		requireNothingTransferred(t, tm, strangerWorkspace.Id, databaseID, storageID)
	})
}

func requestDatabaseTransfer(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	transfer apiclient.DatabaseTransfer,
) *apiclient.TransferDatabaseResponse {
	t.Helper()

	response, err := session.Client.TransferDatabaseWithResponse(t.Context(), databaseID, transfer)
	require.NoError(t, err)

	return response
}

func newDatabaseTransfer(
	t *testing.T,
	targetWorkspaceID uuid.UUID,
	storage apiclient.DatabaseTransferStorage,
) apiclient.DatabaseTransfer {
	t.Helper()

	return apiclient.DatabaseTransfer{
		TargetWorkspaceId: targetWorkspaceID,
		Storage:           storage,
		IsMovingNotifiers: false,
		TargetNotifierIds: nil,
	}
}

func newStorageMovedWithDatabase(t *testing.T) apiclient.DatabaseTransferStorage {
	t.Helper()

	var storage apiclient.DatabaseTransferStorage
	require.NoError(t, storage.FromStorageMovedWithDatabase(apiclient.StorageMovedWithDatabase{
		Type: apiclient.StorageMovedWithDatabaseTypeMOVEWITHDATABASE,
	}))

	return storage
}

func newStorageSwitchedOnTransfer(t *testing.T, targetStorageID uuid.UUID) apiclient.DatabaseTransferStorage {
	t.Helper()

	var storage apiclient.DatabaseTransferStorage
	require.NoError(t, storage.FromStorageSwitchedOnTransfer(apiclient.StorageSwitchedOnTransfer{
		Type:      apiclient.StorageSwitchedOnTransferTypeSWITCHTOSTORAGE,
		StorageId: targetStorageID,
	}))

	return storage
}

func createTransferWebhookNotifier(t *testing.T, session apptest.Session, workspaceID uuid.UUID) apiclient.Notifier {
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

	return apptest.CreateNotifier(t, session, workspaceID, settings)
}

func requireNothingTransferred(
	t *testing.T,
	tm team,
	targetWorkspaceID uuid.UUID,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) {
	t.Helper()

	assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceDatabaseIDs(t, tm.owner, tm.workspace.Id))
	assert.Equal(t, []uuid.UUID{storageID}, listWorkspaceStorageIDs(t, tm.owner, tm.workspace.Id))
	assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM databases WHERE workspace_id = ?", targetWorkspaceID))
	assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM storages WHERE workspace_id = ?", targetWorkspaceID))
	assert.Equal(t, &storageID, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
}

func listWorkspaceDatabaseIDs(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := session.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
		WorkspaceId: workspaceID,
		NotifierId:  nil,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	databaseIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedDatabase := range response.JSON200.Items {
		databaseIDs = append(databaseIDs, listedDatabase.Id)
	}

	return databaseIDs
}

func listWorkspaceStorageIDs(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := session.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	storageIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedStorage := range response.JSON200.Items {
		storageIDs = append(storageIDs, listedStorage.Id)
	}

	return storageIDs
}

func listWorkspaceNotifierIDs(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := session.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	notifierIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedNotifier := range response.JSON200.Items {
		notifierIDs = append(notifierIDs, listedNotifier.Id)
	}

	return notifierIDs
}

func listAttachedNotifierIDsOfDatabase(t *testing.T, tm team, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	var notifierIDs []uuid.UUID
	tm.app.QuerySQL(t, &notifierIDs, "SELECT notifier_id FROM databases_notifiers WHERE database_id = ?", databaseID)

	return notifierIDs
}

func normalizeTransferProblem(t *testing.T, body []byte, workspaceID uuid.UUID) map[string]any {
	t.Helper()

	var problem map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.ReplaceAll(string(body), workspaceID.String(), "")), &problem))

	return problem
}
