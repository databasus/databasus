package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Encrypt_Decrypt_RoundTrip(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "my-secret-password"

	encrypted, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)
	assert.Contains(t, encrypted, "enc:")

	decrypted, err := encryptor.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func Test_Encrypt_EmptyString_ReturnsEmpty(t *testing.T) {
	encryptor := GetFieldEncryptor()

	encrypted, err := encryptor.Encrypt("")
	assert.NoError(t, err)
	assert.Empty(t, encrypted)
}

func Test_Decrypt_EmptyString_ReturnsEmpty(t *testing.T) {
	encryptor := GetFieldEncryptor()

	decrypted, err := encryptor.Decrypt("")
	assert.NoError(t, err)
	assert.Empty(t, decrypted)
}

func Test_Decrypt_PlaintextValue_ReturnsAsIs(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "not-encrypted-password"

	decrypted, err := encryptor.Decrypt(plaintext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func Test_Encrypt_DetectsAlreadyEncryptedFormat(t *testing.T) {
	encryptor := GetFieldEncryptor()
	alreadyEncrypted := "enc:nonce:ciphertext"

	result, err := encryptor.Encrypt(alreadyEncrypted)
	assert.NoError(t, err)
	assert.Equal(t, alreadyEncrypted, result)
}

func Test_Encrypt_SamePlaintext_TwiceProducesDifferentCiphertext(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "shared-secret"

	encrypted1, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)

	assert.NotEqual(t, encrypted1, encrypted2)

	decrypted1, err := encryptor.Decrypt(encrypted1)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := encryptor.Decrypt(encrypted2)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func Test_Encrypt_SamePlaintext_TwiceProducesDifferentNonces(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "regression-test-secret"

	encrypted1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	nonce1 := extractNonce(t, encrypted1)
	nonce2 := extractNonce(t, encrypted2)

	assert.NotEqual(t, nonce1, nonce2, "nonces must be random per call, not deterministic")
}

func Test_Encrypt_AlreadyEncrypted_ReturnsAsIs(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "my-password"

	encrypted1, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(encrypted1)
	assert.NoError(t, err)

	assert.Equal(t, encrypted1, encrypted2)
}

func Test_Decrypt_MalformedData_ReturnsError(t *testing.T) {
	encryptor := GetFieldEncryptor()

	_, err := encryptor.Decrypt("enc:invalid")
	assert.Error(t, err)

	_, err = encryptor.Decrypt("enc:invalid:invalid-base64")
	assert.Error(t, err)
}

func Test_EncryptedFormat_ContainsPrefix(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "test-secret"

	encrypted, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.Contains(t, encrypted, "enc:")
}

// Test_Decrypt_LegacyDeterministicNonceCiphertext_StillDecrypts proves the
// no-migration claim: a ciphertext produced by the previous deterministic-nonce
// implementation must still decrypt under the new random-nonce code, because
// the on-wire format carries the nonce and decryption reads it from there.
func Test_Decrypt_LegacyDeterministicNonceCiphertext_StillDecrypts(t *testing.T) {
	encryptor := GetFieldEncryptor()

	masterKey, err := getFieldEncryptor().secretKeyService.GetSecretKey()
	require.NoError(t, err)

	plaintext := "legacy-payload"
	legacyItemID := []byte("legacy-deterministic-item-id-32!")

	legacyCiphertext := encryptWithLegacyDeterministicNonce(t, masterKey, legacyItemID, plaintext)

	decrypted, err := encryptor.Decrypt(legacyCiphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func extractNonce(t *testing.T, ciphertext string) string {
	t.Helper()

	parts := strings.SplitN(ciphertext, ":", 3)
	require.Len(t, parts, 3, "ciphertext must be in enc:<nonce>:<ct> format")

	return parts[1]
}

// encryptWithLegacyDeterministicNonce reproduces the previous deterministic-nonce
// encryption path so we can prove the new code still decrypts old ciphertexts.
func encryptWithLegacyDeterministicNonce(
	t *testing.T,
	masterKey string,
	itemID []byte,
	plaintext string,
) string {
	t.Helper()

	block, err := aes.NewCipher([]byte(masterKey)[:32])
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	h := hmac.New(sha256.New, []byte(masterKey))
	h.Write(itemID)
	nonce := h.Sum(nil)[:gcm.NonceSize()]

	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	return fmt.Sprintf("%s%s:%s",
		encryptedPrefix,
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ct),
	)
}
