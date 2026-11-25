package totp_test

import (
	"encoding/base32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/totp"
)

func TestGenerateSecret(t *testing.T) {
	t.Parallel()

	t.Run("returns 160 random bits as unpadded base32", func(t *testing.T) {
		t.Parallel()

		firstSecret, err := totp.GenerateSecret()
		require.NoError(t, err)
		secondSecret, err := totp.GenerateSecret()
		require.NoError(t, err)

		secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(firstSecret)
		require.NoError(t, err)
		assert.Len(t, secretBytes, 20)
		assert.NotContains(t, firstSecret, "=")
		assert.NotEqual(t, firstSecret, secondSecret)
	})
}
