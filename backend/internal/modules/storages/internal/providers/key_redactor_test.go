package providers_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const redactedKey = "[REDACTED]"

// keyRedactor removes the access key and the secret key from every message the object storage
// scenario reports. GitHub masks only the exact secret values in CI, so local runs and the escaped
// spellings a client library may print need masking of their own.
type keyRedactor struct {
	keys []string
}

func newKeyRedactor(accessKey, secretKey string) keyRedactor {
	var keys []string

	for _, key := range []string{secretKey, accessKey} {
		if key != "" {
			keys = append(keys, key, url.PathEscape(key), url.QueryEscape(key))
		}
	}

	return keyRedactor{keys: keys}
}

func (r keyRedactor) redact(message string) string {
	for _, key := range r.keys {
		message = strings.ReplaceAll(message, key, redactedKey)
	}

	return message
}

func TestKeyRedactor(t *testing.T) {
	t.Parallel()

	const (
		accessKey = "nak_live_access"
		secretKey = "nsk_live_se/cret+key"
	)

	t.Run("removes both keys in every spelling", func(t *testing.T) {
		t.Parallel()
		redactor := newKeyRedactor(accessKey, secretKey)
		message := strings.Join([]string{
			"access " + accessKey,
			"secret " + secretKey,
			"path-escaped " + url.PathEscape(secretKey),
			"query-escaped " + url.QueryEscape(secretKey),
		}, "; ")

		redactedMessage := redactor.redact(message)

		assert.NotContains(t, redactedMessage, accessKey)
		assert.NotContains(t, redactedMessage, secretKey)
		assert.NotContains(t, redactedMessage, url.PathEscape(secretKey))
		assert.NotContains(t, redactedMessage, url.QueryEscape(secretKey))
	})

	t.Run("keeps the endpoint and the bucket", func(t *testing.T) {
		t.Parallel()
		redactor := newKeyRedactor(accessKey, secretKey)

		redactedMessage := redactor.redact("bucket databasus-test at https://br-example.storage.neon.tech")

		assert.Equal(t, "bucket databasus-test at https://br-example.storage.neon.tech", redactedMessage)
	})
}
