package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestNotifiersAfterRecovery(t *testing.T) {
	t.Parallel()

	t.Run("delivers an automatic notification without any confirmation after a recovery", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithPublishedRecovery())
		webhook := startEndpoint(t, http.StatusOK)
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhook.URL)))

		tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())
		tm.app.WaitForNotificationDeliveries(t)

		assert.Len(t, webhook.getRequests(), 1)
	})
}
