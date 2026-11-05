package crypto_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
)

func TestEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("returns a value that decrypts to the plaintext", func(t *testing.T) {
		t.Parallel()
		encryptor := newFieldEncryptor(t)

		encryptedValue, err := encryptor.Encrypt("my-secret-password")
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(encryptedValue, "enc:"))
		assert.NotContains(t, encryptedValue, "my-secret-password")

		decryptedValue, err := encryptor.Decrypt(encryptedValue)
		require.NoError(t, err)
		assert.Equal(t, "my-secret-password", decryptedValue)
	})

	t.Run("returns an empty string unchanged", func(t *testing.T) {
		t.Parallel()

		encryptedValue, err := newFieldEncryptor(t).Encrypt("")

		require.NoError(t, err)
		assert.Empty(t, encryptedValue)
	})

	t.Run("returns an already encrypted value unchanged", func(t *testing.T) {
		t.Parallel()
		encryptor := newFieldEncryptor(t)
		encryptedValue, err := encryptor.Encrypt("my-password")
		require.NoError(t, err)

		reencryptedValue, err := encryptor.Encrypt(encryptedValue)

		require.NoError(t, err)
		assert.Equal(t, encryptedValue, reencryptedValue)
	})

	t.Run("uses a new nonce when the same plaintext is encrypted twice", func(t *testing.T) {
		t.Parallel()
		encryptor := newFieldEncryptor(t)

		firstValue, err := encryptor.Encrypt("shared-secret")
		require.NoError(t, err)
		secondValue, err := encryptor.Encrypt("shared-secret")
		require.NoError(t, err)

		assert.NotEqual(t, getNonce(t, firstValue), getNonce(t, secondValue))
		for _, encryptedValue := range []string{firstValue, secondValue} {
			decryptedValue, decryptErr := encryptor.Decrypt(encryptedValue)
			require.NoError(t, decryptErr)
			assert.Equal(t, "shared-secret", decryptedValue)
		}
	})
}

func TestDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty string unchanged", func(t *testing.T) {
		t.Parallel()

		decryptedValue, err := newFieldEncryptor(t).Decrypt("")

		require.NoError(t, err)
		assert.Empty(t, decryptedValue)
	})

	t.Run("returns an unencrypted value unchanged", func(t *testing.T) {
		t.Parallel()

		decryptedValue, err := newFieldEncryptor(t).Decrypt("not-encrypted-password")

		require.NoError(t, err)
		assert.Equal(t, "not-encrypted-password", decryptedValue)
	})

	t.Run("fails with a malformed value", func(t *testing.T) {
		t.Parallel()
		encryptor := newFieldEncryptor(t)

		_, err := encryptor.Decrypt("enc:invalid")
		require.ErrorIs(t, err, crypto.ErrMalformedCiphertext)

		_, err = encryptor.Decrypt("enc:invalid:invalid-base64")
		require.Error(t, err)
	})

	t.Run("decrypts a value encrypted with the deterministic nonce of older releases", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		legacyValue := encryptWithLegacyDeterministicNonce(
			t, masterKey.GetSecret(), []byte("legacy-deterministic-item-id-32!"), "legacy-payload",
		)

		decryptedValue, err := crypto.NewFieldEncryptor(masterKey).Decrypt(legacyValue)

		require.NoError(t, err)
		assert.Equal(t, "legacy-payload", decryptedValue)
	})
}

func newFieldEncryptor(t *testing.T) *crypto.FieldEncryptor {
	t.Helper()

	return crypto.NewFieldEncryptor(openMasterKey(t))
}

func openMasterKey(t *testing.T) *crypto.MasterKey {
	t.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(t.TempDir(), "secret.key"))
	require.NoError(t, err)

	return masterKey
}

func getNonce(t *testing.T, encryptedValue string) string {
	t.Helper()

	parts := strings.SplitN(encryptedValue, ":", 3)
	require.Len(t, parts, 3)

	return parts[1]
}

// encryptWithLegacyDeterministicNonce reproduces how older releases derived the nonce from the item's
// ID, which the stored value carries in its nonce part.
func encryptWithLegacyDeterministicNonce(t *testing.T, masterKey string, itemID []byte, plaintext string) string {
	t.Helper()

	block, err := aes.NewCipher([]byte(masterKey)[:32])
	require.NoError(t, err)

	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)

	nonceHash := hmac.New(sha256.New, []byte(masterKey))
	nonceHash.Write(itemID)
	nonce := nonceHash.Sum(nil)[:aead.NonceSize()]

	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)

	return "enc:" + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ciphertext)
}
