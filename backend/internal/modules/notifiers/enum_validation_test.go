package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestNotifierEnumValidation(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 and creates nothing with an unsupported webhook method", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		webhook := newPostWebhook("https://hooks.example.com/backups")
		webhook.WebhookMethod = "PATCH"

		response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Invalid webhook",
			Settings:    newWebhookSettings(t, webhook),
		})

		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})
}
