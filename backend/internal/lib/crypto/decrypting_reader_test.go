package crypto_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
)

// An earlier release encrypted the fixture with the master key, file ID, salt and nonce below. Stored
// backups of earlier releases use that format, so a change that breaks this test breaks their restores.
const (
	fixtureMasterKeySecret = "stream-encryption-fixture-master-key-0123456789"
	fixtureFileID          = "6f1c3b52-8d0e-4a7b-9c21-3e5f4d6a7b80"
	fixturePlainSize       = 1024*1024 + 4096
)

func TestDecryptingReader(t *testing.T) {
	t.Parallel()

	t.Run("decrypts a stream of the previous format to its original bytes", func(t *testing.T) {
		t.Parallel()
		encryptedFixture, err := os.ReadFile("testdata/stream_encryption_fixture.bin")
		require.NoError(t, err)
		masterKey, err := crypto.OpenOrCreateMasterKey(writeMasterKeyFile(t, fixtureMasterKeySecret))
		require.NoError(t, err)

		decryptedBytes := decryptStream(
			t, encryptedFixture, masterKey, uuid.MustParse(fixtureFileID), getFixtureHeader(),
		)

		assert.Equal(t, getFixturePlainBytes(), decryptedBytes)
	})

	t.Run("returns the bytes when they are read five at a time", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		plainBytes := []byte("This is test data that will be read in small chunks.")
		encryptedBytes, header := encryptStream(t, plainBytes, masterKey, fileID)
		reader, err := crypto.NewDecryptingReader(bytes.NewReader(encryptedBytes), masterKey, fileID, header)
		require.NoError(t, err)

		var decryptedBytes []byte
		readBuffer := make([]byte, 5)
		for {
			readBytes, readErr := reader.Read(readBuffer)
			decryptedBytes = append(decryptedBytes, readBuffer[:readBytes]...)
			if errors.Is(readErr, io.EOF) {
				break
			}
			require.NoError(t, readErr)
		}

		assert.Equal(t, plainBytes, decryptedBytes)
	})

	t.Run("fails authentication when a byte of a chunk is changed", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		encryptedBytes, header := encryptStream(t, []byte("This data will be tampered with."), masterKey, fileID)
		encryptedBytes[len(encryptedBytes)-10] ^= 0xFF

		readErr := readWholeStream(t, encryptedBytes, masterKey, fileID, header)

		require.ErrorIs(t, readErr, crypto.ErrStreamChunkAuthentication)
	})

	t.Run("fails when the stream is cut inside a chunk", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		encryptedBytes, header := encryptStream(t, bytes.Repeat([]byte("x"), 4096), masterKey, fileID)

		readErr := readWholeStream(t, encryptedBytes[:len(encryptedBytes)-2048], masterKey, fileID, header)

		require.ErrorIs(t, readErr, io.ErrUnexpectedEOF)
	})

	t.Run("fails authentication with another master key", func(t *testing.T) {
		t.Parallel()
		fileID := uuid.New()
		encryptedBytes, header := encryptStream(t, []byte("Secret data"), openMasterKey(t), fileID)

		readErr := readWholeStream(t, encryptedBytes, openMasterKey(t), fileID, header)

		require.ErrorIs(t, readErr, crypto.ErrStreamChunkAuthentication)
	})

	t.Run("fails authentication with another file ID", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		encryptedBytes, header := encryptStream(t, []byte("Secret data"), masterKey, uuid.New())

		readErr := readWholeStream(t, encryptedBytes, masterKey, uuid.New(), header)

		require.ErrorIs(t, readErr, crypto.ErrStreamChunkAuthentication)
	})

	t.Run("refuses a stream whose header differs from the stored one", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		encryptedBytes, _ := encryptStream(t, []byte("first backup"), masterKey, fileID)
		_, otherHeader := encryptStream(t, []byte("second backup"), masterKey, fileID)

		_, err := crypto.NewDecryptingReader(bytes.NewReader(encryptedBytes), masterKey, fileID, otherHeader)

		require.ErrorIs(t, err, crypto.ErrStreamHeaderMismatch)
	})

	t.Run("refuses a stream without the magic bytes", func(t *testing.T) {
		t.Parallel()
		masterKey := openMasterKey(t)
		fileID := uuid.New()
		encryptedBytes, header := encryptStream(t, []byte("data"), masterKey, fileID)
		copy(encryptedBytes, "INVALID!")

		_, err := crypto.NewDecryptingReader(bytes.NewReader(encryptedBytes), masterKey, fileID, header)

		require.ErrorIs(t, err, crypto.ErrInvalidStreamMagicBytes)
	})
}

func decryptStream(
	t *testing.T,
	encryptedBytes []byte,
	masterKey *crypto.MasterKey,
	fileID uuid.UUID,
	header crypto.StreamHeader,
) []byte {
	t.Helper()

	reader, err := crypto.NewDecryptingReader(bytes.NewReader(encryptedBytes), masterKey, fileID, header)
	require.NoError(t, err)

	decryptedBytes, err := io.ReadAll(reader)
	require.NoError(t, err)

	return decryptedBytes
}

func readWholeStream(
	t *testing.T,
	encryptedBytes []byte,
	masterKey *crypto.MasterKey,
	fileID uuid.UUID,
	header crypto.StreamHeader,
) error {
	t.Helper()

	reader, err := crypto.NewDecryptingReader(bytes.NewReader(encryptedBytes), masterKey, fileID, header)
	require.NoError(t, err)

	_, err = io.ReadAll(reader)

	return err
}

func getFixtureHeader() crypto.StreamHeader {
	header := crypto.StreamHeader{Salt: make([]byte, 32), Nonce: make([]byte, 12)}
	for i := range header.Salt {
		header.Salt[i] = byte(i)
	}

	for i := range header.Nonce {
		header.Nonce[i] = byte(100 + i)
	}

	return header
}

func getFixturePlainBytes() []byte {
	plainBytes := make([]byte, fixturePlainSize)
	for i := range plainBytes {
		plainBytes[i] = byte(i*31 + 7)
	}

	return plainBytes
}
