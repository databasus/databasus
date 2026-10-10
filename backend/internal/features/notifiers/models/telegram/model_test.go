package telegram_notifier

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifier_models "databasus-backend/internal/features/notifiers/models"
)

func Test_Validate_WhenProxyEnabledWithoutURL_ReturnsError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "proxy URL is required")
}

func Test_Validate_WhenProxyURLSchemeUnsupported_ReturnsError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
		ProxyURL:       "ftp://proxy.example.com:3128",
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "http/https/socks5/socks5h")
}

func Test_Validate_WhenProxyURLIsSocks5_ReturnsNoError(t *testing.T) {
	notifier := &TelegramNotifier{
		BotToken:       "token",
		TargetChatID:   "123456",
		IsProxyEnabled: true,
		ProxyURL:       "socks5://user:password@proxy.example.com:1080",
	}

	err := notifier.Validate(notifier_models.PassthroughEncryptor{})

	require.NoError(t, err)
}

func Test_Validate_WithBotAPIURL_AcceptsPlainHTTPAddressesOnly(t *testing.T) {
	testCases := []struct {
		botAPIURL     string
		expectedError string
	}{
		{botAPIURL: ""},
		{botAPIURL: "http://localhost:8081"},
		{botAPIURL: "http://telegram-bot-api:8081"},
		{botAPIURL: "https://gateway.example.com"},
		{botAPIURL: "https://gateway.example.com/telegram/"},
		{botAPIURL: "ftp://telegram.example.com", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "telegram-bot-api:8081", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "https://", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "gateway.example.com/telegram", expectedError: "valid http/https URL with a host"},
		{botAPIURL: "https://user:password@gateway.example.com", expectedError: "username or password"},
		{botAPIURL: "https://user@gateway.example.com", expectedError: "username or password"},
		{botAPIURL: "https://gateway.example.com/telegram?key=1", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/telegram?", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#bot", expectedError: "query or fragment"},
		{botAPIURL: "https://gateway.example.com/#", expectedError: "query or fragment"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.botAPIURL, func(t *testing.T) {
			notifier := &TelegramNotifier{
				BotToken:     "token",
				TargetChatID: "123456",
				BotAPIURL:    testCase.botAPIURL,
			}

			err := notifier.Validate(notifier_models.PassthroughEncryptor{})

			if testCase.expectedError == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func Test_GetSendMessageURL_WithoutBotAPIURL_UsesOfficialTelegramAPI(t *testing.T) {
	notifier := &TelegramNotifier{}

	sendMessageURL, err := notifier.getSendMessageURL("123456:AAH-TOKEN")

	require.NoError(t, err)
	assert.Equal(t, "https://api.telegram.org/bot123456:AAH-TOKEN/sendMessage", sendMessageURL)
}

func Test_Send_WhenTextExceedsTelegramLimit_TruncatesTextByRunes(t *testing.T) {
	botAPIURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK, Body: `{"ok":true}`},
	)
	notifier := &TelegramNotifier{
		BotToken:     "123456:AAH-TOKEN",
		TargetChatID: "123456",
		BotAPIURL:    botAPIURL,
	}
	heading := "Backup failed"

	require.NoError(t, notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifier_models.Notification{
			Type:    notifier_models.NotificationTypeBackupFailed,
			Heading: heading,
			Message: strings.Repeat("界", maxTextRunes),
		},
	))

	require.Equal(t, 1, recorder.GetRequestCount())
	sentForm, err := url.ParseQuery(recorder.GetLastRequest().Body)
	require.NoError(t, err)

	messagePrefixLength := maxTextRunes - len([]rune(heading)) - len([]rune("\n\n"))
	assert.Equal(
		t,
		heading+"\n\n"+strings.Repeat("界", messagePrefixLength),
		sentForm.Get("text"),
	)
}
