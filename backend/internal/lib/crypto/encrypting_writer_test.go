package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
)

func TestEncryptingWriter(t *testing.T) {
	t.Parallel()

	t.Run("decrypts to the written bytes with several writes", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		var encryptedStream bytes.Buffer

		writer, header, err := crypto.NewEncryptingWriter(&encryptedStream, masterKey, fileID)
		require.NoError(t, err)

		for _, part := range []string{"First part of data. ", "Second part of data. ", "Third part of data."} {
			writtenBytes, writeErr := writer.Write([]byte(part))
			require.NoError(t, writeErr)
			assert.Equal(t, len(part), writtenBytes)
		}
		require.NoError(t, writer.Close())

		decryptedBytes := decryptStream(t, encryptedStream.Bytes(), masterKey, fileID, header)
		assert.Equal(t, "First part of data. Second part of data. Third part of data.", string(decryptedBytes))
	})

	t.Run("decrypts to the written bytes when they span several chunks", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		plainBytes := make([]byte, 3*1024*1024+1000)
		_, err := rand.Read(plainBytes)
		require.NoError(t, err)

		encryptedBytes, header := encryptStream(t, plainBytes, masterKey, fileID)

		assert.Equal(t, plainBytes, decryptStream(t, encryptedBytes, masterKey, fileID, header))
	})

	t.Run("writes a stream that decrypts to nothing when nothing is written", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()

		encryptedBytes, header := encryptStream(t, nil, masterKey, fileID)

		assert.NotEmpty(t, encryptedBytes)
		assert.Empty(t, decryptStream(t, encryptedBytes, masterKey, fileID, header))
	})

	t.Run("hides the written bytes in the stream", func(t *testing.T) {
		t.Parallel()

		encryptedBytes, _ := encryptStream(t, []byte("encryption-marker-7f3a"), openMasterKey(t), uuid.New())

		assert.NotContains(t, string(encryptedBytes), "encryption-marker-7f3a")
	})

	t.Run("seals the same bytes into different streams with different headers", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()

		firstStream, firstHeader := encryptStream(t, []byte("unchanged dump"), masterKey, fileID)
		secondStream, secondHeader := encryptStream(t, []byte("unchanged dump"), masterKey, fileID)

		assert.NotEqual(t, firstStream, secondStream)
		assert.NotEqual(t, firstHeader.Salt, secondHeader.Salt)
		assert.NotEqual(t, firstHeader.Nonce, secondHeader.Nonce)
	})
}

func encryptStream(
	t *testing.T,
	plainBytes []byte,
	masterKey *crypto.MasterKey,
	fileID uuid.UUID,
) ([]byte, crypto.StreamHeader) {
	t.Helper()

	var encryptedStream bytes.Buffer

	writer, header, err := crypto.NewEncryptingWriter(&encryptedStream, masterKey, fileID)
	require.NoError(t, err)

	_, err = writer.Write(plainBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return encryptedStream.Bytes(), header
}
