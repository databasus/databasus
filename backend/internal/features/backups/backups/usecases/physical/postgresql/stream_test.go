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
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
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

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.saved[fileName] = data

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
	storage *fakeStorageFileSaver,
	codec physical_enums.PhysicalBackupCompression,
) runStreamParams {
	return runStreamParams{
		FileStore:      NewFakeFileStore(storage),
		StorageID:      uuid.New(),
		FieldEncryptor: nil,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		FileName:       "test-obj",
		Encryption:     backups_core_enums.BackupEncryptionNone,
		MasterKey:      "",
		BackupID:       uuid.New(),
		SystemID:       7361234567890123456,
		Codec:          codec,
	}
}

// shellBuildCmd returns a buildCmd that runs the given sh script, bound to the
// stream's context so cancellation reaches the process — exactly how the real
// command builders wire pg_basebackup.
func shellBuildCmd(script string) func(context.Context) (*exec.Cmd, error) {
	return func(ctx context.Context) (*exec.Cmd, error) {
		return exec.CommandContext(ctx, "sh", "-c", script), nil
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

func Test_RunStream_WhenStreamSucceeds_ReconstructsManifestSidecar(t *testing.T) {
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

	outcome, err := runStream(t.Context(), testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionZstd),
		shellBuildCmd(script), classifyFullStreamError)
	require.NoError(t, err)

	require.Equal(t, physical_enums.PhysicalBackupStatusCompleted, outcome.Status, outcome.ErrorMessage)
	assert.Equal(t, physical_enums.PhysicalBackupCompressionZstd, outcome.Compression)
	assert.Equal(t, "test-obj.manifest", outcome.ManifestFileName)

	assert.Equal(t, artifact, storage.saved["test-obj"], "storage branch must store the raw compressed bytes")

	manifest := storage.saved["test-obj.manifest"]
	require.NotEmpty(t, manifest, "manifest sidecar must be saved")
	assert.Contains(t, string(manifest), `"PostgreSQL-Backup-Manifest-Version": 2`)
	assert.Contains(t, string(manifest), `"Path": "backup_label"`)
}

func Test_RunStream_WhenStreamSucceeds_ReportsProgressDuringStream(t *testing.T) {
	// Tighten the poll cadence so a short test stream still crosses several ticks.
	originalInterval := progressReportInterval
	progressReportInterval = 10 * time.Millisecond
	t.Cleanup(func() { progressReportInterval = originalInterval })

	artifact := writeZstdTar(t, map[string][]byte{
		"backup_label": []byte("START WAL LOCATION: 0/3000060\n"),
		"PG_VERSION":   []byte("17\n"),
	})

	tarPath := filepath.Join(t.TempDir(), "base.tar.zst")
	require.NoError(t, os.WriteFile(tarPath, artifact, 0o600))

	// Emit the artifact, then idle long enough for the poller to tick on the
	// already-written bytes before the process exits.
	script := fmt.Sprintf(
		`cat %q; sleep 0.2; `+
			`echo "pg_basebackup: write-ahead log start point: 0/3000060 on timeline 1" >&2; `+
			`echo "pg_basebackup: write-ahead log end point: 0/3000220" >&2`,
		tarPath)

	var (
		mu              sync.Mutex
		maxReportedSize float64
		reportCount     int
	)

	params := testRunStreamParams(newFakeStorage(), physical_enums.PhysicalBackupCompressionZstd)
	params.ProgressListener = func(completedMb float64, _ int64) {
		mu.Lock()
		defer mu.Unlock()

		reportCount++
		if completedMb > maxReportedSize {
			maxReportedSize = completedMb
		}
	}

	outcome, err := runStream(t.Context(), params, shellBuildCmd(script), classifyFullStreamError)
	require.NoError(t, err)
	require.Equal(t, physical_enums.PhysicalBackupStatusCompleted, outcome.Status, outcome.ErrorMessage)

	mu.Lock()
	defer mu.Unlock()

	assert.Positive(t, reportCount, "progress listener must fire during the stream, not only at the end")
	assert.Positive(t, maxReportedSize, "reported size must reflect the bytes streamed to storage")
}

func Test_RunStream_WhenStorageSaveFails_ReturnsStorageUploadFailedNotStall(t *testing.T) {
	storage := newFakeStorage()
	storage.saveErr = errors.New("backend unavailable")

	outcome, err := runStream(t.Context(), testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone),
		shellBuildCmd(`printf 'some-bytes'`), classifyFullStreamError)
	require.NoError(t, err)

	require.NotNil(t, outcome.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorStorageUploadFailed, *outcome.ErrorReason,
		"a storage failure that cancels the stream must not be reported as a network stall")
}

// runStream sits below the layer that owns cleanup, so what it guarantees is that a
// failed attempt hands back nothing to publish. The bytes it already uploaded are
// removed by the worker, which the use-case tests cover against a real store.
func Test_RunStream_WhenPgBasebackupWritesBytesThenFails_LeavesNoReceiptToPublish(t *testing.T) {
	storage := newFakeStorage()

	outcome, err := runStream(
		t.Context(),
		testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone),
		shellBuildCmd(`printf 'partial physical backup'; exit 1`),
		classifyFullStreamError,
	)
	require.NoError(t, err)
	require.NotNil(t, outcome.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, outcome.Status)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *outcome.ErrorReason)
	assert.Equal(t, []byte("partial physical backup"), storage.saved["test-obj"])
	assert.Empty(t, outcome.Receipts, "a failed attempt must leave its bytes for cleanup, not for publication")
}

func Test_RunStream_WhenManifestWalkFails_ReturnsManifestCorrupted(t *testing.T) {
	// 1 KB of non-tar bytes: pg exits clean, but the walk hits a bad tar header
	// (not a truncation), so the manifest goroutine flags genuine corruption.
	storage := newFakeStorage()

	outcome, err := runStream(t.Context(), testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone),
		shellBuildCmd(`head -c 1024 /dev/zero | tr '\0' 'x'`), classifyFullStreamError)
	require.NoError(t, err)

	require.NotNil(t, outcome.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorManifestCorrupted, *outcome.ErrorReason)
}

func Test_RunStream_WhenCompressionUnsupported_ReturnsRetrySignal(t *testing.T) {
	storage := newFakeStorage()

	outcome, err := runStream(
		t.Context(),
		testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionZstd),
		shellBuildCmd(
			`echo "pg_basebackup: error: could not initiate base backup: ERROR:  invalid compression specification: `+
				`this build does not support compression with ZSTD" >&2; exit 1`,
		),
		classifyFullStreamError,
	)
	require.NoError(t, err)

	assert.True(t, outcome.isCompressionUnsupported,
		"the build-rejection stderr must surface as the retryable downgrade signal")
}

func Test_RunStream_WhenCmdStartFails_ReturnsErrorWithoutLeaking(t *testing.T) {
	storage := newFakeStorage()

	// A valid *exec.Cmd whose binary does not exist: StdoutPipe/StderrPipe succeed,
	// then Start() fails — exercising the early-return teardown that must close and
	// drain BOTH branch goroutines (the test simply completing proves no leak/hang).
	buildCmd := func(ctx context.Context) (*exec.Cmd, error) {
		return exec.CommandContext(ctx, filepath.Join(t.TempDir(), "does-not-exist")), nil
	}

	_, err := runStream(t.Context(), testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone),
		buildCmd, classifyFullStreamError)
	require.Error(t, err)
	assert.False(t, storage.has("test-obj.manifest"), "no manifest on a failed start")
}

// Test_SaveManifestSidecar_WhenEncrypted_RoundTripsWithOwnSaltIV proves the
// secure-default path: the sidecar is encrypted with its OWN fresh salt/nonce
// (not the tar's), and the returned salt/IV decrypt it back. The e2e and FullOnly
// run encryption off, so this is the only coverage of the encrypted manifest path.
func Test_SaveManifestSidecar_WhenEncrypted_RoundTripsWithOwnSaltIV(t *testing.T) {
	storage := newFakeStorage()
	backupID := uuid.New()
	masterKey := "test-master-key-0123456789"
	manifest := []byte(`{ "PostgreSQL-Backup-Manifest-Version": 2, "System-Identifier": 42 }`)

	params := runStreamParams{
		FileStore:  NewFakeFileStore(storage),
		StorageID:  uuid.New(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		MasterKey:  masterKey,
		Encryption: backups_core_enums.BackupEncryptionEncrypted,
		BackupID:   backupID,
	}

	sidecar, err := saveManifestSidecar(t.Context(), params, "obj.manifest", manifest)
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

	decryptor, err := backup_encryption.NewDecryptionReader(bytes.NewReader(stored), masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(decryptor)
	require.NoError(t, err)
	assert.Equal(t, manifest, plaintext, "manifest must decrypt with its own salt/IV")
}

func Test_ManifestSaveTimeout_ScalesWithSizeBetweenFloorAndCeiling(t *testing.T) {
	cases := []struct {
		name             string
		payloadSizeBytes int
		timeout          time.Duration
	}{
		{"empty manifest", 0, manifestSaveMinTimeout},
		{"1 MiB manifest", 1024 * 1024, manifestSaveMinTimeout},
		{"100 MB manifest", 100 * 1024 * 1024, 400 * time.Second},
		{"1 GiB manifest", 1024 * 1024 * 1024, manifestSaveMaxTimeout},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.timeout, manifestSaveTimeout(testCase.payloadSizeBytes))
		})
	}
}

func Test_SaveManifestSidecar_WhenManifestIsSmall_UsesFloorDeadline(t *testing.T) {
	storage := newFakeStorage()
	params := testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone)

	_, err := saveManifestSidecar(t.Context(), params, "obj.manifest", []byte(`{ "Files": [] }`))
	require.NoError(t, err)

	deadline, hasDeadline := storage.deadlineFor("obj.manifest")
	require.True(t, hasDeadline, "the sidecar upload must stay bounded")

	assert.InDelta(t, manifestSaveMinTimeout.Seconds(), time.Until(deadline).Seconds(), 5,
		"a few-KB manifest gets the floor, not a size-derived budget")
}

// A 40 MiB sidecar sits past the knee where the size-derived budget overtakes the
// floor (manifestSaveMinTimeout worth of throughput is 30 MiB), so it proves the
// deadline actually tracks the payload instead of being a renamed constant.
func Test_SaveManifestSidecar_WhenManifestIsLarge_ExtendsDeadlineBeyondFloor(t *testing.T) {
	storage := newFakeStorage()
	params := testRunStreamParams(storage, physical_enums.PhysicalBackupCompressionNone)
	largeManifest := bytes.Repeat([]byte("m"), 40*1024*1024)

	_, err := saveManifestSidecar(t.Context(), params, "obj.manifest", largeManifest)
	require.NoError(t, err)

	deadline, hasDeadline := storage.deadlineFor("obj.manifest")
	require.True(t, hasDeadline, "the sidecar upload must stay bounded")

	assert.Greater(t, time.Until(deadline), manifestSaveMinTimeout,
		"a manifest past the floor's throughput budget must get more time")
	assert.Len(t, storage.saved["obj.manifest"], len(largeManifest))
}
