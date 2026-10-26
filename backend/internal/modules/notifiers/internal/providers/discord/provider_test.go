package discord_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers/discord"
	"databasus-backend/internal/modules/notifiers/internal/providers/providertest"
)

// maxContentRunes is the Discord limit the provider cuts the content at.
const maxContentRunes = 2000

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("cuts the content at the discord limit by characters", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := providertest.StartRecordingServer(t,
			providertest.StubResponse{StatusCode: http.StatusNoContent, Body: ""})
		provider := discord.New(
			discord.Deps{Repositories: discord.Repositories{}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
		)
		heading := "Backup failed"

		err := provider.Send(t.Context(), discord.Settings{ChannelWebhookURL: webhookURL}, notification.Notification{
			Category: notification.CategoryBackupFailed,
			Heading:  heading,
			Message:  strings.Repeat("界", maxContentRunes),
		})

		require.NoError(t, err)
		require.Equal(t, 1, recorder.GetRequestCount())
		request := recorder.GetLastRequest()
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))

		var payload map[string]string
		require.NoError(t, json.Unmarshal([]byte(request.Body), &payload))
		messagePrefixLength := maxContentRunes - len([]rune(heading)) - len([]rune("\n\n"))
		assert.Equal(t, heading+"\n\n"+strings.Repeat("界", messagePrefixLength), payload["content"])
	})
}
