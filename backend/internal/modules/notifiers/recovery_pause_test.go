package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestNotifiersDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("refuses saved and unsaved test sends with recovery_paused and sends nothing", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		webhook := startEndpoint(t, http.StatusOK)
		settings := newWebhookSettings(t, newPostWebhook(webhook.URL))
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, settings)

		savedResponse, err := tm.owner.Client.TestNotifierWithResponse(t.Context(), webhookNotifier.Id)
		require.NoError(t, err)
		settingsResponse, err := tm.owner.Client.TestNotifierSettingsWithResponse(t.Context(),
			apiclient.NotifierSettingsTest{WorkspaceId: tm.workspace.Id, NotifierId: nil, Settings: settings})
		require.NoError(t, err)

		requireProblem(t, savedResponse.StatusCode(), savedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)
		requireProblem(t, settingsResponse.StatusCode(), settingsResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)
		assert.Empty(t, webhook.getRequests())
	})

	t.Run("drops automatic notifications while paused", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		webhook := startEndpoint(t, http.StatusOK)
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhook.URL)))

		tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())
		tm.app.WaitForNotificationDeliveries(t)

		assert.Empty(t, webhook.getRequests())
	})
}
