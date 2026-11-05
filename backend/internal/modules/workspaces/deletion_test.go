package workspaces_test

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
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

var errLaterStepFailed = errors.New("a later step of the deletion failed: secret-internal-cause")

func TestDeleteWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("returns 409 and deletes nothing when the workspace has a database", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
			t.Parallel()

			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createWebhookNotifier(t, tm.owner, tm.workspace.Id)
			storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Moved database")
			otherDatabaseID := tm.app.InsertDatabase(t, otherWorkspace.Id)
			backupID := uuid.New()
			tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
				VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, backupID, otherDatabaseID, storageID)

			response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
			requireWorkspaceMembersAndContentListed(
				t,
				tm,
				workspaceContent{databaseCount: 0, notifierCount: 1, storageCount: 1},
			)
			assert.Equal(t, []uuid.UUID{backupID}, listBackupIDs(t, tm.owner, otherDatabaseID))
		})

	t.Run("returns 409 workspace_not_empty when it has a database and its storage holds backups",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
			tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
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

	t.Run("prioritizes workspace_not_empty when it has a database and pending cleanup", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.WriteUnpublishedFile(t, storageID, "pending.dump")

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeWorkspaceNotEmpty,
		)
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT count(*) FROM databases WHERE id=?", databaseID))
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT count(*) FROM storages WHERE id=?", storageID))
		assert.Equal(
			t,
			1,
			countRows(t, tm.app, "SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=?", storageID),
		)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, "pending.dump"))
	})

	t.Run("refuses workspace deletion when its storage has pending cleanup", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, storageID, pendingFileName)
		require.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeCleanupPending,
		)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT COUNT(*) FROM storages WHERE id = ?", storageID))
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT COUNT(*) FROM storages_local WHERE storage_id = ?", storageID))
	})

	t.Run(
		"rolls back all subscribers with retained references after processing storage with pending cleanup",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createWebhookNotifier(t, tm.owner, tm.workspace.Id)
			pendingStorageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			referencedStorageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			for _, storage := range []struct {
				id   uuid.UUID
				name string
			}{
				{id: pendingStorageID, name: "A pending cleanup"},
				{id: referencedStorageID, name: "Z retained backup"},
			} {
				listed, err := tm.owner.Client.GetStorageWithResponse(t.Context(), storage.id)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, listed.StatusCode())

				renamed, err := tm.owner.Client.UpdateStorageWithResponse(
					t.Context(),
					storage.id,
					apiclient.StorageUpdate{
						Name: storage.name, Settings: listed.JSON200.Settings,
					},
				)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, renamed.StatusCode())
			}

			tm.app.WriteUnpublishedFile(t, pendingStorageID, "pending.dump")
			externalWorkspace := apptest.CreateWorkspace(t, tm.owner, "External database")
			externalDatabaseID := tm.app.InsertDatabase(t, externalWorkspace.Id)
			tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'COMPLETED', 'retained.dump')`, uuid.New(), externalDatabaseID, referencedStorageID)

			response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
			requireWorkspaceMembersAndContentListed(
				t,
				tm,
				workspaceContent{databaseCount: 0, notifierCount: 1, storageCount: 2},
			)
			assert.Equal(
				t,
				1,
				countRows(
					t,
					tm.app,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=?",
					pendingStorageID,
				),
			)
		},
	)

	t.Run("retains cleanup on external storage after its attributed workspace is removed", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		externalWorkspace := apptest.CreateWorkspace(t, tm.owner, "Surviving storage")
		externalStorageID := tm.app.CreateLocalStorage(t, tm.owner, externalWorkspace.Id)
		tm.app.RequestStorageFileDeletions(t, storages.FileDeletionRequest{
			References: []storages.StoredFileReference{{StorageID: externalStorageID, FileName: "orphan.dump"}},
			Scope:      &storages.CleanupScope{DatabaseID: uuid.New(), WorkspaceID: tm.workspace.Id},
		})

		response := deleteWorkspace(t, tm.owner, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
		assert.Equal(
			t,
			1,
			countRows(
				t,
				tm.app,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND cleanup_workspace_id=?",
				externalStorageID,
				tm.workspace.Id,
			),
		)
		tm.app.RunJobPass(t, "storage_file_deletion")
		assert.Zero(
			t,
			countRows(
				t,
				tm.app,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=?",
				externalStorageID,
			),
		)
	})

	t.Run("keeps the accounts and the other workspaces of the deleted workspace's members", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
			t.Parallel()

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
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := deleteWorkspace(t, workspaceAdmin, tm.workspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, http.StatusOK, getWorkspace(t, tm.owner, tm.workspace.Id).StatusCode())
	})

	t.Run("deletes an empty workspace when an instance admin who is not a member deletes it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response := deleteWorkspace(t, tm.admin, tm.workspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		readResponse := getWorkspace(t, tm.owner, tm.workspace.Id)
		requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 404 when an instance admin deletes a workspace that does not exist", func(t *testing.T) {
		t.Parallel()

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
	assert.Equal(t, expectedContent, countContent(t, tm.owner, tm.workspace.Id))
}

func listBackupIDs(t *testing.T, session apptest.Session, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := session.Client.ListLogicalBackupsWithResponse(t.Context(), &apiclient.ListLogicalBackupsParams{
		DatabaseId:    databaseID,
		Status:        nil,
		CreatedBefore: nil,
		Limit:         nil,
		After:         nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	backupIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedBackup := range response.JSON200.Items {
		backupIDs = append(backupIDs, listedBackup.Id)
	}

	return backupIDs
}

// The API does not list the content of a workspace that no longer exists.
func countRows(t *testing.T, app *apptest.App, query string, values ...any) int {
	t.Helper()

	var rowCount int
	app.QuerySQL(t, &rowCount, query, values...)

	return rowCount
}
