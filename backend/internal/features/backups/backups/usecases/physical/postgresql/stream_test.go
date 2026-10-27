package usecases_physical_postgresql

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/toolstub"
)

// fakeStorageFileSaver records SaveFile payloads in memory and can be told to
// fail, so the streaming concurrency can be exercised without a real backend.
type fakeStorageFileSaver struct {
	mu      sync.Mutex
	saved   map[string][]byte
	saveErr error

	// The deadline is recorded rather than the context itself so no context is ever
	// parked in a struct field.
	savedDeadlines map[string]time.Time
}

func newFakeStorage() *fakeStorageFileSaver {
	return &fakeStorageFileSaver{
		saved:          map[string][]byte{},
		savedDeadlines: map[string]time.Time{},
	}
}

func (f *fakeStorageFileSaver) SaveFile(ctx context.Context, fileName string, file io.Reader) error {
	if f.saveErr != nil {
		return f.saveErr
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.saved[fileName] = contents

	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		f.savedDeadlines[fileName] = deadline
	}

	f.mu.Unlock()

	return nil
}

func (f *fakeStorageFileSaver) deadlineFor(fileName string) (time.Time, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	deadline, hasDeadline := f.savedDeadlines[fileName]

	return deadline, hasDeadline
}

func (f *fakeStorageFileSaver) has(fileName string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, ok := f.saved[fileName]

	return ok
}

func (f *fakeStorageFileSaver) DeleteFile(context.Context, string) error {
	return nil
}

func testRunStreamParams(
	t *testing.T,
	storage *fakeStorageFileSaver,
	codec physical_enums.PhysicalBackupCompression,
) runStreamParams {
	return runStreamParams{
		FileStore:  NewFakeFileStore(storage),
		StorageID:  uuid.New(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		FileName:   "test-obj",
		Encryption: backups_core_enums.BackupEncryptionNone,
		MasterKey:  nil,
		BackupID:   uuid.New(),
		Source:     pgphysical.NewSource(pgphysical.Settings{}),
		Codec:      codec,
	}
}

func testStreamBackupSpec(t *testing.T, codec pgphysical.Codec, script string) pgphysical.BaseBackupSpec {
	t.Helper()
	tools := pgshared.ClientTools{Root: t.TempDir()}
	toolstub.Write(
		t,
		tools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup),
		toolstub.Spec{Script: script},
	)
	return pgphysical.BaseBackupSpec{
		ClientTools: tools, Version: pgshared.Version17, Credentials: &pgshared.CredentialTempFiles{}, Codec: codec,
		SystemID: 7361234567890123456, Clock: clock.NewSystem(), FirstByteTimeout: time.Hour, StallTimeout: time.Hour,
		StallPollInterval: time.Second, ProgressInterval: time.Second, Logger: slog.New(slog.DiscardHandler),
	}
}

func writeZstdTar(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()

	var tarBuf bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuf)

	for name, body := range entries {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg, Name: name, Mode: 0o600, Size: int64(len(body)),
		}))
		_, err := tarWriter.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(tarBuf.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	return compressed.Bytes()
}

func TestRunStreamManifestSidecar(t *testing.T) {
	t.Parallel()
	t.Run("stores the artifact and rebuilt manifest when streaming succeeds", func(t *testing.T) {
		t.Parallel()
		artifact := writeZstdTar(t, map[string][]byte{
			"backup_label": []byte("START WAL LOCATION: 0/3000060\n"),
			"PG_VERSION":   []byte("17\n"),
		})

		tarPath := filepath.Join(t.TempDir(), "base.tar.zst")
		require.NoError(t, os.WriteFile(tarPath, artifact, 0o600))

		// stdout = the artifact; stderr = the start/stop point lines runStream parses.
		script := fmt.Sprintf(
			`cat %q; `+
				`echo "pg_basebackup: write-ahead log start point: 0/3000060 on timeline 1" >&2; `+
				`echo "pg_basebackup: write-ahead log end point: 0/3000220" >&2`,
			tarPath)

		storage := newFakeStorage()

		outcome, err := runStream(
			t.Context(),
			streamSpec{
				Params:   testRunStreamParams(t, storage, physical_enums.PhysicalBackupCompressionZstd),
				Backup:   testStreamBackupSpec(t, pgphysical.CodecZstd, script),
				Classify: classifyFullStreamError,
			},
		)
		require.NoError(t, err)

		require.Equal(t, physical_enums.PhysicalBackupStatusCompleted, outcome.Status, outcome.ErrorMessage)
		assert.Equal(t, physical_enums.PhysicalBackupCompressionZstd, outcome.Compression)
		assert.Equal(t, "test-obj.manifest", outcome.ManifestFileName)

		assert.Equal(t, artifact, storage.saved["test-obj"], "storage branch must store the raw compressed bytes")

		manifest := storage.saved["test-obj.manifest"]
		require.NotEmpty(t, manifest, "manifest sidecar must be saved")
		assert.Contains(t, string(manifest), `"PostgreSQL-Backup-Manifest-Version": 2`)
		assert.Contains(t, string(manifest), `"Path": "backup_label"`)
	})
}

func TestRunStreamStorageFailure(t *testing.T) {
	t.Parallel()
	t.Run("reports the storage failure when the upload cancels streaming", func(t *testing.T) {
		t.Parallel()
		storage := newFakeStorage()
		storage.saveErr = errors.New("backend unavailable")

		outcome, err := runStream(
			t.Context(),
			streamSpec{
				Params:   testRunStreamParams(t, storage, physical_enums.PhysicalBackupCompressionNone),
				Backup:   testStreamBackupSpec(t, pgphysical.CodecNone, `printf 'some-bytes'`),
				Classify: classifyFullStreamError,
			},
		)
		require.NoError(t, err)

		require.NotNil(t, outcome.ErrorReason)
		assert.Equal(t, physical_enums.PhysicalBackupErrorStorageUploadFailed, *outcome.ErrorReason,
			"a storage failure that cancels the stream must not be reported as a network stall")
	})
}

func TestRunStreamFailedAttempt(t *testing.T) {
	t.Parallel()
	t.Run("leaves no receipt to publish when the tool fails after writing bytes", func(t *testing.T) {
		t.Parallel()
		storage := newFakeStorage()

		outcome, err := runStream(
			t.Context(),
			streamSpec{
				Params:   testRunStreamParams(t, storage, physical_enums.PhysicalBackupCompressionNone),
				Backup:   testStreamBackupSpec(t, pgphysical.CodecNone, `printf 'partial physical backup'; exit 1`),
				Classify: classifyFullStreamError,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, outcome.ErrorReason)
		assert.Equal(t, physical_enums.PhysicalBackupStatusError, outcome.Status)
		assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *outcome.ErrorReason)
		assert.Equal(t, []byte("partial physical backup"), storage.saved["test-obj"])
		assert.Empty(t, outcome.Receipts, "a failed attempt must leave its bytes for cleanup, not for publication")
	})
}

func TestSaveEncryptedManifestSidecar(t *testing.T) {
	t.Parallel()
	t.Run("uses fresh encryption material when saving the manifest", func(t *testing.T) {
		t.Parallel()
		storage := newFakeStorage()
		backupID := uuid.New()
		masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(t.TempDir(), "secret.key"))
		require.NoError(t, err)
		manifest := []byte(`{ "PostgreSQL-Backup-Manifest-Version": 2, "System-Identifier": 42 }`)

		params := runStreamParams{
			FileStore:  NewFakeFileStore(storage),
			StorageID:  uuid.New(),
			Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			MasterKey:  masterKey,
			Encryption: backups_core_enums.BackupEncryptionEncrypted,
			BackupID:   backupID,
		}

		sidecar, err := saveManifestSidecar(
			t.Context(),
			manifestSidecarSpec{Params: params, FileName: "obj.manifest", Manifest: manifest},
		)
		require.NoError(t, err)
		require.NotEmpty(t, sidecar.SaltBase64)
		require.NotEmpty(t, sidecar.NonceBase64)

		saltB64, nonceB64 := sidecar.SaltBase64, sidecar.NonceBase64

		stored := storage.saved["obj.manifest"]
		require.NotEqual(t, manifest, stored, "stored sidecar must be ciphertext")

		salt, err := base64.StdEncoding.DecodeString(saltB64)
		require.NoError(t, err)
		nonce, err := base64.StdEncoding.DecodeString(nonceB64)
		require.NoError(t, err)

		decryptor, err := crypto.NewDecryptingReader(
			bytes.NewReader(stored), masterKey, backupID, crypto.StreamHeader{Salt: salt, Nonce: nonce},
		)
		require.NoError(t, err)

		plaintext, err := io.ReadAll(decryptor)
		require.NoError(t, err)
		assert.Equal(t, manifest, plaintext, "manifest must decrypt with its own salt/IV")
	})
}

func TestManifestSaveTimeout(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		payloadSizeBytes int
		timeout          time.Duration
	}{
		{"uses the minimum timeout with an empty manifest", 0, manifestSaveMinTimeout},
		{"uses the minimum timeout with a 1 mib manifest", 1024 * 1024, manifestSaveMinTimeout},
		{"scales the timeout with a 100 mib manifest", 100 * 1024 * 1024, 400 * time.Second},
		{"caps the timeout with a 1 gib manifest", 1024 * 1024 * 1024, manifestSaveMaxTimeout},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.timeout, manifestSaveTimeout(testCase.payloadSizeBytes))
		})
	}
}

func TestSaveSmallManifestSidecar(t *testing.T) {
	t.Parallel()
	t.Run("uses the minimum deadline when the manifest is small", func(t *testing.T) {
		t.Parallel()
		storage := newFakeStorage()
		params := testRunStreamParams(t, storage, physical_enums.PhysicalBackupCompressionNone)

		_, err := saveManifestSidecar(
			t.Context(),
			manifestSidecarSpec{Params: params, FileName: "obj.manifest", Manifest: []byte(`{ "Files": [] }`)},
		)
		require.NoError(t, err)

		deadline, hasDeadline := storage.deadlineFor("obj.manifest")
		require.True(t, hasDeadline, "the sidecar upload must stay bounded")

		assert.InDelta(t, manifestSaveMinTimeout.Seconds(), time.Until(deadline).Seconds(), 5,
			"a few-KB manifest gets the floor, not a size-derived budget")
	})
}

// A 40 MiB sidecar sits past the knee where the size-derived budget overtakes the
// floor (manifestSaveMinTimeout worth of throughput is 30 MiB), so it proves the
// deadline actually tracks the payload instead of being a renamed constant.
func TestSaveLargeManifestSidecar(t *testing.T) {
	t.Parallel()
	t.Run("extends the upload deadline when the manifest is large", func(t *testing.T) {
		t.Parallel()
		storage := newFakeStorage()
		params := testRunStreamParams(t, storage, physical_enums.PhysicalBackupCompressionNone)
		largeManifest := bytes.Repeat([]byte("m"), 40*1024*1024)

		_, err := saveManifestSidecar(
			t.Context(),
			manifestSidecarSpec{Params: params, FileName: "obj.manifest", Manifest: largeManifest},
		)
		require.NoError(t, err)

		deadline, hasDeadline := storage.deadlineFor("obj.manifest")
		require.True(t, hasDeadline, "the sidecar upload must stay bounded")

		assert.Greater(t, time.Until(deadline), manifestSaveMinTimeout,
			"a manifest past the floor's throughput budget must get more time")
		assert.Len(t, storage.saved["obj.manifest"], len(largeManifest))
	})
}
