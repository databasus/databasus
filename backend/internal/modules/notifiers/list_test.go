package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestListNotifiers(t *testing.T) {
	t.Parallel()

	t.Run("pages through the notifiers of the workspace by name", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		for _, name := range []string{"b", "a", "c"} {
			response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        name,
				Settings:    newWebhookSettings(t, newPostWebhook("https://hooks.example.com/"+name)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		}

		pageLimit := 2
		firstPage, err := tm.owner.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       &pageLimit,
			After:       nil,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, firstPage.StatusCode(), string(firstPage.Body))
		require.NotNil(t, firstPage.JSON200.NextCursor)

		secondPage, err := tm.owner.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       &pageLimit,
			After:       firstPage.JSON200.NextCursor,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, secondPage.StatusCode(), string(secondPage.Body))

		assert.Equal(t, []string{"a", "b"}, getNames(firstPage.JSON200.Items))
		assert.Equal(t, []string{"c"}, getNames(secondPage.JSON200.Items))
		assert.Nil(t, secondPage.JSON200.NextCursor)
	})

	t.Run("returns 400 invalid_request when the cursor is not one of the list", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		malformedCursor := "not-a-cursor"

		response, err := tm.owner.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       nil,
			After:       &malformedCursor,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func getNames(listedNotifiers []apiclient.Notifier) []string {
	names := make([]string, 0, len(listedNotifiers))
	for _, listedNotifier := range listedNotifiers {
		names = append(names, listedNotifier.Name)
	}

	return names
}
