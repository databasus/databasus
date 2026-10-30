package files

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossInstanceWriteProtection(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"active upload", "canceled upload", "elapsed cleanup deadline", "storage drain"} {
		t.Run("protects a peer's unfinished write with "+scenario, func(t *testing.T) {
			t.Parallel()
			files := newTestFiles(t)
			peerDeps := Deps{
				Repository: files.repository, Batches: NewDeletionBatchRepository(files.db),
				WriteProtection: NewWriteProtectionRepository(files.db),
				Transactor:      files.transactor, Locator: files.locator, Logger: files.job.logger,
				Clock: files.clock, Timings: testTimings(), Uploads: files.uploads,
			}
			peer := NewDeletionJob(NewStore(peerDeps), peerDeps)
			reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "peer-upload.dump"}
			ctx, cancelWrite := context.WithCancel(t.Context())
			defer cancelWrite()
			var batch DeletionBatch
			var terminal DeletionBatch
			files.provider.onSave = func() {
				batch = files.requestBatch(t, reference)
				files.requestDeletions(t, reference)
				if scenario == "canceled upload" {
					cancelWrite()
				}
				if scenario == "elapsed cleanup deadline" {
					files.clock.Advance(24 * time.Hour)
				}
				if scenario == "storage drain" {
					request, err := peer.PrepareDrain(t.Context(), reference.StorageID, time.Minute)
					require.NoError(t, err)
					assert.Equal(t, []string{reference.FileName}, peer.DrainFiles(t.Context(), request))
				}
				require.NoError(t, peer.RunPass(t.Context(), files.clock.Now()))
				assert.Empty(t, files.provider.getDeleted())
				terminal = files.getBatch(t, batch.ID)
				if scenario == "elapsed cleanup deadline" {
					assert.Equal(t, CleanupWithRemainders, terminal.Status)
				} else {
					assert.Equal(t, CleanupPending, terminal.Status)
				}
				var failedAttempts int
				require.NoError(t, files.db.WithContext(t.Context()).Raw(
					"SELECT failed_attempts FROM storages_deletion_batch_items WHERE batch_id=?", batch.ID).
					Scan(&failedAttempts).Error)
				assert.Zero(t, failedAttempts)
			}
			_, writeErr := files.store.WriteFile(ctx, reference, strings.NewReader("payload"))
			require.Error(t, writeErr)
			require.NoError(t, peer.RunPass(t.Context(), files.clock.Now()))
			assert.Equal(t, []string{reference.FileName}, files.provider.getDeleted())
			if scenario == "elapsed cleanup deadline" {
				assert.Equal(t, terminal, files.getBatch(t, batch.ID))
			} else {
				assert.Equal(t, CleanupCompleted, files.getBatch(t, batch.ID).Status)
			}
		})
	}
}

func TestWriteProtectionConnection(t *testing.T) {
	t.Parallel()
	t.Run("rejects receipts and recovers capacity when the shared session disconnects", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		startedWrites := 0
		files.provider.onSave = func() {
			startedWrites++
			if startedWrites == 1 {
				_, err := files.store.WriteFile(t.Context(), StoredFileReference{
					StorageID: storageID, FileName: "interrupted-second.dump",
				}, strings.NewReader("second"))
				require.ErrorIs(t, err, ErrWriteProtectionLost)
			} else {
				var sessionPID int
				require.NoError(t, files.store.writeProtection.connection.QueryRowContext(
					t.Context(), "SELECT pg_backend_pid()").Scan(&sessionPID))
				files.execute(t, "SELECT pg_terminate_backend(?)", sessionPID)
			}
		}
		_, err := files.store.WriteFile(t.Context(), StoredFileReference{
			StorageID: storageID, FileName: "interrupted-first.dump",
		}, strings.NewReader("first"))
		require.ErrorIs(t, err, ErrWriteProtectionLost)
		assert.Empty(t, files.store.writeProtection.activeWrites)
		assert.Nil(t, files.store.writeProtection.connection)
		files.provider.onSave = nil
		_, err = files.store.WriteFile(t.Context(), StoredFileReference{
			StorageID: storageID, FileName: "after-disconnect.dump",
		}, strings.NewReader("recovered"))
		require.NoError(t, err)
	})
	t.Run("shares one connection across overlapping writes and returns it after completion", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		startedWrites := 0
		files.provider.onSave = func() {
			startedWrites++
			if startedWrites == 1 {
				_, err := files.store.WriteFile(t.Context(), StoredFileReference{
					StorageID: storageID, FileName: "second.dump",
				}, strings.NewReader("second"))
				require.NoError(t, err)
			} else {
				assert.Len(t, files.store.writeProtection.activeWrites, 2)
				pool, err := files.db.DB()
				require.NoError(t, err)
				assert.Equal(t, 1, pool.Stats().InUse)
			}
		}
		_, err := files.store.WriteFile(t.Context(), StoredFileReference{
			StorageID: storageID, FileName: "first.dump",
		}, strings.NewReader("first"))
		require.NoError(t, err)
		assert.Empty(t, files.store.writeProtection.activeWrites)
		assert.Nil(t, files.store.writeProtection.connection)
	})
}
