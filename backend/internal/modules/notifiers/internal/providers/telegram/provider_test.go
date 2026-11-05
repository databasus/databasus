package telegram_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers/providertest"
	"databasus-backend/internal/modules/notifiers/internal/providers/telegram"
)

const (
	botToken = "123456:AAH-TOKEN"
	// maxTextUTF16Units is the Bot API limit the provider cuts the text at.
	maxTextUTF16Units = 4096
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("refuses the proxy URL when the proxy is on without one", func(t *testing.T) {
		t.Parallel()

		err := newProvider(t, "").Validate(telegram.Settings{
			BotToken: "token", TargetChatID: "123456", ThreadID: nil, IsProxyEnabled: true, ProxyURL: "", BotAPIURL: "",
		})

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "proxyUrl"), "is required")
	})

	t.Run("refuses the proxy URL when its scheme is not supported", func(t *testing.T) {
		t.Parallel()

		err := newProvider(t, "").Validate(newSettingsWithProxy("ftp://proxy.example.com:3128"))

		assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "proxyUrl"), "http/https/socks5/socks5h")
	})

	t.Run("accepts a socks5 proxy URL", func(t *testing.T) {
		t.Parallel()

		err := newProvider(t, "").Validate(newSettingsWithProxy("socks5://user:password@proxy.example.com:1080"))

		require.NoError(t, err)
	})

	botAPIURLs := []struct {
		botAPIURL     string
		refusalDetail string
	}{
		{botAPIURL: ""},
		{botAPIURL: "http://localhost:8081"},
		{botAPIURL: "http://telegram-bot-api:8081"},
		{botAPIURL: "https://gateway.example.com"},
		{botAPIURL: "https://gateway.example.com/telegram/"},
		{botAPIURL: "ftp://telegram.example.com", refusalDetail: "valid http/https URL with a host"},
		{botAPIURL: "telegram-bot-api:8081", refusalDetail: "valid http/https URL with a host"},
		{botAPIURL: "https://", refusalDetail: "valid http/https URL with a host"},
		{botAPIURL: "gateway.example.com/telegram", refusalDetail: "valid http/https URL with a host"},
		{botAPIURL: "https://user:password@gateway.example.com", refusalDetail: "username or password"},
		{botAPIURL: "https://user@gateway.example.com", refusalDetail: "username or password"},
		{botAPIURL: "https://gateway.example.com/telegram?key=1", refusalDetail: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/telegram?", refusalDetail: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#bot", refusalDetail: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#", refusalDetail: "query or fragment"},
	}

	for _, botAPIURL := range botAPIURLs {
		scenario := "accepts the bot API URL " + botAPIURL.botAPIURL
		if botAPIURL.refusalDetail != "" {
			scenario = "refuses the bot API URL " + botAPIURL.botAPIURL
		}

		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			err := newProvider(t, "").Validate(telegram.Settings{
				BotToken: "token", TargetChatID: "123456", ThreadID: nil, IsProxyEnabled: false, ProxyURL: "",
				BotAPIURL: botAPIURL.botAPIURL,
			})

			if botAPIURL.refusalDetail == "" {
				require.NoError(t, err)

				return
			}

			assert.Contains(t, providertest.GetInvalidSettingDetail(t, err, "botApiUrl"), botAPIURL.refusalDetail)
		})
	}
}

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("posts to the official bot API when the notifier names none", func(t *testing.T) {
		t.Parallel()
		officialBotAPIURL, recorder := startBotAPI(t)

		err := newProvider(t, officialBotAPIURL).Send(t.Context(), newSettings(""), newBackupFailedNotification("x"))

		require.NoError(t, err)
		require.Equal(t, 1, recorder.GetRequestCount())
		assert.Equal(t, "/bot"+botToken+"/sendMessage", recorder.GetLastRequest().Path)
	})

	t.Run("cuts the text at the telegram limit by characters", func(t *testing.T) {
		t.Parallel()
		heading := "Backup failed"

		sentForm := sendAndCaptureForm(t, newBackupFailedNotification(strings.Repeat("界", maxTextUTF16Units)))

		messagePrefixLength := maxTextUTF16Units - len(heading) - len("\n\n")
		assert.Equal(t, heading+"\n\n"+strings.Repeat("界", messagePrefixLength), sentForm.Get("text"))
	})

	t.Run("counts an emoji as two units when it cuts the text", func(t *testing.T) {
		t.Parallel()
		heading := "Backup failed"

		sentForm := sendAndCaptureForm(t, newBackupFailedNotification(strings.Repeat("🔥", maxTextUTF16Units)))

		emojiThatFit := (maxTextUTF16Units - len(heading) - len("\n\n")) / 2
		assert.Equal(t, heading+"\n\n"+strings.Repeat("🔥", emojiThatFit), sentForm.Get("text"))
	})

	t.Run("sends plain text without a parse mode when the message has HTML characters", func(t *testing.T) {
		t.Parallel()
		message := "mysqldump: Got error: 1146: Table 'db.<orders>' doesn't exist & aborted"

		sentForm := sendAndCaptureForm(t, newBackupFailedNotification(message))

		assert.Equal(t, "Backup failed\n\n"+message, sentForm.Get("text"))
		assert.False(t, sentForm.Has("parse_mode"))
	})

	t.Run("goes through the proxy when the proxy is on", func(t *testing.T) {
		t.Parallel()
		proxyURL, proxyRecorder := startBotAPI(t)
		settings := newSettingsWithProxy(proxyURL)
		settings.BotToken = botToken
		settings.BotAPIURL = "http://telegram-bot-api.invalid"

		err := newProvider(t, "").Send(t.Context(), settings, newBackupFailedNotification("x"))

		require.NoError(t, err)
		require.Equal(t, 1, proxyRecorder.GetRequestCount())
		assert.Equal(t, "telegram-bot-api.invalid", proxyRecorder.GetLastRequest().Host)
	})

	t.Run("reports the API's answer without the bot token when it refuses the message", func(t *testing.T) {
		t.Parallel()
		botAPIURL, _ := providertest.StartRecordingServer(t, providertest.StubResponse{
			StatusCode: http.StatusBadRequest, Body: `{"description":"bad token ` + botToken + `"}`,
		})

		err := newProvider(t, "").Send(t.Context(), newSettings(botAPIURL), newBackupFailedNotification("x"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.NotContains(t, err.Error(), botToken)
	})
}

func newProvider(t *testing.T, officialBotAPIURL string) *telegram.Provider {
	t.Helper()

	return telegram.New(telegram.Deps{
		Repositories:      telegram.Repositories{},
		FieldEncryptor:    providertest.NewFieldEncryptor(t),
		OfficialBotAPIURL: officialBotAPIURL,
	})
}

func newSettings(botAPIURL string) telegram.Settings {
	return telegram.Settings{
		BotToken: botToken, TargetChatID: "123456", ThreadID: nil, IsProxyEnabled: false, ProxyURL: "",
		BotAPIURL: botAPIURL,
	}
}

func newSettingsWithProxy(proxyURL string) telegram.Settings {
	return telegram.Settings{
		BotToken: "token", TargetChatID: "123456", ThreadID: nil, IsProxyEnabled: true, ProxyURL: proxyURL,
		BotAPIURL: "",
	}
}

func newBackupFailedNotification(message string) notification.Notification {
	return notification.Notification{
		Category: notification.CategoryBackupFailed,
		Heading:  "Backup failed",
		Message:  message,
	}
}

func startBotAPI(t *testing.T) (string, *providertest.RequestRecorder) {
	t.Helper()

	return providertest.StartRecordingServer(
		t,
		providertest.StubResponse{StatusCode: http.StatusOK, Body: `{"ok":true}`},
	)
}

func sendAndCaptureForm(t *testing.T, message notification.Notification) url.Values {
	t.Helper()
	botAPIURL, recorder := startBotAPI(t)

	require.NoError(t, newProvider(t, "").Send(t.Context(), newSettings(botAPIURL), message))
	require.Equal(t, 1, recorder.GetRequestCount())

	sentForm, err := url.ParseQuery(recorder.GetLastRequest().Body)
	require.NoError(t, err)

	return sentForm
}
