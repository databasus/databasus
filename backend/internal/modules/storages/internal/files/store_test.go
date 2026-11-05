package files

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	t.Parallel()

	t.Run("returns a receipt and holds the obligation when the provider accepts the file", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.NoError(t, err)
		assert.Equal(t, reference, receipt.Reference)
		assert.Equal(t, 0, receipt.Generation)
		assert.Equal(t, []string{"backup-1"}, files.provider.getSaved())

		_, isPending := files.findPending(t, reference)
		require.True(t, isPending, "the obligation stands until a catalog transaction claims it")
		assert.False(t, files.isDue(t, reference), "a fresh upload gets its commit window before cleanup may act")
		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("sends no bytes when the name is already registered", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		files.requestDeletions(t, reference)

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.ErrorIs(t, err, ErrFileNameAlreadyRegistered)
		assert.Empty(t, files.provider.getSaved())
		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("leaves the file eligible at once when the provider fails", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.saveErr = errors.New("provider is down")
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.ErrorIs(t, err, files.provider.saveErr)

		_, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.True(t, files.isDue(t, reference))
		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("keeps the row out of reach of cleanup while it runs", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		var liveWriteCountDuringWrite int

		files.provider.onSave = func() {
			files.requestDeletions(t, reference)

			liveWriteCountDuringWrite = files.getLiveWriteCount()
		}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.ErrorIs(t, err, ErrFileTakenForDeletion)
		assert.Equal(t, 1, liveWriteCountDuringWrite, "the write is excluded from cleanup while it runs")
		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("leaves the live set when the provider panics", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.savePanics = true
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		require.Panics(t, func() {
			_, _ = files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		})

		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("leaves the file eligible when the storage cannot be located", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.locator.err = errors.New("no such storage")
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.ErrorIs(t, err, files.locator.err)

		_, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.True(t, files.isDue(t, reference))
	})

	t.Run("fails without sending bytes when the storage was deleted", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		files.execute(t, "DELETE FROM storages WHERE id = ?", storageID)
		reference := StoredFileReference{StorageID: storageID, FileName: "backup-1"}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.Error(t, err)
		assert.Empty(t, files.provider.getSaved())
		assert.Zero(t, files.getLiveWriteCount())
	})

	t.Run("keeps the obligation when the caller's transaction rolls back", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		err := files.inTransaction(t, func(ctx context.Context) error {
			if _, writeErr := files.store.WriteFile(ctx, reference, strings.NewReader("payload")); writeErr != nil {
				return writeErr
			}

			return assert.AnError
		})
		require.ErrorIs(t, err, assert.AnError)

		assert.Equal(t, []string{"backup-1"}, files.provider.getSaved())

		_, isPending := files.findPending(t, reference)
		assert.True(t, isPending, "an uploaded file keeps its obligation when the caller's transaction rolls back")
	})

	// A process that stops between the upload and the catalog commit leaves a receipt nobody will
	// ever spend, which is indistinguishable from a caller that never published. Both end with the
	// file gone once the commit window closes.
	t.Run("removes an unpublished file after its commit window and keeps a confirmed one", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		unpublished := StoredFileReference{StorageID: storageID, FileName: "unpublished"}
		published := StoredFileReference{StorageID: storageID, FileName: "published"}

		_, err := files.store.WriteFile(t.Context(), unpublished, strings.NewReader("payload"))
		require.NoError(t, err)

		publishedReceipt, err := files.store.WriteFile(t.Context(), published, strings.NewReader("payload"))
		require.NoError(t, err)

		require.NoError(t, files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{publishedReceipt})
		}))

		files.runPass(t)
		assert.Empty(t, files.provider.getDeleted(), "a file inside its commit window is not cleanup's yet")

		files.clock.Advance(testTimings().CommitWindow)
		files.runPass(t)

		assert.Equal(t, []string{"unpublished"}, files.provider.getDeleted(),
			"the published file carries no obligation, so nothing can delete it")
	})

	t.Run("becomes due only once the clock passes the commit window", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		files.clock.Advance(testTimings().CommitWindow - time.Second)
		files.runPass(t)

		assert.False(t, files.isDue(t, reference))
		assert.Empty(t, files.provider.getDeleted(), "a second before the window closes the file is not cleanup's yet")

		files.clock.Advance(time.Second)
		files.runPass(t)

		assert.Equal(t, []string{"backup-1"}, files.provider.getDeleted())
	})
}

func TestConfirmFileWrites(t *testing.T) {
	t.Parallel()

	t.Run("refuses when the context carries no transaction", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		err = files.store.ConfirmFileWrites(t.Context(), []WriteReceipt{receipt})

		assert.ErrorIs(t, err, ErrTransactionRequired)
	})

	t.Run("keeps every file when the transaction commits", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		backup := StoredFileReference{StorageID: storageID, FileName: "backup-1"}
		sidecar := StoredFileReference{StorageID: storageID, FileName: "backup-1.metadata"}

		backupReceipt, err := files.store.WriteFile(t.Context(), backup, strings.NewReader("payload"))
		require.NoError(t, err)

		sidecarReceipt, err := files.store.WriteFile(t.Context(), sidecar, strings.NewReader("meta"))
		require.NoError(t, err)

		require.NoError(t, files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{backupReceipt, sidecarReceipt})
		}))

		for _, reference := range []StoredFileReference{backup, sidecar} {
			_, isPending := files.findPending(t, reference)
			assert.False(t, isPending, "a published file carries no obligation")
		}
	})

	t.Run("keeps the obligation when the transaction rolls back", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		err = files.inTransaction(t, func(ctx context.Context) error {
			if confirmErr := files.store.ConfirmFileWrites(ctx, []WriteReceipt{receipt}); confirmErr != nil {
				return confirmErr
			}

			return assert.AnError
		})
		require.ErrorIs(t, err, assert.AnError)

		_, isPending := files.findPending(t, reference)
		assert.True(t, isPending, "an uncommitted publication leaves the file for cleanup")
	})

	t.Run("fails the second time a receipt is spent", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		require.NoError(t, files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{receipt})
		}))

		err = files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{receipt})
		})

		assert.ErrorIs(t, err, ErrReceiptNoLongerValid)
	})

	t.Run("fails when a deletion was requested meanwhile", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		receipt, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		files.requestDeletions(t, reference)

		err = files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{receipt})
		})

		require.ErrorIs(t, err, ErrReceiptNoLongerValid)

		_, isPending := files.findPending(t, reference)
		assert.True(t, isPending)
	})

	t.Run("fails the whole batch when one receipt is stale", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		valid := StoredFileReference{StorageID: storageID, FileName: "backup-1"}
		stale := StoredFileReference{StorageID: storageID, FileName: "backup-1.metadata"}

		validReceipt, err := files.store.WriteFile(t.Context(), valid, strings.NewReader("payload"))
		require.NoError(t, err)

		staleReceipt, err := files.store.WriteFile(t.Context(), stale, strings.NewReader("meta"))
		require.NoError(t, err)

		files.requestDeletions(t, stale)

		err = files.inTransaction(t, func(ctx context.Context) error {
			return files.store.ConfirmFileWrites(ctx, []WriteReceipt{validReceipt, staleReceipt})
		})

		require.ErrorIs(t, err, ErrReceiptNoLongerValid)

		_, isPending := files.findPending(t, valid)
		assert.True(t, isPending, "a failed batch leaves every file in the batch for cleanup")
	})
}

func TestRequestFileDeletions(t *testing.T) {
	t.Parallel()

	t.Run("refuses when the context carries no transaction", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		references := []StoredFileReference{{StorageID: files.createStorage(t), FileName: "backup-1"}}

		err := files.store.RequestFileDeletions(t.Context(), references)

		assert.ErrorIs(t, err, ErrTransactionRequired)
	})

	t.Run("keeps one obligation when repeated", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		files.requestDeletions(t, reference)
		files.requestDeletions(t, reference)

		pendingDeletions, err := files.repository.ListByStorage(t.Context(), reference.StorageID)
		require.NoError(t, err)
		assert.Len(t, pendingDeletions, 1)
	})

	t.Run("succeeds while a writer is running and takes the file from it", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		var requestErr error

		files.provider.onSave = func() {
			requestErr = files.inTransaction(t, func(ctx context.Context) error {
				return files.store.RequestFileDeletions(ctx, []StoredFileReference{reference})
			})
		}

		_, writeErr := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

		require.NoError(t, requestErr, "a cascade cannot wait for an upload it does not own")
		require.ErrorIs(t, writeErr, ErrFileTakenForDeletion)

		_, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.True(t, files.isDue(t, reference), "the writer must not push a file that was taken from it further out")
	})

	t.Run("skips a reference without a file name", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		unnamed := StoredFileReference{StorageID: files.createStorage(t), FileName: ""}

		files.requestDeletions(t, unnamed)

		_, isPending := files.findPending(t, unnamed)
		assert.False(t, isPending)
	})

	t.Run("succeeds when the batch is empty", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)

		files.requestDeletions(t)
	})
}
