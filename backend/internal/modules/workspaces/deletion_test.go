package workspaces_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/workspaces"
)

var errLaterStepFailed = errors.New("a later step of the deletion failed: secret-internal-cause")

func TestDeleteWorkspace(t *testing.T) {
	t.Run("returns 409 and deletes nothing when the workspace has a database", func(t *testing.T) {
		tm := newTeam(t)
		tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createWebhookNotifier(t, tm.owner, tm.workspace.Id)
		tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		tm.app.InsertDatabase(t, tm.workspace.Id)

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeWorkspaceNotEmpty)
		requireWorkspaceMembersAndContentListed(
			t,
			tm,
			workspaceContent{databaseCount: 1, notifierCount: 1, storageCount: 1},
		)
	})

	t.Run("deletes the workspace with its notifier and storage when it has no databases", func(t *testing.T) {
		tm := newTeam(t)
		createWebhookNotifier(t, tm.owner, tm.workspace.Id)
		tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listAllWorkspaces(t, tm.owner.Client, 100))
		assert.Equal(
			t,
			0,
			countRows(t, tm.app, "SELECT COUNT(*) FROM notifiers WHERE workspace_id = ?", tm.workspace.Id),
		)
		assert.Equal(
			t,
			0,
			countRows(t, tm.app, "SELECT COUNT(*) FROM storages WHERE workspace_id = ?", tm.workspace.Id),
		)
	})

	t.Run("returns 409 and keeps everything when its storage holds a backup of another workspace's database",
		func(t *testing.T) {
			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createWebhookNotifier(t, tm.owner, tm.workspace.Id)
			storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Moved database")
			otherDatabaseID := tm.app.InsertDatabase(t, otherWorkspace.Id)
			backupID := uuid.New()
			tm.app.ExecuteSQL(t, `INSERT INTO logical_backups (id, database_id, storage_id, status, file_name)
				VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, backupID, otherDatabaseID, storageID)

			response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
			requireWorkspaceMembersAndContentListed(
				t,
				tm,
				workspaceContent{databaseCount: 0, notifierCount: 1, storageCount: 1},
			)
			assert.Equal(t, []uuid.UUID{backupID}, listBackupIDs(t, tm.app, tm.owner, otherDatabaseID))
		})

	t.Run("returns 409 workspace_not_empty when it has a database and its storage holds backups",
		func(t *testing.T) {
			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
			tm.app.ExecuteSQL(t, `INSERT INTO logical_backups (id, database_id, storage_id, status, file_name)
				VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, uuid.New(), databaseID, storageID)

			response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
				apiclient.ProblemCodeWorkspaceNotEmpty)
			requireWorkspaceMembersAndContentListed(
				t,
				tm,
				workspaceContent{databaseCount: 1, notifierCount: 0, storageCount: 1},
			)
		})

	t.Run("deletes the workspace and removes the file its storage was about to clean up", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, storageID, pendingFileName)
		require.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 0, countRows(t, tm.app, "SELECT COUNT(*) FROM storages WHERE id = ?", storageID))
		assert.Equal(t, 0, countRows(t, tm.app, "SELECT COUNT(*) FROM storages_local WHERE storage_id = ?", storageID))
	})

	t.Run("keeps the accounts and the other workspaces of the deleted workspace's members", func(t *testing.T) {
		tm := newTeam(t)
		viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Kept")
		apptest.AddWorkspaceMember(t, tm.owner, otherWorkspace.Id, viewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		for _, user := range []apptest.Session{tm.owner, viewer} {
			signInResponse, err := tm.app.NewClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
				Email:                    user.Email,
				Password:                 new(apptest.TestPassword),
				CloudflareTurnstileToken: nil,
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, signInResponse.StatusCode(), string(signInResponse.Body))

			listedWorkspaces := listAllWorkspaces(t, user.Client, 100)
			require.Len(t, listedWorkspaces, 1)
			assert.Equal(t, otherWorkspace.Id, listedWorkspaces[0].Id)
		}
	})

	t.Run("returns 500 and keeps everything when a later step fails", func(t *testing.T) {
		tm := newTeam(t)
		tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createWebhookNotifier(t, tm.owner, tm.workspace.Id)
		tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		tm.app.SubscribeToWorkspaceDeleting(func(context.Context, workspaces.WorkspaceDeleting) error {
			return errLaterStepFailed
		})

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusInternalServerError,
			apiclient.ProblemCodeInternal)
		assert.NotContains(t, string(response.Body), "secret-internal-cause")
		requireWorkspaceMembersAndContentListed(
			t,
			tm,
			workspaceContent{databaseCount: 0, notifierCount: 1, storageCount: 1},
		)
	})

	t.Run(
		"returns 409 and keeps everything when a database is added after the databases were checked",
		func(t *testing.T) {
			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createWebhookNotifier(t, tm.owner, tm.workspace.Id)
			tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			tm.app.SubscribeToWorkspaceDeleting(func(_ context.Context, event workspaces.WorkspaceDeleting) error {
				tm.app.InsertDatabase(t, event.WorkspaceID)

				return nil
			})

			response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
				apiclient.ProblemCodeWorkspaceNotEmpty)
			requireWorkspaceMembersAndContentListed(
				t,
				tm,
				workspaceContent{databaseCount: 1, notifierCount: 1, storageCount: 1},
			)
		},
	)

	t.Run("returns 403 and keeps the workspace when an admin of it deletes it", func(t *testing.T) {
		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := deleteWorkspace(t, workspaceAdmin, tm.workspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, http.StatusOK, getWorkspace(t, tm.owner, tm.workspace.Id).StatusCode())
	})

	t.Run("deletes an empty workspace when an instance admin who is not a member deletes it", func(t *testing.T) {
		tm := newTeam(t)

		response := deleteWorkspace(t, tm.admin, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		readResponse := getWorkspace(t, tm.owner, tm.workspace.Id)
		requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 404 when an instance admin deletes a workspace that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := deleteWorkspace(t, tm.admin, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

func deleteWorkspace(t *testing.T, actor apptest.Session, workspaceID uuid.UUID) *apiclient.DeleteWorkspaceResponse {
	t.Helper()

	response, err := actor.Client.DeleteWorkspaceWithResponse(t.Context(), workspaceID)
	require.NoError(t, err)

	return response
}

func requireWorkspaceMembersAndContentListed(t *testing.T, tm team, expectedContent workspaceContent) {
	t.Helper()

	require.Equal(t, http.StatusOK, getWorkspace(t, tm.owner, tm.workspace.Id).StatusCode())
	assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 2)
	assert.Equal(t, expectedContent, countContent(t, tm.app, tm.owner, tm.workspace.Id))
}

// listBackupIDs lists the backups of a database through the old backups route, which the contract
// does not describe yet.
func listBackupIDs(t *testing.T, app *apptest.App, session apptest.Session, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	status, body := app.CallOldAPI(t, session, apptest.OldAPICall{
		Method: http.MethodGet,
		Path:   "/backups?database_id=" + databaseID.String(),
		Body:   nil,
	})
	require.Equal(t, http.StatusOK, status, string(body))

	var listedBackups struct {
		Backups []struct {
			ID uuid.UUID `json:"id"`
		} `json:"backups"`
	}
	require.NoError(t, json.Unmarshal(body, &listedBackups), string(body))

	backupIDs := make([]uuid.UUID, 0, len(listedBackups.Backups))
	for _, listedBackup := range listedBackups.Backups {
		backupIDs = append(backupIDs, listedBackup.ID)
	}

	return backupIDs
}

// countRows reads a table directly, because the old routes refuse to list the content of a
// workspace that no longer exists.
func countRows(t *testing.T, app *apptest.App, query string, values ...any) int {
	t.Helper()

	var rowCount int
	app.QuerySQL(t, &rowCount, query, values...)

	return rowCount
}
