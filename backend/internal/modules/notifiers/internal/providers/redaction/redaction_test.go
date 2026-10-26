package redaction_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

func TestStripURLCredentials(t *testing.T) {
	t.Parallel()

	tokenBearingURLs := []struct {
		name         string
		rawURL       string
		secret       string
		sanitizedURL string
	}{
		{
			name:         "removes the token telegram fuses into the bot segment",
			rawURL:       "https://api.telegram.org/bot123456:AAH-REALTOKEN/sendMessage",
			secret:       "AAH-REALTOKEN",
			sanitizedURL: "https://api.telegram.org/botredacted/redacted",
		},
		{
			name:         "removes the token discord carries as a trailing path segment",
			rawURL:       "https://discord.com/api/webhooks/999/SECRETWEBHOOKTOKEN",
			secret:       "SECRETWEBHOOKTOKEN",
			sanitizedURL: "https://discord.com/api/redacted/redacted/redacted",
		},
		{
			name:         "removes the hook of a mattermost incoming webhook",
			rawURL:       "https://chat.example.com/hooks/xxxSECRETHOOKxxx",
			secret:       "xxxSECRETHOOKxxx",
			sanitizedURL: "https://chat.example.com/hooks/redacted",
		},
		{
			name:         "removes a token from the query string",
			rawURL:       "https://hooks.example.com/notify?token=SECRETQUERYTOKEN",
			secret:       "SECRETQUERYTOKEN",
			sanitizedURL: "https://hooks.example.com/notify",
		},
		{
			name:         "removes the password of the user info",
			rawURL:       "https://user:SECRETPASSWORD@hooks.example.com/notify",
			secret:       "SECRETPASSWORD",
			sanitizedURL: "https://redacted@hooks.example.com/notify",
		},
	}

	for _, tokenBearingURL := range tokenBearingURLs {
		t.Run(tokenBearingURL.name, func(t *testing.T) {
			t.Parallel()

			sanitizedURL := redaction.StripURLCredentials(tokenBearingURL.rawURL)

			assert.NotContains(t, sanitizedURL, tokenBearingURL.secret)
			assert.Equal(t, tokenBearingURL.sanitizedURL, sanitizedURL)
		})
	}

	t.Run("returns the redacted marker when the URL does not parse", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "redacted", redaction.StripURLCredentials("://not a url\x7f"))
	})

	t.Run("keeps the host readable", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, redaction.StripURLCredentials("https://api.telegram.org/bot1:X/sendMessage"),
			"api.telegram.org")
	})
}

func TestErrorWithoutURLCredentials(t *testing.T) {
	t.Parallel()

	t.Run("keeps no bot token when telegram's transport fails", func(t *testing.T) {
		t.Parallel()
		transportError := &url.Error{
			Op:  "Post",
			URL: "https://api.telegram.org/bot123456:AAH-REALBOTTOKEN/sendMessage",
			Err: errors.New("dial tcp: i/o timeout"),
		}

		sanitizedError := redaction.ErrorWithoutURLCredentials(transportError)

		assert.NotContains(t, sanitizedError.Error(), "AAH-REALBOTTOKEN")
		assert.Contains(t, sanitizedError.Error(), "api.telegram.org")
		assert.Contains(t, sanitizedError.Error(), "i/o timeout")
	})

	t.Run("keeps no webhook token when a webhook's transport fails", func(t *testing.T) {
		t.Parallel()
		transportError := &url.Error{
			Op:  "Post",
			URL: "https://discord.com/api/webhooks/999/SECRETWEBHOOKTOKEN",
			Err: errors.New("connection refused"),
		}

		sanitizedError := redaction.ErrorWithoutURLCredentials(transportError)

		assert.NotContains(t, sanitizedError.Error(), "SECRETWEBHOOKTOKEN")
		assert.Contains(t, sanitizedError.Error(), "discord.com")
		assert.Contains(t, sanitizedError.Error(), "connection refused")
	})

	t.Run("returns the error unchanged when it is not a transport error", func(t *testing.T) {
		t.Parallel()
		plainError := errors.New("failed to decrypt bot token")

		assert.Equal(t, plainError, redaction.ErrorWithoutURLCredentials(plainError))
	})
}

func TestRedactSecretsInText(t *testing.T) {
	t.Parallel()

	t.Run("replaces every occurrence of the echoed secret", func(t *testing.T) {
		t.Parallel()

		redactedText := redaction.RedactSecretsInText(
			"Cannot POST /bot123456:AAH-SECRET/sendMessage (123456:AAH-SECRET)",
			"123456:AAH-SECRET",
		)

		assert.Equal(t, "Cannot POST /botredacted/sendMessage (redacted)", redactedText)
	})

	t.Run("replaces each of several secrets", func(t *testing.T) {
		t.Parallel()

		redactedText := redaction.RedactSecretsInText("echo: Bearer first, X-Key second", "Bearer first", "second")

		assert.Equal(t, "echo: redacted, X-Key redacted", redactedText)
	})

	t.Run("returns the text unchanged when the secret is empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "chat not found", redaction.RedactSecretsInText("chat not found", ""))
	})
}

func TestGetSecretURLForms(t *testing.T) {
	t.Parallel()

	t.Run("returns the URL and its path", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			[]string{"https://discord.example.com/api/webhooks/1/token", "/api/webhooks/1/token"},
			redaction.GetSecretURLForms("https://discord.example.com/api/webhooks/1/token"))
	})

	t.Run("returns only the URL when it has no path", func(t *testing.T) {
		t.Parallel()

		assert.Equal(
			t,
			[]string{"https://discord.example.com"},
			redaction.GetSecretURLForms("https://discord.example.com"),
		)
	})
}
