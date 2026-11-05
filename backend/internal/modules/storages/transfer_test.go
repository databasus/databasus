package storages_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

var errSaveNotBlockedYet = errors.New("the backup config save does not wait for the storage lock yet")

func TestTransferStorage(t *testing.T) {
	t.Parallel()

	t.Run("moves an unused storage to another workspace of the same user", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")

		response := transferStorage(t, tm.owner, createdStorage.Id, targetWorkspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, targetWorkspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a database of its workspace uses it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")
		tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: tm.app.InsertDatabase(t, tm.workspace.Id),
			StorageID:  createdStorage.Id,
		})

		response := transferStorage(t, tm.owner, createdStorage.Id, targetWorkspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		assert.Empty(t, listStorages(t, tm.owner, targetWorkspace.Id))
	})

	t.Run("returns 403 permission_denied and keeps the storage when the user only views the target",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			otherOwner := tm.stranger(t)
			targetWorkspace := apptest.CreateWorkspace(t, otherOwner, "Target")
			apptest.AddWorkspaceMember(t, otherOwner, targetWorkspace.Id, tm.owner.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createdStorage := tm.createLocalStorage(t, "Local")

			response := transferStorage(t, tm.owner, createdStorage.Id, targetWorkspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		})

	t.Run("answers a target workspace of another team exactly as a workspace that does not exist",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			otherTeamWorkspace := apptest.CreateWorkspace(t, tm.stranger(t), "Other team")
			missingWorkspaceID := uuid.New()
			createdStorage := tm.createLocalStorage(t, "Local")

			otherTeamResponse := transferStorage(t, tm.owner, createdStorage.Id, otherTeamWorkspace.Id)
			missingResponse := transferStorage(t, tm.owner, createdStorage.Id, missingWorkspaceID)

			requireProblem(t, otherTeamResponse.StatusCode(), otherTeamResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t,
				normalizeProblem(t, missingResponse.Body, missingWorkspaceID),
				normalizeProblem(t, otherTeamResponse.Body, otherTeamWorkspace.Id))
			assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		})

	t.Run("moves the database and its storage together and keeps the config on the storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: databaseID,
			StorageID:  createdStorage.Id,
		})

		var movedStorage apiclient.DatabaseTransferStorage
		require.NoError(t, movedStorage.FromStorageMovedWithDatabase(apiclient.StorageMovedWithDatabase{
			Type: apiclient.StorageMovedWithDatabaseTypeMOVEWITHDATABASE,
		}))

		response, err := tm.owner.Client.TransferDatabaseWithResponse(t.Context(), databaseID,
			apiclient.DatabaseTransfer{
				TargetWorkspaceId: targetWorkspace.Id,
				Storage:           movedStorage,
				IsMovingNotifiers: false,
				TargetNotifierIds: nil,
			})
		require.NoError(t, err)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, targetWorkspace.Id))
		assert.Equal(t, 1, countRows(t, tm.app,
			"SELECT COUNT(*) FROM databases WHERE id = ? AND workspace_id = ?", databaseID, targetWorkspace.Id))
		assert.Equal(t, &createdStorage.Id, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
	})

	t.Run("refuses a backup config save that waited for a transfer of its storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		storageLock := tm.app.LockStorageRow(t, createdStorage.Id)
		saveStatus := make(chan int, 1)

		go func() {
			saveStatus <- saveLogicalBackupConfig(t.Context(), tm, databaseID, createdStorage.Id)
		}()

		testwait.Until(t, "the backup config save waits for the storage lock", func(context.Context) error {
			if countRows(t, tm.app, `SELECT COUNT(*) FROM pg_stat_activity
				WHERE datname = current_database() AND wait_event_type = 'Lock' AND query ILIKE '%FOR SHARE%'`) == 0 {
				return errSaveNotBlockedYet
			}

			return nil
		})

		storageLock.MoveToWorkspace(t, createdStorage.Id, targetWorkspace.Id)
		storageLock.Commit(t)

		assert.Equal(t, http.StatusBadRequest, <-saveStatus)
		assert.Zero(t, countRows(t, tm.app, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
	})
}

func TestTransferStoragePermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		sourceRole     *apiclient.AssignableWorkspaceRole
		targetRole     *apiclient.AssignableWorkspaceRole
		isGlobalAdmin  bool
		expectedStatus int
	}{
		{
			name:           "moves the storage when the user is a workspace admin of both workspaces",
			sourceRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEADMIN),
			targetRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEADMIN),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "moves the storage when the user is a member of both workspaces",
			sourceRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER),
			targetRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "returns 403 when the user is a viewer of both workspaces",
			sourceRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER),
			targetRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "returns 403 when the user is a member of the source and a viewer of the target",
			sourceRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER),
			targetRole:     new(apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "moves the storage when the user is an instance admin of neither workspace",
			isGlobalAdmin:  true,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			targetOwner := tm.stranger(t)
			targetWorkspace := apptest.CreateWorkspace(t, targetOwner, "Target")
			createdStorage := tm.createLocalStorage(t, "Local")

			actor := tm.admin
			if !testCase.isGlobalAdmin {
				actor = tm.addMember(t, *testCase.sourceRole)
				apptest.AddWorkspaceMember(t, targetOwner, targetWorkspace.Id, actor.Email, *testCase.targetRole)
			}

			response := transferStorage(t, actor, createdStorage.Id, targetWorkspace.Id)

			require.Equal(t, testCase.expectedStatus, response.StatusCode(), string(response.Body))

			expectedWorkspaceID := tm.workspace.Id
			if testCase.expectedStatus == http.StatusNoContent {
				expectedWorkspaceID = targetWorkspace.Id
			}

			assert.Equal(t, expectedWorkspaceID, getStorage(t, tm.admin, createdStorage.Id).WorkspaceId)
		})
	}

	t.Run("returns 404 not_found when the owner of the source is not a member of the target", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		otherTeamWorkspace := apptest.CreateWorkspace(t, tm.stranger(t), "Other team")
		createdStorage := tm.createLocalStorage(t, "Local")

		response := transferStorage(t, tm.owner, createdStorage.Id, otherTeamWorkspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Equal(t, tm.workspace.Id, getStorage(t, tm.owner, createdStorage.Id).WorkspaceId)
	})
}

func transferStorage(
	t *testing.T,
	session apptest.Session,
	storageID uuid.UUID,
	targetWorkspaceID uuid.UUID,
) *apiclient.TransferStorageResponse {
	t.Helper()

	response, err := session.Client.TransferStorageWithResponse(t.Context(), storageID, apiclient.StorageTransfer{
		TargetWorkspaceId: targetWorkspaceID,
	})
	require.NoError(t, err)

	return response
}

// saveLogicalBackupConfig saves a logical config with the storage and returns the answer's status. It
// reports a broken request as status 0 instead of failing the test, because it runs outside the test's
// goroutine.
func saveLogicalBackupConfig(ctx context.Context, tm team, databaseID, storageID uuid.UUID) int {
	timeOfDay := "04:00"

	var settings apiclient.BackupSettings

	err := settings.FromLogicalBackupSettings(apiclient.LogicalBackupSettings{
		Type:     apiclient.LogicalBackupSettingsTypeLOGICAL,
		Schedule: apiclient.Schedule{Type: apiclient.ScheduleTypeDAILY, TimeOfDay: &timeOfDay},
		Retention: apiclient.LogicalRetention{
			Policy: apiclient.LogicalRetentionPolicyTIMEPERIOD,
			Period: apiclient.RetentionPeriodWEEK,
		},
		NotificationTriggers: []apiclient.LogicalNotificationTrigger{
			apiclient.LogicalNotificationTriggerBACKUPFAILED,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
	})
	if err != nil {
		return 0
	}

	response, err := tm.owner.Client.UpdateBackupConfigWithResponse(ctx, databaseID, apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        &storageID,
		Encryption:       apiclient.BackupEncryptionNONE,
		Settings:         settings,
	})
	if err != nil {
		return 0
	}

	return response.StatusCode()
}
