package files

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPass(t *testing.T) {
	t.Parallel()
	t.Run("persists timeout failures and backs off ordinary deletions", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "timed-out.dump"}
		files.requestDeletions(t, reference)
		files.provider.onDelete = func() { files.clock.Advance(testTimings().AttemptLease) }
		files.provider.setDeleteErr(context.DeadlineExceeded)

		files.runPass(t)

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		require.NotNil(t, pending.LastError)
		assert.Equal(t, context.DeadlineExceeded.Error(), *pending.LastError)
		assert.Equal(t, files.clock.Now().Add(testTimings().RetryBaseDelay), pending.NotBefore.UTC())
		assert.Equal(t, 1, pending.AttemptCount)
		files.runPass(t)
		pending, _ = files.findPending(t, reference)
		assert.Equal(t, 1, pending.AttemptCount)
		files.clock.Advance(testTimings().RetryBaseDelay)
		files.runPass(t)
		pending, _ = files.findPending(t, reference)
		assert.Equal(t, 2, pending.AttemptCount)
		assert.Equal(t, files.clock.Now().Add(2*testTimings().RetryBaseDelay), pending.NotBefore.UTC())
	})

	t.Run("deletes a due file and releases its obligation", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		files.requestDeletions(t, reference)

		files.runPass(t)

		assert.Equal(t, []string{"backup-1"}, files.provider.getDeleted())

		_, isPending := files.findPending(t, reference)
		assert.False(t, isPending)
	})

	t.Run("leaves a file alone until its writer is gone", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		var deletedDuringWrite []string

		files.provider.onSave = func() {
			files.requestDeletions(t, reference)
			files.runPass(t)

			deletedDuringWrite = files.provider.getDeleted()
		}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.ErrorIs(t, err, ErrFileTakenForDeletion)

		assert.Empty(t, deletedDuringWrite, "cleanup must not delete a file whose write is still running")

		files.runPass(t)

		assert.Equal(t, []string{"backup-1"}, files.provider.getDeleted())
	})

	t.Run("claims a write inherited from a stopped process on the first pass", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, -time.Minute))
		require.NoError(t, err)

		files.runPass(t)

		assert.Equal(t, []string{"backup-1"}, files.provider.getDeleted())
	})

	t.Run("invalidates an outstanding receipt when it claims the file", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		files.clock.Advance(testTimings().CommitWindow)
		files.runPass(t)

		err = files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{receipt})
		})

		require.ErrorIs(t, err, ErrReceiptNoLongerValid)
		assert.Equal(t, []string{"backup-1"}, files.provider.getDeleted())
	})

	t.Run("retries after a failed deletion until the file is gone", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.setDeleteErr(errors.New("storage is unavailable"))
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		files.requestDeletions(t, reference)

		files.runPass(t)

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending, "a failed deletion stays an obligation")
		assert.Equal(t, 1, pending.AttemptCount)
		require.NotNil(t, pending.LastError)
		assert.Contains(t, *pending.LastError, "storage is unavailable")

		files.provider.setDeleteErr(nil)
		files.clock.Advance(testTimings().RetryBaseDelay - time.Second)
		files.runPass(t)

		assert.Empty(t, files.provider.getDeleted(), "the retry waits for its delay")

		files.clock.Advance(time.Second)
		files.runPass(t)

		_, isPending = files.findPending(t, reference)
		assert.False(t, isPending)
	})

	t.Run("counts only the attempt when the deletion was requested repeatedly", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.setDeleteErr(errors.New("storage is unavailable"))
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		for range 3 {
			files.requestDeletions(t, reference)
		}

		files.runPass(t)

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.Equal(t, 3, pending.Generation, "each request takes the file from whoever held it")
		assert.Equal(t, 1, pending.AttemptCount,
			"three requests and one failed deletion is one attempt, so the retry waits the base delay")
	})

	t.Run("keeps the obligation that another caller took during the attempt", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		files.requestDeletions(t, reference)

		files.provider.onDelete = func() {
			files.requestDeletions(t, reference)
		}

		files.runPass(t)

		_, isPending := files.findPending(t, reference)
		assert.True(t, isPending, "an attempt that lost its claim must not release the newer one")
	})

	t.Run("finishes quietly when the storage disappears during the attempt", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		reference := StoredFileReference{StorageID: storageID, FileName: "backup-1"}
		files.requestDeletions(t, reference)

		files.provider.onDelete = func() {
			files.execute(t, "DELETE FROM storages WHERE id = ?", storageID)
		}

		files.runPass(t)

		_, isPending := files.findPending(t, reference)
		assert.False(t, isPending)
	})

	t.Run("logs the file and the retry of a failed attempt", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.setDeleteErr(errors.New("storage is unavailable"))
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		files.requestDeletions(t, reference)

		files.runPass(t)

		attempt, isLogged := files.logs.findRecord("file deletion attempt failed, retrying")
		require.True(t, isLogged)
		assert.Equal(t, reference.StorageID.String(), attempt["storage_id"])
		assert.Equal(t, reference.FileName, attempt["file_name"])
		assert.Equal(t, "1", attempt["attempt"])
		assert.Equal(t, testTimings().RetryBaseDelay.String(), attempt["retry_in"])

		summary, isLogged := files.logs.findRecord("pending file deletions remain")
		require.True(t, isLogged)
		assert.Equal(t, "1", summary["pending_count"])
	})
}

func TestDrainFiles(t *testing.T) {
	t.Parallel()

	t.Run("removes every pending file of the storage", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		files.requestDeletions(t,
			StoredFileReference{StorageID: storageID, FileName: "backup-1"},
			StoredFileReference{StorageID: storageID, FileName: "backup-2"},
		)

		remainingFileNames := files.drain(t, storageID, time.Minute)

		assert.Empty(t, remainingFileNames)
		assert.ElementsMatch(t, []string{"backup-1", "backup-2"}, files.provider.getDeleted())
	})

	t.Run("names the files it could not remove", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.setDeleteErr(errors.New("storage is unavailable"))
		storageID := files.createStorage(t)
		files.requestDeletions(t, StoredFileReference{StorageID: storageID, FileName: "backup-1"})

		remainingFileNames := files.drain(t, storageID, time.Minute)

		assert.Equal(t, []string{"backup-1"}, remainingFileNames)
	})

	t.Run("skips a file whose write is still running", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		reference := StoredFileReference{StorageID: storageID, FileName: "backup-1"}

		var remainingFileNames []string

		files.provider.onSave = func() {
			remainingFileNames = files.drain(t, storageID, time.Minute)
		}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		assert.Equal(t, []string{"backup-1"}, remainingFileNames)
		assert.Empty(t, files.provider.getDeleted(), "a deletion never pulls a file from under its writer")
	})

	t.Run("starts no file once the budget on the clock is spent", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		files.requestDeletions(t,
			StoredFileReference{StorageID: storageID, FileName: "backup-1"},
			StoredFileReference{StorageID: storageID, FileName: "backup-2"},
		)

		files.provider.onDelete = func() {
			files.clock.Advance(2 * time.Minute)
		}

		remainingFileNames := files.drain(t, storageID, time.Minute)

		assert.Len(t, files.provider.getDeleted(), 1)
		assert.Len(t, remainingFileNames, 1, "the second file starts after the budget ran out")
	})
}

func TestGetRetryDelay(t *testing.T) {
	t.Parallel()

	t.Run("grows exponentially and stops at the maximum", func(t *testing.T) {
		t.Parallel()
		job := NewDeletionJob(nil, Deps{
			Repository:      nil,
			Batches:         nil,
			WriteProtection: nil,
			Transactor:      nil,
			Locator:         nil,
			Logger:          nil,
			Clock:           nil,
			Timings:         Timings{RetryBaseDelay: 10 * time.Millisecond, RetryMaxDelay: 40 * time.Millisecond},
			Uploads:         nil,
		})
		job.jitter = func(delay time.Duration) time.Duration { return delay }

		assert.Equal(t, 10*time.Millisecond, job.getRetryDelay(1))
		assert.Equal(t, 20*time.Millisecond, job.getRetryDelay(2))
		assert.Equal(t, 40*time.Millisecond, job.getRetryDelay(3))
		assert.Equal(t, 40*time.Millisecond, job.getRetryDelay(9))
		assert.Equal(t, 40*time.Millisecond, job.getRetryDelay(500), "an overflowing exponent still lands on the cap")
	})
}

func TestSanitizeError(t *testing.T) {
	t.Parallel()

	t.Run("removes credentials", func(t *testing.T) {
		t.Parallel()

		sanitized := sanitizeError(errors.New("dial postgres://admin:hunter2@db:5432 failed"))

		assert.NotContains(t, sanitized, "hunter2")
	})

	t.Run("bounds the length", func(t *testing.T) {
		t.Parallel()

		truncated := sanitizeError(errors.New(strings.Repeat("x", maxPersistedErrorLength*2)))

		assert.Len(t, truncated, maxPersistedErrorLength)
	})
}

func (f *testFiles) drain(t *testing.T, storageID uuid.UUID, budget time.Duration) []string {
	t.Helper()

	pendingDeletions, err := f.repository.ListByStorage(t.Context(), storageID)
	require.NoError(t, err)

	located, err := f.locator.LocateStorage(t.Context(), storageID)
	require.NoError(t, err)

	return f.job.DrainFiles(t.Context(), DrainRequest{
		Storage:          located,
		PendingDeletions: pendingDeletions,
		Budget:           budget,
	})
}
