package notifiers_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestNotifierAccess(t *testing.T) {
	t.Parallel()

	t.Run("lets a viewer list and read the notifiers of the workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		listedNotifiers := listNotifiers(t, viewer, tm.workspace.Id)
		readNotifier := getNotifier(t, viewer, createdNotifier.Id)

		require.Len(t, listedNotifiers, 1)
		assert.Equal(t, createdNotifier.Id, listedNotifiers[0].Id)
		assert.Equal(t, createdNotifier.Id, readNotifier.Id)
	})

	t.Run("lets a member create, change and delete a notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		createdNotifier := apptest.CreateNotifier(t, member, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		updateResponse := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     "Renamed",
			Settings: newWebhookSettings(t, newPostWebhook("https://hooks.example.com/renamed")),
		})
		deleteResponse, err := member.Client.DeleteNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))
		assert.Equal(t, http.StatusNoContent, deleteResponse.StatusCode(), string(deleteResponse.Body))
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})

	t.Run("answers a notifier of another team as a missing one", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))
		missingNotifierID := uuid.New()

		foreignResponse, err := stranger.Client.GetNotifierWithResponse(t.Context(), createdNotifier.Id)
		require.NoError(t, err)
		missingResponse, err := stranger.Client.GetNotifierWithResponse(t.Context(), missingNotifierID)
		require.NoError(t, err)

		requireProblem(t, foreignResponse.StatusCode(), foreignResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.Equal(t,
			string(bytes.ReplaceAll(foreignResponse.Body, []byte(createdNotifier.Id.String()), nil)),
			string(bytes.ReplaceAll(missingResponse.Body, []byte(missingNotifierID.String()), nil)),
		)
	})

	t.Run("returns 404 not_found when a user lists the notifiers of another team", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)

		response, err := stranger.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       nil,
			After:       nil,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 403 permission_denied and keeps the name when a viewer renames a notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response := updateNotifier(t, viewer, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     "Renamed by viewer",
			Settings: newWebhookSettings(t, newPostWebhook("https://hooks.example.com")),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, createdNotifier.Name, getNotifier(t, tm.owner, createdNotifier.Id).Name)
	})

	t.Run("returns 404 not_found and keeps the notifier when a user of another team changes it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response := updateNotifier(t, stranger, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     "Renamed",
			Settings: newWebhookSettings(t, newPostWebhook("https://hooks.example.com/renamed")),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Equal(t, createdNotifier.Name, getNotifier(t, tm.owner, createdNotifier.Id).Name)
	})

	t.Run("returns 404 not_found and keeps the notifier when a user of another team deletes it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com")))

		response, err := stranger.Client.DeleteNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		require.Len(t, listNotifiers(t, tm.owner, tm.workspace.Id), 1)
	})

	t.Run("returns 403 permission_denied and creates nothing when a viewer creates a notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response, err := viewer.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Hook",
			Settings:    newWebhookSettings(t, newPostWebhook("https://hooks.example.com")),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})
}
