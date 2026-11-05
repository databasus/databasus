package files

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/testwait"
)

func TestRunPass(t *testing.T) {
	t.Parallel()

	t.Run("drains several batches before the periodic tick when cleanup is requested", func(t *testing.T) {
		t.Parallel()

		files := newTestFiles(t)
		storageID := files.createStorage(t)
		references := make([]StoredFileReference, 0, 2*testTimings().ClaimBatchSize+1)
		for index := range cap(references) {
			references = append(references, StoredFileReference{
				StorageID: storageID,
				FileName:  fmt.Sprintf("backup-%d.dump", index),
			})
		}
		files.requestDeletions(t, references...)
		runner := background.NewRunner(background.RunnerDeps{
			Logger: slog.New(slog.DiscardHandler),
			Clock:  files.clock,
		}, files.job.GetJob())
		jobsCtx, cancelJobs := context.WithCancel(t.Context())
		runner.Start(jobsCtx)
		t.Cleanup(func() {
			cancelJobs()
			require.NoError(t, runner.Wait(context.WithoutCancel(t.Context())))
		})

		testwait.Within(5*time.Second).
			Until(t, "all cleanup batches finish before the periodic tick", func(context.Context) error {
				if len(files.provider.getDeleted()) != len(references) {
					return fmt.Errorf("deleted %d of %d files", len(files.provider.getDeleted()), len(references))
				}

				return nil
			})
	})

	t.Run("counts only dispatched attempts when the first file exhausts a batch lease", func(t *testing.T) {
		t.Parallel()

		files := newTestFiles(t)
		storageID := files.createStorage(t)
		references := []StoredFileReference{
			{StorageID: storageID, FileName: "first.dump"},
			{StorageID: storageID, FileName: "second.dump"},
		}
		files.requestDeletions(t, references...)
		files.provider.setDeleteErr(context.DeadlineExceeded)
		files.provider.onDelete = func() { files.clock.Advance(testTimings().AttemptLease) }

		files.runPass(t)

		var skippedReference StoredFileReference
		for _, reference := range references {
			pending, isPending := files.findPending(t, reference)
			require.True(t, isPending)

			if pending.LastError == nil {
				skippedReference = reference
				assert.Zero(t, pending.AttemptCount)
			} else {
				assert.Equal(t, 1, pending.AttemptCount)
			}
		}
		require.NotEmpty(t, skippedReference.FileName)
		files.provider.onDelete = nil
		files.clock.Advance(testTimings().RetryBaseDelay)

		files.runPass(t)

		pending, isPending := files.findPending(t, skippedReference)
		require.True(t, isPending)
		require.NotNil(t, pending.LastError)
		assert.Equal(t, 1, pending.AttemptCount)
		assert.Equal(t, files.clock.Now().Add(testTimings().RetryBaseDelay), pending.NotBefore.UTC())
	})

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

	t.Run(
		"redacts signed URL credentials from persisted errors and logs when a provider refuses deletion",
		func(t *testing.T) {
			t.Parallel()

			files := newTestFiles(t)
			reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "signed.dump"}
			files.requestDeletions(t, reference)
			files.provider.setDeleteErr(
				errors.New(
					"delete https://blob.example.com/backups/signed.dump?sv=2026-01-01&sig=azureSignature%2Bsecret failed; https://bucket.example.com/signed.dump?X-Amz-Credential=awsCredential%2Fscope&X-Amz-Signature=awsSignature&X-Amz-Security-Token=awsSession failed",
				),
			)

			files.runPass(t)

			pending, isPending := files.findPending(t, reference)
			require.True(t, isPending)
			require.NotNil(t, pending.LastError)
			attempt, isLogged := files.logs.findRecord("file deletion attempt failed, retrying")
			require.True(t, isLogged)
			for _, credential := range []string{"azureSignature", "awsCredential", "awsSignature", "awsSession"} {
				assert.NotContains(t, *pending.LastError, credential)
				assert.NotContains(t, attempt["error"], credential)
			}
			assert.Contains(t, *pending.LastError, "blob.example.com")
			assert.Contains(t, attempt["error"], "bucket.example.com")
		},
	)

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

func TestGetRetryDelay(t *testing.T) {
	t.Parallel()

	t.Run("grows exponentially and stops at the maximum", func(t *testing.T) {
		t.Parallel()
		job := NewDeletionJob(nil, Deps{
			Repository:      nil,
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
