package storages_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	t.Run("moves an unused storage to another workspace of the same user", func(t *testing.T) {
		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")

		response := transferStorage(t, tm.owner, createdStorage.Id, targetWorkspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, targetWorkspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a database of its workspace uses it", func(t *testing.T) {
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
		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target")
		createdStorage := tm.createLocalStorage(t, "Local")
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: databaseID,
			StorageID:  createdStorage.Id,
		})

		status, body := tm.app.CallOldAPI(t, tm.owner, apptest.OldAPICall{
			Method: http.MethodPost,
			Path:   "/backup-configs/database/" + databaseID.String() + "/transfer",
			Body:   map[string]any{"targetWorkspaceId": targetWorkspace.Id, "isTransferWithStorage": true},
		})

		require.Equal(t, http.StatusOK, status, string(body))
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, targetWorkspace.Id))
		assert.Equal(t, 1, countRows(t, tm.app,
			"SELECT COUNT(*) FROM databases WHERE id = ? AND workspace_id = ?", databaseID, targetWorkspace.Id))
		assert.Equal(t, 1, countRows(t, tm.app,
			"SELECT COUNT(*) FROM logical_backup_configs WHERE database_id = ? AND storage_id = ?",
			databaseID, createdStorage.Id))
	})

	t.Run("refuses a backup config save that waited for a transfer of its storage", func(t *testing.T) {
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
		assert.Equal(t, 0, countRows(t, tm.app,
			"SELECT COUNT(*) FROM logical_backup_configs WHERE database_id = ? AND storage_id = ?",
			databaseID, createdStorage.Id))
	})
}

func TestTransferStoragePermissions(t *testing.T) {
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

// saveLogicalBackupConfig sends the save through the old route and returns its status. It reports a
// broken request as status 0 instead of failing the test, because it runs outside the test's
// goroutine.
func saveLogicalBackupConfig(ctx context.Context, tm team, databaseID, storageID uuid.UUID) int {
	body, err := json.Marshal(map[string]any{
		"databaseId":          databaseID,
		"isBackupsEnabled":    true,
		"retentionPolicyType": "TIME_PERIOD",
		"retentionTimePeriod": "WEEK",
		"backupInterval":      map[string]any{"type": "DAILY", "timeOfDay": "04:00"},
		"storage":             map[string]any{"id": storageID},
		"sendNotificationsOn": []string{"BACKUP_FAILED"},
		"isRetryIfFailed":     true,
		"maxFailedTriesCount": 3,
		"encryption":          "NONE",
	})
	if err != nil {
		return 0
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/backup-configs/save", tm.app.URL), bytes.NewReader(body))
	if err != nil {
		return 0
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+tm.owner.Token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0
	}

	defer func() { _ = response.Body.Close() }()

	return response.StatusCode
}
