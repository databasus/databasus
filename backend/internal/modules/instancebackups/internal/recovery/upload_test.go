package recovery

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/diskusage"
)

func TestUploadPreparation(t *testing.T) {
	t.Parallel()

	t.Run(
		"prepares an upload with the exact entered key bytes and keeps the installation unchanged",
		func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			uploadTarget := newUploadTarget(t)
			activeKey := uploadTarget.readKey(t)

			preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))

			archiveSha256, err := hashFile(source.path)
			require.NoError(t, err)
			stagedKey, err := os.ReadFile(uploadTarget.getDirectory().getStagedKeyPath(preparedRecovery.ID))
			require.NoError(t, err)
			require.Equal(t, StatusReady, preparedRecovery.Status)
			assert.Equal(t, OriginUpload, preparedRecovery.Origin)
			assert.Equal(t, archiveSha256, preparedRecovery.ArchiveSha256)
			require.NotNil(t, preparedRecovery.SourceRelease)
			assert.Equal(t, testRelease, *preparedRecovery.SourceRelease)
			assert.NotNil(t, preparedRecovery.SnapshotAt)
			require.NotNil(t, preparedRecovery.ExpiresAt)
			assert.Equal(t, uploadTarget.clock.Now().Add(time.Hour), *preparedRecovery.ExpiresAt)
			assert.Equal(t, source.keyContent, stagedKey)
			assert.Equal(t, activeKey, uploadTarget.readKey(t))
			assert.Empty(t, uploadTarget.listWorkspaceNames(t))
			assert.Len(t, uploadTarget.listRecoveryDatabases(t), 1)
			assert.NotContains(t, uploadTarget.getLogs(), strings.TrimSpace(string(source.keyContent)))
		},
	)

	for _, refusal := range []struct {
		name            string
		changeUpload    func(t *testing.T, source sourceArchive) []uploadPart
		release         string
		expectedFailure Failure
	}{
		{
			name: "fails with MASTER_KEY_INVALID for a key differing only by its line ending",
			changeUpload: func(t *testing.T, source sourceArchive) []uploadPart {
				t.Helper()

				return newUploadParts(t, source.path, bytes.TrimSuffix(source.keyContent, []byte("\n")))
			},
			release:         testRelease,
			expectedFailure: FailureMasterKeyInvalid,
		},
		{
			name: "fails with ARCHIVE_INCOMPLETE for a truncated archive",
			changeUpload: func(t *testing.T, source sourceArchive) []uploadPart {
				t.Helper()
				parts := newUploadParts(t, source.path, source.keyContent)
				parts[1].content = parts[1].content[:len(parts[1].content)-1]

				return parts
			},
			release:         testRelease,
			expectedFailure: FailureArchiveIncomplete,
		},
		{
			name: "fails with ARCHIVE_INVALID for a file that is not a Databasus backup",
			changeUpload: func(t *testing.T, source sourceArchive) []uploadPart {
				t.Helper()
				parts := newUploadParts(t, source.path, source.keyContent)
				parts[1].content = []byte("not a Databasus backup")

				return parts
			},
			release:         testRelease,
			expectedFailure: FailureArchiveInvalid,
		},
		{
			name: "fails with RELEASE_MISMATCH for an archive of another release",
			changeUpload: func(t *testing.T, source sourceArchive) []uploadPart {
				t.Helper()

				return newUploadParts(t, source.path, source.keyContent)
			},
			release:         "v9.9.10",
			expectedFailure: FailureReleaseMismatch,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			uploadTarget := newUploadTargetWith(t, uploadTargetSpec{
				release: refusal.release, maxArchiveBytes: defaultMaxArchiveBytes, inspectDisk: diskusage.Get,
			})
			activeKey := uploadTarget.readKey(t)

			failedRecovery := uploadTarget.uploadAndPrepare(t, refusal.changeUpload(t, source))

			assert.Equal(t, StatusFailed, failedRecovery.Status)
			require.NotNil(t, failedRecovery.Failure)
			assert.Equal(t, refusal.expectedFailure, *failedRecovery.Failure)
			uploadTarget.assertNothingStaged(t, activeKey)
			assert.NotContains(t, uploadTarget.getLogs(), strings.TrimSpace(string(source.keyContent)))
		})
	}

	t.Run("expires a prepared upload after an hour without apply and releases it", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))

		uploadTarget.clock.Advance(time.Hour - time.Second)
		uploadTarget.runPass(t)
		stillPreparedRecovery := uploadTarget.getRecovery(t, preparedRecovery)
		uploadTarget.clock.Advance(time.Second)
		uploadTarget.runPass(t)

		expiredRecovery := uploadTarget.getRecovery(t, preparedRecovery)
		assert.Equal(t, StatusReady, stillPreparedRecovery.Status)
		assert.Equal(t, StatusFailed, expiredRecovery.Status)
		require.NotNil(t, expiredRecovery.Failure)
		assert.Equal(t, FailureExpired, *expiredRecovery.Failure)
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("records a prepared upload as interrupted at the next start and releases it", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))

		err := uploadTarget.openForServer(t, uploadTarget.newStartup(t, sourceArchive{}))

		require.NoError(t, err)
		interruptedRecovery := uploadTarget.getRecovery(t, preparedRecovery)
		assert.Equal(t, StatusFailed, interruptedRecovery.Status)
		require.NotNil(t, interruptedRecovery.Failure)
		assert.Equal(t, FailureInterrupted, *interruptedRecovery.Failure)
		uploadTarget.assertNothingStaged(t, activeKey)
	})
}

func TestUploadCancellation(t *testing.T) {
	t.Parallel()

	t.Run("cancels a prepared upload, releasing its staging database and files", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))

		cancelledRecovery, err := uploadTarget.service.cancel(t.Context(), preparedRecovery.ID)
		require.NoError(t, err)
		repeatedRecovery, err := uploadTarget.service.cancel(t.Context(), preparedRecovery.ID)
		require.NoError(t, err)

		assert.Equal(t, StatusCancelled, cancelledRecovery.Status)
		assert.Nil(t, cancelledRecovery.Failure)
		assert.Equal(t, cancelledRecovery, repeatedRecovery)
		assert.Equal(t, cancelledRecovery, uploadTarget.getRecovery(t, preparedRecovery))
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("cancels an upload still validating, so the preparation never runs", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)
		uploadedRecovery, err := uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))
		require.NoError(t, err)

		cancelledRecovery, err := uploadTarget.service.cancel(t.Context(), uploadedRecovery.ID)
		require.NoError(t, err)
		uploadTarget.runPass(t)

		assert.Equal(t, StatusCancelled, cancelledRecovery.Status)
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("ends cancelled with nothing staged when cancelled while the preparation runs", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)
		uploadedRecovery, err := uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))
		require.NoError(t, err)
		passErrors := make(chan error, 1)

		go func() { passErrors <- uploadTarget.service.RunPass(t.Context(), uploadTarget.clock.Now()) }()
		cancelledRecovery, err := uploadTarget.service.cancel(t.Context(), uploadedRecovery.ID)
		require.NoError(t, err)
		require.NoError(t, <-passErrors)

		assert.Equal(t, StatusCancelled, cancelledRecovery.Status)
		assert.Equal(t, StatusCancelled, uploadTarget.getRecovery(t, uploadedRecovery).Status)
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("refuses to cancel once apply was accepted and keeps the operation", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))
		journal := uploadTarget.readJournal(t)
		journal.Operation.IsApplyAccepted = true
		require.NoError(t, NewJournalStore(uploadTarget.getDirectory()).Write(journal))

		_, err := uploadTarget.service.cancel(t.Context(), preparedRecovery.ID)

		require.ErrorIs(t, err, ErrRecoveryNotCancellable)
		assert.Equal(t, StatusApplying, uploadTarget.getRecovery(t, preparedRecovery).Status)
	})
}

func TestUploadReceiving(t *testing.T) {
	t.Parallel()

	t.Run("refuses a second upload while one is pending", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		firstRecovery, err := uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))
		require.NoError(t, err)

		_, err = uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))

		require.ErrorIs(t, err, ErrRecoveryAlreadyRunning)
		assert.Equal(t, firstRecovery.ID, uploadTarget.readJournal(t).Operation.ID)
	})

	for _, refusal := range []struct {
		name          string
		changeParts   func(parts []uploadPart) []uploadPart
		expectedField string
	}{
		{
			name:          "refuses an upload without an archive",
			changeParts:   func(parts []uploadPart) []uploadPart { return parts[:1] },
			expectedField: archiveField,
		},
		{
			name:          "refuses an upload without a master key",
			changeParts:   func(parts []uploadPart) []uploadPart { return parts[1:] },
			expectedField: masterKeyField,
		},
		{
			name: "refuses an upload with an empty master key",
			changeParts: func(parts []uploadPart) []uploadPart {
				parts[0].content = []byte{}

				return parts
			},
			expectedField: masterKeyField,
		},
		{
			name: "refuses a master key longer than 64 KiB",
			changeParts: func(parts []uploadPart) []uploadPart {
				parts[0].content = bytes.Repeat([]byte("k"), maxMasterKeyBytes+1)

				return parts
			},
			expectedField: masterKeyField,
		},
		{
			name: "refuses a master key shorter than a Databasus key",
			changeParts: func(parts []uploadPart) []uploadPart {
				parts[0].content = []byte("short")

				return parts
			},
			expectedField: masterKeyField,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			uploadTarget := newUploadTarget(t)
			activeKey := uploadTarget.readKey(t)

			_, err := uploadTarget.upload(t, refusal.changeParts(newUploadParts(t, source.path, source.keyContent)))

			requireFieldProblem(t, err, refusal.expectedField)
			uploadTarget.assertNothingStaged(t, activeKey)
			assert.NotContains(t, err.Error(), strings.TrimSpace(string(source.keyContent)))
		})
	}

	t.Run("refuses an archive beyond the limit while streaming it", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		archiveInfo, err := os.Stat(source.path)
		require.NoError(t, err)
		uploadTarget := newUploadTargetWith(t, uploadTargetSpec{
			release: testRelease, maxArchiveBytes: archiveInfo.Size() - 1, inspectDisk: diskusage.Get,
		})
		activeKey := uploadTarget.readKey(t)

		_, err = uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))

		require.ErrorIs(t, err, ErrArchiveTooLarge)
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("accepts an archive of exactly the limit", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		archiveInfo, err := os.Stat(source.path)
		require.NoError(t, err)
		uploadTarget := newUploadTargetWith(t, uploadTargetSpec{
			release: testRelease, maxArchiveBytes: archiveInfo.Size(), inspectDisk: diskusage.Get,
		})

		uploadedRecovery, err := uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))

		require.NoError(t, err)
		assert.Equal(t, StatusValidating, uploadedRecovery.Status)
	})

	t.Run("refuses an upload when the data volume lacks free space", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTargetWith(t, uploadTargetSpec{
			release: testRelease, maxArchiveBytes: defaultMaxArchiveBytes,
			inspectDisk: func(string) (diskusage.Usage, error) {
				return diskusage.Usage{TotalBytes: 1 << 30, UsedBytes: 1 << 30, FreeBytes: 0}, nil
			},
		})
		activeKey := uploadTarget.readKey(t)

		_, err := uploadTarget.upload(t, newUploadParts(t, source.path, source.keyContent))

		require.ErrorIs(t, err, ErrInsufficientDiskSpace)
		uploadTarget.assertNothingStaged(t, activeKey)
	})

	t.Run("releases an upload whose client disconnected in the middle of the archive", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		activeKey := uploadTarget.readKey(t)

		_, err := uploadTarget.service.receive(t.Context(),
			newInterruptedUploadBody(t, newUploadParts(t, source.path, source.keyContent)))

		requireFieldProblem(t, err, archiveField)
		uploadTarget.assertNothingStaged(t, activeKey)
	})
}
