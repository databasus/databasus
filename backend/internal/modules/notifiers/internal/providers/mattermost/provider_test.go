package mattermost_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/mattermost"
	"databasus-backend/internal/modules/notifiers/internal/providers/providertest"
)

const (
	channelIDLength = 26
	maxPostLength   = 16383
	// maxErrorBodyBytes is the part of a failed response body the provider reports.
	maxErrorBodyBytes = 8 * 1024
	createPostPath    = "/api/v4/posts"
)

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("posts the markdown text to the hook URL in the webhook mode", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startMattermost(t, http.StatusOK, "ok")

		require.NoError(t, send(t, newWebhookSettings(webhookURL), newBackupFailedNotification()))

		require.Equal(t, 1, recorder.GetRequestCount())
		request := recorder.GetLastRequest()
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))
		assert.Empty(t, request.Headers.Get("Authorization"))

		body := decodeBody(t, request.Body)
		assert.Equal(t, "**Backup failed**\n\nmydb: connection refused", body["text"])
		assert.NotContains(t, body, "channel")
		assert.NotContains(t, body, "username")
		assert.NotContains(t, body, "icon_url")
	})

	t.Run("posts the channel and the identity in the webhook mode with overrides", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startMattermost(t, http.StatusOK, "ok")
		settings := newWebhookSettings(webhookURL)
		settings.TargetChannelName = "town-square"
		settings.OverrideUsername = "Databasus"
		settings.OverrideIconURL = "https://databasus.com/icon.png"

		require.NoError(t, send(t, settings, newBackupFailedNotification()))

		body := decodeBody(t, recorder.GetLastRequest().Body)
		assert.Equal(t, "town-square", body["channel"])
		assert.Equal(t, "Databasus", body["username"])
		assert.Equal(t, "https://databasus.com/icon.png", body["icon_url"])
	})

	t.Run("posts the bold heading without trailing newlines when there is no message", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startMattermost(t, http.StatusOK, "ok")

		require.NoError(t, send(t, newWebhookSettings(webhookURL), notification.Notification{
			Category: notification.CategoryBackupSuccess,
			Heading:  "Backup completed",
			Message:  "",
		}))

		assert.Equal(t, "**Backup completed**", decodeBody(t, recorder.GetLastRequest().Body)["text"])
	})

	t.Run("cuts the post at the mattermost limit", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startMattermost(t, http.StatusOK, "ok")

		require.NoError(t, send(t, newWebhookSettings(webhookURL), notification.Notification{
			Category: notification.CategoryBackupFailed,
			Heading:  "Backup failed",
			Message:  strings.Repeat("é", maxPostLength),
		}))

		postText, isText := decodeBody(t, recorder.GetLastRequest().Body)["text"].(string)
		require.True(t, isText)
		assert.Len(t, []rune(postText), maxPostLength)
	})

	t.Run("posts to the create post endpoint with the bearer token in the bot mode", func(t *testing.T) {
		t.Parallel()
		serverURL, recorder := startMattermost(t, http.StatusCreated, `{"id":"post123"}`)

		require.NoError(t, send(t, newBotSettings(serverURL+"/"), newBackupFailedNotification()))

		request := recorder.GetLastRequest()
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, createPostPath, request.Path)
		assert.Equal(t, "Bearer bot-token", request.Headers.Get("Authorization"))

		body := decodeBody(t, request.Body)
		assert.Equal(t, strings.Repeat("c", channelIDLength), body["channel_id"])
		assert.Equal(t, "**Backup failed**\n\nmydb: connection refused", body["message"])
		assert.NotContains(t, body, "props")
	})

	t.Run("posts the identity in the props in the bot mode with overrides", func(t *testing.T) {
		t.Parallel()
		serverURL, recorder := startMattermost(t, http.StatusCreated, `{"id":"post123"}`)
		settings := newBotSettings(serverURL)
		settings.OverrideUsername = "Databasus"
		settings.OverrideIconURL = "https://databasus.com/icon.png"

		require.NoError(t, send(t, settings, newBackupFailedNotification()))

		props, isObject := decodeBody(t, recorder.GetLastRequest().Body)["props"].(map[string]any)
		require.True(t, isObject)
		assert.Equal(t, "Databasus", props["override_username"])
		assert.Equal(t, "https://databasus.com/icon.png", props["override_icon_url"])
	})

	t.Run("reports the body when the server answers with an error status", func(t *testing.T) {
		t.Parallel()
		webhookURL, _ := startMattermost(t, http.StatusInternalServerError, "boom")

		err := send(t, newWebhookSettings(webhookURL), newBackupFailedNotification())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("reports the message of a mattermost error body", func(t *testing.T) {
		t.Parallel()
		serverURL, _ := startMattermost(t, http.StatusForbidden,
			`{"id":"api.context.permissions.app_error","message":"You do not have the appropriate permissions."}`)

		err := send(t, newBotSettings(serverURL), newBackupFailedNotification())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "You do not have the appropriate permissions.")
	})

	t.Run("reports the error without the webhook path the server echoes", func(t *testing.T) {
		t.Parallel()
		serverURL, _ := startMattermost(t, http.StatusInternalServerError, "no hook at /hooks/secret-key")

		err := send(t, newWebhookSettings(serverURL+"/hooks/secret-key"), newBackupFailedNotification())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no hook at redacted")
		assert.NotContains(t, err.Error(), "secret-key")
	})

	t.Run("reports the error without the bot token the server echoes", func(t *testing.T) {
		t.Parallel()
		serverURL, _ := startMattermost(t, http.StatusUnauthorized,
			`{"message":"token Bearer bot-token is not valid"}`)

		err := send(t, newBotSettings(serverURL), newBackupFailedNotification())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not valid")
		assert.NotContains(t, err.Error(), "bot-token")
	})

	t.Run("reports no webhook URL when the server cannot be reached", func(t *testing.T) {
		t.Parallel()
		webhookURL, _ := startMattermost(t, http.StatusOK, "")

		err := send(t, newWebhookSettings(webhookURL+"-gone/hooks/secret-key"), newBackupFailedNotification())

		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret-key")
	})

	t.Run("keeps the server URL in the error when the bot mode server cannot be reached", func(t *testing.T) {
		t.Parallel()
		serverURL, _ := startMattermost(t, http.StatusOK, "")

		err := send(t, newBotSettings(serverURL+"-gone"), newBackupFailedNotification())

		require.Error(t, err)
		assert.Contains(t, err.Error(), serverURL+"-gone")
	})

	t.Run("cuts an error body longer than the read limit", func(t *testing.T) {
		t.Parallel()
		serverURL, _ := startMattermost(t, http.StatusInternalServerError, strings.Repeat("x", 10*maxErrorBodyBytes))

		err := send(t, newWebhookSettings(serverURL), newBackupFailedNotification())

		require.Error(t, err)
		assert.NotContains(t, err.Error(), strings.Repeat("x", maxErrorBodyBytes+1))
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("refuses an unknown delivery mode", func(t *testing.T) {
		t.Parallel()
		settings := newWebhookSettings("https://mm.example.com/hooks/key")
		settings.DeliveryMode = "CARRIER_PIGEON"

		err := newProvider(t).Validate(settings)

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "deliveryMode"), "WEBHOOK or BOT")
	})

	t.Run("accepts the webhook mode with an http URL", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, newProvider(t).Validate(newWebhookSettings("https://mm.example.com/hooks/abc123")))
	})

	t.Run("refuses the webhook mode without a URL", func(t *testing.T) {
		t.Parallel()

		err := newProvider(t).Validate(newWebhookSettings(""))

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "webhookUrl"), "is required")
	})

	t.Run("refuses the webhook mode with a URL that is not http", func(t *testing.T) {
		t.Parallel()

		err := newProvider(t).Validate(newWebhookSettings("mm.example.com/hooks/abc"))

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "webhookUrl"), "http or https")
	})

	t.Run("accepts the bot mode with every credential", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, newProvider(t).Validate(newBotSettings("https://mm.example.com")))
	})

	t.Run("refuses the bot mode without a token", func(t *testing.T) {
		t.Parallel()
		settings := newBotSettings("https://mm.example.com")
		settings.BotToken = ""

		err := newProvider(t).Validate(settings)

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "botToken"), "is required")
	})

	t.Run("refuses a channel name in place of the channel ID", func(t *testing.T) {
		t.Parallel()
		settings := newBotSettings("https://mm.example.com")
		settings.TargetChannelID = "town-square"

		err := newProvider(t).Validate(settings)

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "targetChannelId"), "not the channel name")
	})

	t.Run("refuses an override icon URL that is not http", func(t *testing.T) {
		t.Parallel()
		settings := newWebhookSettings("https://mm.example.com/hooks/abc123")
		settings.OverrideIconURL = "javascript:alert(1)"

		err := newProvider(t).Validate(settings)

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "overrideIconUrl"), "http or https")
	})
}

func TestMergeStoredSecrets(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stored credentials when the incoming settings leave them out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newBotSettings("https://mm.example.com")
		storedSettings.BotToken = "stored-token"
		storedSettings.WebhookURL = "https://mm.example.com/hooks/stored"
		incomingSettings := newBotSettings("https://mm.example.com")
		incomingSettings.BotToken = ""
		incomingSettings.TargetChannelID = strings.Repeat("d", channelIDLength)
		incomingSettings.OverrideUsername = "Databasus"

		mergedSettings, err := newProvider(t).MergeStoredSecrets(providers.SecretMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		mergedMattermostSettings, err := providers.CastSettings[mattermost.Settings](mergedSettings)
		require.NoError(t, err)
		assert.Equal(t, "stored-token", mergedMattermostSettings.BotToken)
		assert.Equal(t, "https://mm.example.com/hooks/stored", mergedMattermostSettings.WebhookURL)
		assert.Equal(t, "https://mm.example.com", mergedMattermostSettings.ServerURL)
		assert.Equal(t, strings.Repeat("d", channelIDLength), mergedMattermostSettings.TargetChannelID)
		assert.Equal(t, "Databasus", mergedMattermostSettings.OverrideUsername)
	})

	t.Run("keeps the webhook url and leaves the bot token empty when the server url changes", func(t *testing.T) {
		t.Parallel()
		storedSettings := newBotSettings("https://mm.example.com")
		storedSettings.BotToken = "stored-token"
		storedSettings.WebhookURL = "https://mm.example.com/hooks/stored"
		incomingSettings := newBotSettings("https://moved.example.com")
		incomingSettings.BotToken = ""

		mergedSettings, err := newProvider(t).MergeStoredSecrets(providers.SecretMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		mergedMattermostSettings, err := providers.CastSettings[mattermost.Settings](mergedSettings)
		require.NoError(t, err)
		assert.Empty(t, mergedMattermostSettings.BotToken)
		assert.Equal(t, "https://mm.example.com/hooks/stored", mergedMattermostSettings.WebhookURL)
	})
}

func newProvider(t *testing.T) *mattermost.Provider {
	t.Helper()

	return mattermost.New(
		mattermost.Deps{Repositories: mattermost.Repositories{}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
	)
}

func send(t *testing.T, settings mattermost.Settings, message notification.Notification) error {
	t.Helper()

	return newProvider(t).Send(t.Context(), settings, message)
}

func newWebhookSettings(webhookURL string) mattermost.Settings {
	return mattermost.Settings{
		DeliveryMode:         mattermost.DeliveryModeWebhook,
		WebhookURL:           webhookURL,
		ServerURL:            "",
		BotToken:             "",
		TargetChannelName:    "",
		TargetChannelID:      "",
		OverrideUsername:     "",
		OverrideIconURL:      "",
		IsInsecureSkipVerify: false,
	}
}

func newBotSettings(serverURL string) mattermost.Settings {
	return mattermost.Settings{
		DeliveryMode:         mattermost.DeliveryModeBot,
		WebhookURL:           "",
		ServerURL:            serverURL,
		BotToken:             "bot-token",
		TargetChannelName:    "",
		TargetChannelID:      strings.Repeat("c", channelIDLength),
		OverrideUsername:     "",
		OverrideIconURL:      "",
		IsInsecureSkipVerify: false,
	}
}

func newBackupFailedNotification() notification.Notification {
	return notification.Notification{
		Category: notification.CategoryBackupFailed,
		Heading:  "Backup failed",
		Message:  "mydb: connection refused",
	}
}

func startMattermost(t *testing.T, statusCode int, body string) (string, *providertest.RequestRecorder) {
	t.Helper()

	return providertest.StartRecordingServer(t, providertest.StubResponse{StatusCode: statusCode, Body: body})
}

func decodeBody(t *testing.T, body string) map[string]any {
	t.Helper()

	var decodedBody map[string]any

	require.NoError(t, json.Unmarshal([]byte(body), &decodedBody))

	return decodedBody
}
