package webhook_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/providertest"
	"databasus-backend/internal/modules/notifiers/internal/providers/webhook"
)

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("posts the default JSON body without a body template", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)

		require.NoError(t, send(t, newSettings(webhookURL, webhook.MethodPOST), notification.CategoryBackupSuccess))

		require.Equal(t, 1, recorder.GetRequestCount())
		request := recorder.GetLastRequest()
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))

		var payload map[string]string
		require.NoError(t, json.Unmarshal([]byte(request.Body), &payload))
		assert.Equal(t, "Backup completed", payload["heading"])
		assert.Equal(t, "All good", payload["message"])
	})

	t.Run("fills and escapes the placeholders of the body template", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.BodyTemplate = `{"title":"{{heading}}","text":"{{message}}"}`

		err := newProvider(t).Send(t.Context(), settings, notification.Notification{
			Category: notification.CategoryBackupSuccess,
			Heading:  `He said "hi"`,
			Message:  "line1\nline2",
		})
		require.NoError(t, err)

		var payload map[string]string
		require.NoError(t, json.Unmarshal([]byte(recorder.GetLastRequest().Body), &payload))
		assert.Equal(t, `He said "hi"`, payload["title"])
		assert.Equal(t, "line1\nline2", payload["text"])
	})

	t.Run("keeps a content type the headers set", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.Headers = []webhook.Header{{Name: "Content-Type", Value: "application/xml"}}

		require.NoError(t, send(t, settings, notification.CategoryBackupSuccess))

		assert.Equal(t, "application/xml", recorder.GetLastRequest().Headers.Get("Content-Type"))
	})

	t.Run("sends the heading and the message in the query with GET", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)

		require.NoError(t, send(t, newSettings(webhookURL, webhook.MethodGET), notification.CategoryBackupSuccess))

		request := recorder.GetLastRequest()
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "Backup completed", request.Query.Get("heading"))
		assert.Equal(t, "All good", request.Query.Get("message"))
		assert.Empty(t, request.Body)
	})

	t.Run("sends the decrypted value of an encrypted header", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		fieldEncryptor := providertest.NewFieldEncryptor(t)
		encryptedValue, err := fieldEncryptor.Encrypt("secret-value")
		require.NoError(t, err)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.Headers = []webhook.Header{{Name: "X-Api-Key", Value: encryptedValue}}

		err = webhook.New(webhook.Deps{Repositories: webhook.Repositories{}, FieldEncryptor: fieldEncryptor}).
			Send(t.Context(), settings, newNotification(notification.CategoryBackupSuccess))

		require.NoError(t, err)
		assert.Equal(t, "secret-value", recorder.GetLastRequest().Headers.Get("X-Api-Key"))
	})

	t.Run("fails naming the header and sends nothing when its stored value cannot be decrypted", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.Headers = []webhook.Header{{Name: "Authorization", Value: "enc:not-a-valid-ciphertext"}}

		err := send(t, settings, notification.CategoryBackupFailed)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to decrypt webhook header "Authorization"`)
		assert.Zero(t, recorder.GetRequestCount())
	})

	t.Run("fails when the endpoint answers with an error status", func(t *testing.T) {
		t.Parallel()
		webhookURL, _ := startEndpoint(t, http.StatusInternalServerError)

		err := send(t, newSettings(webhookURL, webhook.MethodPOST), notification.CategoryBackupSuccess)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("fails when the endpoint cannot be reached", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		unreachableURL := server.URL
		server.Close()

		require.Error(t, send(t, newSettings(unreachableURL, webhook.MethodPOST), notification.CategoryBackupSuccess))
	})

	t.Run("sends every category when the accepted list is empty", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.AcceptedCategories = nil

		require.NoError(t, send(t, settings, notification.CategoryBackupSuccess))
		require.NoError(t, send(t, settings, notification.CategoryHealthcheckFailed))

		assert.Equal(t, 2, recorder.GetRequestCount())
	})

	t.Run("sends every category when the accepted list holds ALL", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)

		require.NoError(t, send(t, settings, notification.CategoryBackupSuccess))
		require.NoError(t, send(t, settings, notification.CategoryVerificationFailed))

		assert.Equal(t, 2, recorder.GetRequestCount())
	})

	t.Run("skips the categories outside a narrowed accepted list", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.AcceptedCategories = []notification.Category{notification.CategoryBackupSuccess}

		require.NoError(t, send(t, settings, notification.CategoryBackupFailed))
		require.NoError(t, send(t, settings, notification.CategoryHealthcheckSuccess))
		assert.Zero(t, recorder.GetRequestCount())

		require.NoError(t, send(t, settings, notification.CategoryBackupSuccess))
		assert.Equal(t, 1, recorder.GetRequestCount())
	})

	t.Run("sends a test notification when the accepted list is narrowed", func(t *testing.T) {
		t.Parallel()
		webhookURL, recorder := startEndpoint(t, http.StatusOK)
		settings := newSettings(webhookURL, webhook.MethodPOST)
		settings.AcceptedCategories = []notification.Category{notification.CategoryBackupFailed}

		require.NoError(t, send(t, settings, notification.CategoryAll))

		assert.Equal(t, 1, recorder.GetRequestCount())
	})
}

func TestMergeStoredSecrets(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stored value of a header sent without one and drops a header left out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings("https://hooks.example.com/saved", webhook.MethodPOST)
		storedSettings.Headers = []webhook.Header{
			{Name: "Authorization", Value: "enc:stored-authorization"},
			{Name: "X-Team", Value: "enc:stored-team"},
		}
		incomingSettings := newSettings("https://hooks.example.com/saved", webhook.MethodPOST)
		incomingSettings.Headers = []webhook.Header{
			{Name: "authorization", Value: ""},
			{Name: "X-Region", Value: "eu"},
		}

		mergedSettings, err := newProvider(t).MergeStoredSecrets(providers.SecretMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		mergedWebhookSettings, err := providers.CastSettings[webhook.Settings](mergedSettings)
		require.NoError(t, err)
		assert.Equal(t, "https://hooks.example.com/saved", mergedWebhookSettings.URL)
		assert.Equal(t, []webhook.Header{
			{Name: "authorization", Value: "enc:stored-authorization"},
			{Name: "X-Region", Value: "eu"},
		}, mergedWebhookSettings.Headers)
	})
}

func newProvider(t *testing.T) *webhook.Provider {
	t.Helper()

	return webhook.New(
		webhook.Deps{Repositories: webhook.Repositories{}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
	)
}

func send(t *testing.T, settings webhook.Settings, category notification.Category) error {
	t.Helper()

	return newProvider(t).Send(t.Context(), settings, newNotification(category))
}

func newSettings(webhookURL string, method webhook.Method) webhook.Settings {
	return webhook.Settings{
		URL:                webhookURL,
		Method:             method,
		BodyTemplate:       "",
		Headers:            nil,
		AcceptedCategories: []notification.Category{notification.CategoryAll},
	}
}

func newNotification(category notification.Category) notification.Notification {
	return notification.Notification{Category: category, Heading: "Backup completed", Message: "All good"}
}

func startEndpoint(t *testing.T, statusCode int) (string, *providertest.RequestRecorder) {
	t.Helper()

	return providertest.StartRecordingServer(t, providertest.StubResponse{StatusCode: statusCode, Body: ""})
}
