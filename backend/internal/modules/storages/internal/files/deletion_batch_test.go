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

	"databasus-backend/internal/lib/logging"
)

func (f *testFiles) requestBatch(t *testing.T, references ...StoredFileReference) DeletionBatch {
	t.Helper()
	var batch DeletionBatch
	require.NoError(t, f.inTransaction(t, func(ctx context.Context) error {
		var err error
		batch, err = f.store.RequestDeletionBatch(ctx, DeletionBatchRequest{ID: uuid.New(), References: references})
		return err
	}))
	return batch
}

func (f *testFiles) getBatch(t *testing.T, id uuid.UUID) DeletionBatch {
	t.Helper()
	batches, err := f.store.GetDeletionBatches(t.Context(), []uuid.UUID{id})
	require.NoError(t, err)
	require.Len(t, batches, 1)
	batch := batches[0]
	assert.Equal(t, batch.Total, batch.Pending+batch.Deleted+batch.Remaining)
	assert.Equal(t, batch.Pending == 0, batch.CompletedAt != nil)
	return batch
}

func TestDeletionBatch(t *testing.T) {
	t.Parallel()
	t.Run(
		"counts failures when repeated requests supersede the queue generation before lease expiry",
		func(t *testing.T) {
			t.Parallel()
			files := newTestFiles(t)
			reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "repeated.dump"}
			batch := files.requestBatch(t, reference)

			files.provider.setDeleteErr(errors.New("provider unavailable"))
			files.provider.onDelete = func() { files.requestDeletions(t, reference) }

			for attempt := range 5 {
				files.runPass(t)
				var failedAttempts int
				require.NoError(t, files.db.WithContext(t.Context()).Raw(
					"SELECT failed_attempts FROM storages_deletion_batch_items WHERE batch_id=?", batch.ID).
					Scan(&failedAttempts).Error)
				assert.Equal(t, attempt+1, failedAttempts)
				files.clock.Advance(time.Second)
			}

			terminal := files.getBatch(t, batch.ID)
			assert.Equal(t, CleanupWithRemainders, terminal.Status)
			assert.EqualValues(t, 1, terminal.Remaining)
		},
	)
	t.Run("keeps an unsettled attempt when a repeated request starts another worker pass", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "active.dump"}
		batch := files.requestBatch(t, reference)

		files.provider.setDeleteErr(errors.New("provider unavailable"))
		providerCalls := 0
		files.provider.onDelete = func() {
			providerCalls++
			if providerCalls == 1 {
				files.requestDeletions(t, reference)
				files.runPass(t)
			}
		}

		files.runPass(t)
		assert.Equal(t, 1, providerCalls)

		var failedAttempts int
		require.NoError(t, files.db.WithContext(t.Context()).Raw(
			"SELECT failed_attempts FROM storages_deletion_batch_items WHERE batch_id=?", batch.ID).
			Scan(&failedAttempts).Error)
		assert.Equal(t, 1, failedAttempts)
		assert.Equal(t, CleanupPending, files.getBatch(t, batch.ID).Status)
	})
	t.Run("preserves three failed attempts when the worker restarts", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference)
		files.provider.setDeleteErr(errors.New("unreachable"))

		for range 3 {
			files.runPass(t)
			files.clock.Advance(time.Minute)
		}

		files.job = NewDeletionJob(
			files.store,
			Deps{
				Repository:      files.repository,
				Batches:         NewDeletionBatchRepository(files.db),
				WriteProtection: files.store.writeProtection,
				Transactor:      files.transactor,
				Locator:         files.locator,
				Logger:          logging.NewWithSinks(files.logs).GetLogger(),
				Clock:           files.clock,
				Timings:         testTimings(),
				Uploads:         files.uploads,
			},
		)

		for range 2 {
			files.runPass(t)
			files.clock.Advance(time.Minute)
		}

		terminal := files.getBatch(t, batch.ID)
		assert.Equal(t, CleanupWithRemainders, terminal.Status)
		assert.True(t, batch.CleanupDeadline.Equal(terminal.CleanupDeadline))
	})
	t.Run("bounds a provider call by the remaining cleanup deadline", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference)
		files.clock.Advance(24*time.Hour - time.Second)

		files.provider.onDeleteContext = func(ctx context.Context) {
			deadline, hasDeadline := ctx.Deadline()
			require.True(t, hasDeadline)
			assert.LessOrEqual(t, time.Until(deadline), time.Second)
			files.clock.Advance(time.Second)
		}

		files.runPass(t)
		assert.Equal(t, CleanupPending, files.getBatch(t, batch.ID).Status)
		files.runPass(t)
		assert.Equal(t, CleanupWithRemainders, files.getBatch(t, batch.ID).Status)
		assert.Len(t, files.provider.getDeleted(), 1)
	})
	t.Run("completes an empty batch without provider calls", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		batch := files.requestBatch(t)
		assert.Equal(t, CleanupCompleted, batch.Status)
		assert.Equal(t, batch, files.getBatch(t, batch.ID))
		assert.Empty(t, files.provider.getDeleted())
	})
	t.Run("counts each confirmed absence once after deduplicating requests", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference, reference)
		files.runPass(t)
		files.runPass(t)
		completed := files.getBatch(t, batch.ID)
		assert.Equal(t, CleanupCompleted, completed.Status)
		assert.EqualValues(t, 1, completed.Deleted)
		assert.Equal(t, []string{reference.FileName}, files.provider.getDeleted())
	})
	t.Run("preserves the captured set and deadline when a batch request is repeated", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference)
		files.clock.Advance(time.Hour)

		require.NoError(t, files.inTransaction(t, func(ctx context.Context) error {
			repeated, err := files.store.RequestDeletionBatch(
				ctx,
				DeletionBatchRequest{
					ID:         batch.ID,
					References: []StoredFileReference{{StorageID: reference.StorageID, FileName: "new.dump"}},
				},
			)
			assert.Equal(t, batch, repeated)
			return err
		}))

		files.runPass(t)
		assert.Equal(t, []string{reference.FileName}, files.provider.getDeleted())
	})
	t.Run("stops after five failures while an independent obligation keeps retrying", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		files.requestDeletions(t, reference)
		batch := files.requestBatch(t, reference)
		files.provider.setDeleteErr(errors.New("provider unavailable"))

		for range 5 {
			files.runPass(t)
			files.clock.Advance(time.Minute)
		}

		terminal := files.getBatch(t, batch.ID)
		assert.Equal(t, CleanupWithRemainders, terminal.Status)
		assert.EqualValues(t, 1, terminal.Remaining)
		_, isPending := files.findPending(t, reference)
		require.True(t, isPending)

		files.provider.setDeleteErr(nil)
		files.runPass(t)
		assert.Equal(t, terminal, files.getBatch(t, batch.ID))
		_, isPending = files.findPending(t, reference)
		assert.False(t, isPending)
	})
	t.Run("releases switch ownership after five failed attempts", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference)
		files.provider.setDeleteErr(errors.New("provider unavailable"))

		for range 5 {
			files.runPass(t)
			files.clock.Advance(time.Minute)
		}

		terminal := files.getBatch(t, batch.ID)
		assert.Equal(t, CleanupWithRemainders, terminal.Status)
		_, isPending := files.findPending(t, reference)
		assert.False(t, isPending)

		files.provider.setDeleteErr(nil)
		files.runPass(t)
		assert.Empty(t, files.provider.getDeleted())
	})
	t.Run("reports a live writer unresolved exactly at the deadline", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "live.dump"}
		var batch DeletionBatch

		files.provider.onSave = func() {
			batch = files.requestBatch(t, reference)
			files.clock.Advance(24*time.Hour - time.Nanosecond)
			files.runPass(t)
			assert.Equal(t, CleanupPending, files.getBatch(t, batch.ID).Status)
			files.clock.Advance(time.Nanosecond)
			files.runPass(t)
			assert.Equal(t, CleanupWithRemainders, files.getBatch(t, batch.ID).Status)
			assert.Empty(t, files.provider.getDeleted())
		}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.ErrorIs(t, err, ErrFileTakenForDeletion)

		terminal := files.getBatch(t, batch.ID)
		files.runPass(t)
		assert.Equal(t, []string{reference.FileName}, files.provider.getDeleted())
		assert.Equal(t, terminal, files.getBatch(t, batch.ID))
	})
	t.Run("ignores a late success after its lease expires", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "old.dump"}
		batch := files.requestBatch(t, reference)
		files.provider.onDelete = func() { files.clock.Advance(testTimings().AttemptLease) }
		files.runPass(t)
		assert.Equal(t, CleanupPending, files.getBatch(t, batch.ID).Status)

		files.provider.onDelete = nil
		files.runPass(t)
		assert.Equal(t, CleanupCompleted, files.getBatch(t, batch.ID).Status)
	})
}
