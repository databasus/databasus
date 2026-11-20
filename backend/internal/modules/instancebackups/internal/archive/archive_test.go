package archive

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
)

// encryptedStreamStart is where the first chunk's length prefix sits: after the plaintext header
// and the 64-byte header of the encrypted stream.
const encryptedStreamStart = outerHeaderLength + 64

func TestArchive(t *testing.T) {
	t.Parallel()

	t.Run("returns the exact dump and manifest after a complete write", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		dump := newRandomDump(t, 3*maxDumpRecordSize+17)
		manifest := newTestManifest()

		file, summary := writeArchive(t, masterKey, manifest, dump)
		reader, err := Open(bytes.NewReader(file), masterKey)
		require.NoError(t, err)
		readDump, err := io.ReadAll(reader)

		require.NoError(t, err)
		assert.Equal(t, dump, readDump)
		assert.Equal(t, sha256.Sum256(dump), summary.DumpSha256)
		assert.Equal(t, uint64(len(dump)), summary.DumpLength)
		assert.Equal(t, manifest.BackupID, reader.GetManifest().BackupID)
		assert.Equal(t, manifest.SourceRelease, reader.GetManifest().SourceRelease)
		assert.Equal(t, manifest.SchemaIdentity, reader.GetManifest().SchemaIdentity)
		assert.True(t, manifest.SnapshotAt.Equal(reader.GetManifest().SnapshotAt))
	})

	t.Run("validates an empty dump", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)

		file, _ := writeArchive(t, masterKey, newTestManifest(), nil)
		_, err := Validate(bytes.NewReader(file), masterKey)

		require.NoError(t, err)
	})

	t.Run("refuses another master key before reading the dump", func(t *testing.T) {
		t.Parallel()
		file, _ := writeArchive(t, newTestMasterKey(t), newTestManifest(), newRandomDump(t, 1000))

		_, err := Open(bytes.NewReader(file), newTestMasterKey(t))

		require.ErrorIs(t, err, ErrMasterKeyMismatch)
	})

	t.Run("refuses a file whose bytes were changed", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		file, _ := writeArchive(t, masterKey, newTestManifest(), newRandomDump(t, 2*maxDumpRecordSize))

		file[len(file)/2] ^= 0xFF
		_, err := Validate(bytes.NewReader(file), masterKey)

		require.ErrorIs(t, err, ErrArchiveCorrupted)
	})

	t.Run("refuses a file cut exactly between encrypted chunks before its end record", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		file, _ := writeArchive(t, masterKey, newTestManifest(), newRandomDump(t, 3*maxDumpRecordSize))
		boundaries := getChunkBoundaries(t, file)
		require.Greater(t, len(boundaries), 2)

		for _, boundary := range boundaries[:len(boundaries)-1] {
			_, err := Validate(bytes.NewReader(file[:boundary]), masterKey)

			require.ErrorIs(t, err, ErrArchiveIncomplete, "cut at byte %d", boundary)
		}
	})

	t.Run("refuses a file cut inside its last chunk", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		file, _ := writeArchive(t, masterKey, newTestManifest(), newRandomDump(t, 5000))

		_, err := Validate(bytes.NewReader(file[:len(file)-10]), masterKey)

		require.ErrorIs(t, err, ErrArchiveIncomplete)
	})

	t.Run("refuses bytes appended after the end record", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		file, _ := writeArchive(t, masterKey, newTestManifest(), newRandomDump(t, 5000))

		for _, appended := range [][]byte{{0}, newRandomDump(t, 40)} {
			_, err := Validate(bytes.NewReader(append(bytes.Clone(file), appended...)), masterKey)

			require.Error(t, err)
			assert.NotErrorIs(t, err, ErrMasterKeyMismatch)
		}
	})

	t.Run("refuses a file that stops without an end record", func(t *testing.T) {
		t.Parallel()
		masterKey := newTestMasterKey(t)
		var file bytes.Buffer
		writer, err := NewWriter(&file, masterKey, newTestManifest())
		require.NoError(t, err)
		_, err = writer.Write(newRandomDump(t, 1000))
		require.NoError(t, err)
		require.NoError(t, writer.encryptor.Close())

		_, err = Validate(bytes.NewReader(file.Bytes()), masterKey)

		require.ErrorIs(t, err, ErrArchiveIncomplete)
	})

	t.Run("refuses a file of another format", func(t *testing.T) {
		t.Parallel()

		_, err := Open(bytes.NewReader([]byte("PGDMP plain dump of something else")), newTestMasterKey(t))

		require.ErrorIs(t, err, ErrNotDatabasusBackup)
	})
}

func writeArchive(t *testing.T, masterKey *crypto.MasterKey, manifest Manifest, dump []byte) ([]byte, Summary) {
	t.Helper()

	var file bytes.Buffer

	writer, err := NewWriter(&file, masterKey, manifest)
	require.NoError(t, err)
	_, err = writer.Write(dump)
	require.NoError(t, err)
	summary, err := writer.Finish()
	require.NoError(t, err)

	return file.Bytes(), summary
}

// getChunkBoundaries returns the offset after every encrypted chunk, the last being the file's end.
func getChunkBoundaries(t *testing.T, file []byte) []int {
	t.Helper()

	boundaries := []int{}
	offset := encryptedStreamStart

	for offset < len(file) {
		chunkLength := int(binary.BigEndian.Uint32(file[offset : offset+4]))
		offset += 4 + chunkLength
		boundaries = append(boundaries, offset)
	}

	require.Equal(t, len(file), offset)

	return boundaries
}

func newTestManifest() Manifest {
	return Manifest{
		FormatVersion: 0,
		BackupID:      uuid.New(),
		SourceRelease: "v3.4.5",
		SnapshotAt:    time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC),
		SchemaIdentity: database.SchemaIdentity{
			MigrationBundleDigest:    "abc123",
			AppliedMigrationVersions: []int64{20251031143019, 20261114100000},
		},
	}
}

func newTestMasterKey(t *testing.T) *crypto.MasterKey {
	t.Helper()

	masterKey, err := crypto.NewMasterKey([]byte(uuid.NewString() + uuid.NewString()))
	require.NoError(t, err)

	return masterKey
}

func newRandomDump(t *testing.T, length int) []byte {
	t.Helper()

	dump := make([]byte, length)
	_, err := rand.Read(dump)
	require.NoError(t, err)

	return dump
}
