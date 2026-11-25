package totp_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/totp"
)

func TestBuildAddress(t *testing.T) {
	t.Parallel()

	t.Run("names the issuer, the account and the code parameters", func(t *testing.T) {
		t.Parallel()

		address := totp.BuildAddress("Databasus", "owner+ops@example.com", rfcSecret)

		parsedAddress, err := url.Parse(address)
		require.NoError(t, err)
		assert.Equal(t, "otpauth", parsedAddress.Scheme)
		assert.Equal(t, "totp", parsedAddress.Host)
		assert.Equal(t, "/Databasus:owner+ops@example.com", parsedAddress.Path)
		query := parsedAddress.Query()
		assert.Equal(t, rfcSecret, query.Get("secret"))
		assert.Equal(t, "Databasus", query.Get("issuer"))
		assert.Equal(t, "SHA1", query.Get("algorithm"))
		assert.Equal(t, "6", query.Get("digits"))
		assert.Equal(t, "30", query.Get("period"))
	})
}
