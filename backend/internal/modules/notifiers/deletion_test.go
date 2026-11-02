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
	"databasus-backend/internal/modules/workspaces"
)

var errLaterHandlerRefused = errors.New("a later handler refuses the deletion")

func TestDeleteNotifier(t *testing.T) {
	t.Parallel()

	t.Run("returns 409 in_use and keeps the notifier when a database uses it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.AttachNotifierToDatabase(t, apptest.NotifierAttachment{
			DatabaseID: databaseID,
			NotifierID: createdNotifier.Id,
		})

		response, err := tm.owner.Client.DeleteNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Len(t, listNotifiers(t, tm.owner, tm.workspace.Id), 1)
		assert.Equal(t, []uuid.UUID{createdNotifier.Id}, listAttachedNotifierIDs(t, tm.app, databaseID))
	})

	t.Run("deletes a notifier that no database uses", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response, err := tm.owner.Client.DeleteNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 403 permission_denied and keeps the notifier when a viewer deletes it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response, err := viewer.Client.DeleteNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Len(t, listNotifiers(t, tm.owner, tm.workspace.Id), 1)
	})
}

func TestDeleteWorkspaceWithNotifiers(t *testing.T) {
	t.Parallel()

	t.Run("deletes the notifiers of the workspace with it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Zero(t, countNotifierRows(t, tm.app, tm.workspace.Id))
	})

	t.Run("keeps the notifiers when a later handler rolls the deletion back", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		tm.app.SubscribeToWorkspaceDeleting(func(context.Context, workspaces.WorkspaceDeleting) error {
			return errLaterHandlerRefused
		})

		response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode(), string(response.Body))
		assert.Len(t, listNotifiers(t, tm.owner, tm.workspace.Id), 1)
	})
}

func listAttachedNotifierIDs(t *testing.T, app *apptest.App, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	var notifierIDs []uuid.UUID
	app.QuerySQL(t, &notifierIDs, "SELECT notifier_id FROM databases_notifiers WHERE database_id = ?", databaseID)

	return notifierIDs
}

func countNotifierRows(t *testing.T, app *apptest.App, workspaceID uuid.UUID) int {
	t.Helper()

	var notifierCount int
	app.QuerySQL(t, &notifierCount, "SELECT COUNT(*) FROM notifiers WHERE workspace_id = ?", workspaceID)

	return notifierCount
}
