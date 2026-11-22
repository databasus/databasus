package neon

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const redactedSecret = "[REDACTED]"

// secretRedactor removes the connection string and its password from every message the package
// reports. GitHub masks only the exact secret value, so the password alone, as a driver or a
// server may print it, needs its own mask.
type secretRedactor struct {
	secrets []string
}

func newSecretRedactor(connectionURL string) secretRedactor {
	secrets := []string{connectionURL}

	if parsedURL, err := url.Parse(connectionURL); err == nil && parsedURL.User != nil {
		if password, isPasswordSet := parsedURL.User.Password(); isPasswordSet && password != "" {
			secrets = append(secrets, password, url.PathEscape(password), url.QueryEscape(password))
		}
	}

	return secretRedactor{secrets: secrets}
}

func (r secretRedactor) redact(message string) string {
	for _, secret := range r.secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, redactedSecret)
		}
	}

	return message
}

func TestSecretRedactor(t *testing.T) {
	t.Parallel()

	const (
		password      = "npg_s3cr/et value"
		connectionURL = "postgresql://owner:npg_s3cr%2Fet%20value@ep-young-dew-123456.neon.tech/neondb?sslmode=require"
	)

	t.Run("removes the connection string and every spelling of its password", func(t *testing.T) {
		t.Parallel()
		redactor := newSecretRedactor(connectionURL)
		message := strings.Join([]string{
			"connect to " + connectionURL,
			"password " + password,
			"path-escaped " + url.PathEscape(password),
			"query-escaped " + url.QueryEscape(password),
		}, "; ")

		redactedMessage := redactor.redact(message)

		assert.NotContains(t, redactedMessage, connectionURL)
		assert.NotContains(t, redactedMessage, password)
		assert.NotContains(t, redactedMessage, url.PathEscape(password))
		assert.NotContains(t, redactedMessage, url.QueryEscape(password))
		assert.Contains(t, redactedMessage, redactedSecret)
	})

	t.Run("keeps the host, the user and the database name", func(t *testing.T) {
		t.Parallel()
		redactor := newSecretRedactor(connectionURL)

		redactedMessage := redactor.redact("user owner on ep-young-dew-123456.neon.tech, database neondb")

		assert.Equal(t, "user owner on ep-young-dew-123456.neon.tech, database neondb", redactedMessage)
	})

	t.Run("removes the whole string when it does not parse", func(t *testing.T) {
		t.Parallel()
		const unparsableURL = "postgresql://owner:secret@host:notaport/neondb"
		redactor := newSecretRedactor(unparsableURL)

		redactedMessage := redactor.redact(`parse "` + unparsableURL + `": invalid port`)

		assert.NotContains(t, redactedMessage, unparsableURL)
	})
}
