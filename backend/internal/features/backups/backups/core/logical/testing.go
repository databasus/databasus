//go:build testhelpers

package backups_core_logical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	storage_files "databasus-backend/internal/features/storages/files"
	test_utils "databasus-backend/internal/util/testing"
)

// FailingBackupFileStore fails every write with FailureErr. It either drains the whole stream
// first, so the dump runs to completion before the storage fails, or fails before reading a
// byte, so the storage fails while the dump is still producing data.
type FailingBackupFileStore struct {
	FailureErr             error
	IsFailingBeforeReading bool

	readBytes atomic.Int64
}

func (s *FailingBackupFileStore) WriteFile(
	_ context.Context,
	_ storage_files.StoredFileReference,
	file io.Reader,
) (storage_files.WriteReceipt, error) {
	if !s.IsFailingBeforeReading {
		readBytes, _ := io.Copy(io.Discard, file)
		s.readBytes.Store(readBytes)
	}

	return storage_files.WriteReceipt{}, s.FailureErr
}

func (s *FailingBackupFileStore) GetReadBytes() int64 {
	return s.readBytes.Load()
}

// discardingBackupFileStore reads every write to the end and keeps nothing, for tests whose
// backup fails for a reason other than the storage.
type discardingBackupFileStore struct{}

func (s *discardingBackupFileStore) WriteFile(
	_ context.Context,
	_ storage_files.StoredFileReference,
	file io.Reader,
) (storage_files.WriteReceipt, error) {
	_, err := io.Copy(io.Discard, file)

	return storage_files.WriteReceipt{}, err
}

// ExecuteBackupInto runs one engine's real backup use case against a test database, writing into
// the given file store.
type ExecuteBackupInto func(
	ctx context.Context,
	backup *LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	fileStore BackupFileStore,
) error

// AssertStorageTimeoutReportedAsStorageFailure runs an engine's real dump into a storage that
// times out, once after the dump finished and once while it is still running, and checks the
// engine reports the storage error rather than a cancellation.
func AssertStorageTimeoutReportedAsStorageFailure(t *testing.T, executeBackupInto ExecuteBackupInto) {
	t.Helper()

	storageErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))

	t.Run("Test_Execute_WhenStorageTimesOutAfterDumpFinished_ReturnsStorageError", func(t *testing.T) {
		fileStore := &FailingBackupFileStore{FailureErr: storageErr}

		backupErr := executeBackupInto(
			t.Context(),
			newTestBackup(),
			newUnencryptedTestBackupConfig(),
			fileStore,
		)

		require.Positive(t, fileStore.GetReadBytes(), "the dump never reached the storage")
		assertStorageErrorReported(t, backupErr, storageErr)
	})

	t.Run("Test_Execute_WhenStorageFailsBeforeReading_ReturnsStorageError", func(t *testing.T) {
		fileStore := &FailingBackupFileStore{FailureErr: storageErr, IsFailingBeforeReading: true}

		backupErr := executeBackupInto(
			t.Context(),
			newTestBackup(),
			newUnencryptedTestBackupConfig(),
			fileStore,
		)

		assertStorageErrorReported(t, backupErr, storageErr)
	})
}

// ExecuteBackupWithDumpTool runs one engine's backup use case with the given binary in place of
// its dump tool, writing into the given file store.
type ExecuteBackupWithDumpTool func(
	ctx context.Context,
	backup *LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	fileStore BackupFileStore,
	dumpToolBin string,
) error

// AssertDumpToolErrorReported runs an engine's backup with a dump tool that writes an error and
// exits before producing any data, and checks the engine reports that error.
func AssertDumpToolErrorReported(t *testing.T, executeBackupWithDumpTool ExecuteBackupWithDumpTool) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("stub dump tool relies on a POSIX shell")
	}

	const dumpToolError = "error: query failed: ERROR:  permission denied for sequence orders_id_seq"

	// The error reaches the error output only after the dump tool has exited, which is how a
	// fast-exiting tool looks when the reader of its output is scheduled late, as on a loaded CI
	// runner. A capture that stops reading when the process exits loses it every time.
	dumpToolBin := filepath.Join(t.TempDir(), "dump-tool")
	dumpToolScript := "#!/bin/sh\n(sleep 0.2; echo '" + dumpToolError + "' >&2) >/dev/null &\nexit 1\n"
	require.NoError(t, os.WriteFile(dumpToolBin, []byte(dumpToolScript), 0o700))

	backupErr := executeBackupWithDumpTool(
		t.Context(),
		newTestBackup(),
		newUnencryptedTestBackupConfig(),
		&discardingBackupFileStore{},
		dumpToolBin,
	)

	require.Error(t, backupErr)
	assert.Contains(t, backupErr.Error(), dumpToolError)
	assert.NotContains(t, backupErr.Error(), "save to storage")
}

func assertStorageErrorReported(t *testing.T, backupErr, storageErr error) {
	t.Helper()

	require.Error(t, backupErr)
	assert.True(t, errors.Is(backupErr, storageErr), "the backup error must wrap the storage error, got: %v", backupErr)
	assert.Contains(t, backupErr.Error(), "save to storage")
	assert.NotContains(t, backupErr.Error(), "backup cancelled")
}

func newTestBackup() *LogicalBackup {
	backupID := uuid.New()

	return &LogicalBackup{
		ID:         backupID,
		DatabaseID: uuid.New(),
		StorageID:  uuid.New(),
		FileName:   "test-backup-" + backupID.String(),
		Status:     BackupStatusInProgress,
	}
}

func newUnencryptedTestBackupConfig() *backups_config_logical.LogicalBackupConfig {
	return &backups_config_logical.LogicalBackupConfig{Encryption: backups_core_enums.BackupEncryptionNone}
}
