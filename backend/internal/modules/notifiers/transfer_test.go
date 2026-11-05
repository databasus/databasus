package notifiers_test

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

var errTransferNotWaiting = errors.New("no transfer waits for the lock on the notifier")

func TestTransferNotifier(t *testing.T) {
	t.Parallel()

	t.Run("moves an unused notifier to another workspace of the user", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response := transferNotifier(t, tm.owner, createdNotifier.Id, targetWorkspace.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
		require.Len(t, listNotifiers(t, tm.owner, targetWorkspace.Id), 1)
		assert.Equal(t, createdNotifier.Id, listNotifiers(t, tm.owner, targetWorkspace.Id)[0].Id)
	})

	t.Run("returns 409 in_use and keeps the notifier when a database of its workspace uses it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: tm.app.InsertDatabase(t, tm.workspace.Id),
			NotifierID: createdNotifier.Id,
		})

		response := transferNotifier(t, tm.owner, createdNotifier.Id, targetWorkspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, tm.workspace.Id, getNotifier(t, tm.owner, createdNotifier.Id).WorkspaceId)
	})

	t.Run("waits for an attach in flight and then returns 409 in_use", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		commitAttachment := tm.app.BeginNotifierAttachment(t, apptest.NotifierAttachment{
			DatabaseID: tm.app.InsertDatabase(t, tm.workspace.Id),
			NotifierID: createdNotifier.Id,
		})

		transferResponses := make(chan *apiclient.TransferNotifierResponse, 1)
		go func() {
			response, _ := tm.owner.Client.TransferNotifierWithResponse(t.Context(), createdNotifier.Id,
				apiclient.NotifierTransfer{TargetWorkspaceId: targetWorkspace.Id})
			transferResponses <- response
		}()

		testwait.Until(t, "the transfer waits for the attachment's lock on the notifier", func(context.Context) error {
			if !isNotifierLockAwaited(t, tm) {
				return errTransferNotWaiting
			}

			return nil
		})
		commitAttachment()
		response := <-transferResponses

		require.NotNil(t, response)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, tm.workspace.Id, getNotifier(t, tm.owner, createdNotifier.Id).WorkspaceId)
	})

	t.Run("returns 403 permission_denied when the user is only a viewer of the target workspace",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
			apptest.AddWorkspaceMember(t, tm.owner, targetWorkspace.Id, member.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			createdNotifier := apptest.CreateNotifier(t, member, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

			response := transferNotifier(t, member, createdNotifier.Id, targetWorkspace.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Equal(t, tm.workspace.Id, getNotifier(t, tm.owner, createdNotifier.Id).WorkspaceId)
		})

	t.Run("returns 404 not_found when the user is not a member of the notifier's workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Stranger "+uuid.NewString()[:8])
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response := transferNotifier(t, stranger, createdNotifier.Id, strangerWorkspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Equal(t, tm.workspace.Id, getNotifier(t, tm.owner, createdNotifier.Id).WorkspaceId)
	})
}

func TestTransferDatabaseWithNotifiers(t *testing.T) {
	t.Parallel()

	t.Run("moves the database and its notifier and keeps the notifier attached", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: databaseID,
			NotifierID: createdNotifier.Id,
		})
		targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)

		var switchedStorage apiclient.DatabaseTransferStorage
		require.NoError(t, switchedStorage.FromStorageSwitchedOnTransfer(apiclient.StorageSwitchedOnTransfer{
			Type:      apiclient.StorageSwitchedOnTransferTypeSWITCHTOSTORAGE,
			StorageId: targetStorageID,
		}))

		response, err := tm.owner.Client.TransferDatabaseWithResponse(t.Context(), databaseID,
			apiclient.DatabaseTransfer{
				TargetWorkspaceId: targetWorkspace.Id,
				Storage:           switchedStorage,
				IsMovingNotifiers: true,
				TargetNotifierIds: nil,
			})
		require.NoError(t, err)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, targetWorkspace.Id, getNotifier(t, tm.owner, createdNotifier.Id).WorkspaceId)
		assert.Equal(t, targetWorkspace.Id, getDatabaseWorkspaceID(t, tm, databaseID))
		assert.Equal(t, []uuid.UUID{createdNotifier.Id}, listAttachedNotifierIDs(t, tm.app, databaseID))
	})
}

func transferNotifier(
	t *testing.T,
	session apptest.Session,
	notifierID uuid.UUID,
	targetWorkspaceID uuid.UUID,
) *apiclient.TransferNotifierResponse {
	t.Helper()

	response, err := session.Client.TransferNotifierWithResponse(t.Context(), notifierID, apiclient.NotifierTransfer{
		TargetWorkspaceId: targetWorkspaceID,
	})
	require.NoError(t, err)

	return response
}

func isNotifierLockAwaited(t *testing.T, tm team) bool {
	t.Helper()

	var waitingPIDs []int
	tm.app.QuerySQL(t, &waitingPIDs, `
		SELECT pid FROM pg_stat_activity
		WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%FROM notifiers%FOR UPDATE%'
	`)

	return len(waitingPIDs) > 0
}

func getDatabaseWorkspaceID(t *testing.T, tm team, databaseID uuid.UUID) uuid.UUID {
	t.Helper()

	response, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return response.JSON200.WorkspaceId
}
