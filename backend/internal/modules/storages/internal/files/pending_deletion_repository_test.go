package files

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertIfAbsent(t *testing.T) {
	t.Parallel()

	t.Run("registers the obligation when the name is free", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		pending, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, time.Hour))

		require.NoError(t, err)
		assert.Equal(t, 0, pending.Generation)
		assert.Equal(t, 0, pending.AttemptCount)
		assert.Equal(t, reference, pending.GetReference())
	})

	t.Run("refuses when the name is already registered", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, 0))
		require.NoError(t, err)

		_, err = files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, 0))

		assert.ErrorIs(t, err, ErrFileNameAlreadyRegistered)
	})

	t.Run("leaves no row when the transaction rolls back", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		err := files.inTransaction(t, func(ctx context.Context) error {
			_, insertErr := files.repository.InsertIfAbsent(ctx, files.newRegistration(reference, 0))
			require.NoError(t, insertErr)

			return assert.AnError
		})
		require.ErrorIs(t, err, assert.AnError)

		_, isPending := files.findPending(t, reference)
		assert.False(t, isPending)
	})
}

func TestInsertOrTake(t *testing.T) {
	t.Parallel()

	t.Run("keeps one row and raises the generation when requested twice", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{reference}, Scope: nil},
				files.clock.Now(),
			),
		)
		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{reference}, Scope: nil},
				files.clock.Now(),
			),
		)

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.Equal(t, 1, pending.Generation)
		assert.Equal(t, 0, pending.AttemptCount,
			"asking for a file again is not a deletion attempt and must not advance the backoff")
	})

	t.Run("does not fail when the batch repeats a reference", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		err := files.repository.InsertOrTake(
			t.Context(),
			FileDeletionRequest{References: []StoredFileReference{reference, reference}, Scope: nil},
			files.clock.Now(),
		)

		require.NoError(t, err)

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.Equal(t, 0, pending.Generation)
	})

	t.Run("does nothing when the batch is empty", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)

		assert.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: nil, Scope: nil},
				files.clock.Now(),
			),
		)
	})

	t.Run("takes the row from its writer by raising the generation", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		pending, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, time.Hour))
		require.NoError(t, err)

		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{reference}, Scope: nil},
				files.clock.Now(),
			),
		)

		isCompleted, err := files.repository.CompleteIfGeneration(t.Context(), pending.ID, pending.Generation)

		require.NoError(t, err)
		assert.False(t, isCompleted, "the writer's receipt must not release an obligation someone else took")
	})
}

func TestCompleteIfGeneration(t *testing.T) {
	t.Parallel()

	t.Run("releases the obligation when the generation matches", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		pending, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, 0))
		require.NoError(t, err)

		isCompleted, err := files.repository.CompleteIfGeneration(t.Context(), pending.ID, pending.Generation)

		require.NoError(t, err)
		assert.True(t, isCompleted)

		_, isPending := files.findPending(t, reference)
		assert.False(t, isPending)
	})
}

func TestRescheduleIfGeneration(t *testing.T) {
	t.Parallel()

	t.Run("changes nothing when the generation is stale", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		pending, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, 0))
		require.NoError(t, err)

		isRescheduled, err := files.repository.RescheduleIfGeneration(t.Context(), Rescheduling{
			PendingDeletionID: pending.ID,
			Generation:        pending.Generation + 1,
			NotBefore:         files.clock.Now().Add(time.Hour),
			Now:               files.clock.Now(),
		}, "stale")

		require.NoError(t, err)
		assert.False(t, isRescheduled)

		storedPending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.Nil(t, storedPending.LastError)
	})
}

func TestClaimDue(t *testing.T) {
	t.Parallel()

	t.Run("returns nothing when nothing is due", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.repository.InsertIfAbsent(t.Context(), files.newRegistration(reference, time.Hour))
		require.NoError(t, err)

		claimed, err := files.repository.ClaimDue(t.Context(), files.newClaimRequest(nil))

		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("takes a due row and raises its generation", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{reference}, Scope: nil},
				files.clock.Now(),
			),
		)

		claimed, err := files.repository.ClaimDue(t.Context(), files.newClaimRequest(nil))

		require.NoError(t, err)
		require.Equal(t, []StoredFileReference{reference}, referencesOf(claimed))

		pending, isPending := files.findPending(t, reference)
		require.True(t, isPending)
		assert.Equal(t, 1, pending.Generation)
		assert.Zero(t, pending.AttemptCount)
		assert.False(t, files.isDue(t, reference), "a claimed row is not due again until its lease runs out")
	})

	t.Run("skips a live write", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{reference}, Scope: nil},
				files.clock.Now(),
			),
		)

		live, _ := files.findPending(t, reference)

		claimed, err := files.repository.ClaimDue(t.Context(), files.newClaimRequest([]uuid.UUID{live.ID}))

		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("skips a row another transaction holds instead of waiting for it", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		storageID := files.createStorage(t)
		held := StoredFileReference{StorageID: storageID, FileName: "held"}
		free := StoredFileReference{StorageID: storageID, FileName: "free"}
		require.NoError(
			t,
			files.repository.InsertOrTake(
				t.Context(),
				FileDeletionRequest{References: []StoredFileReference{held, free}, Scope: nil},
				files.clock.Now(),
			),
		)

		isHeld := make(chan struct{})
		release := make(chan struct{})
		isReleased := make(chan struct{})

		go func() {
			defer close(isReleased)

			_ = files.inTransaction(t, func(ctx context.Context) error {
				if err := files.repository.InsertOrTake(
					ctx,
					FileDeletionRequest{References: []StoredFileReference{held}, Scope: nil},
					files.clock.Now(),
				); err != nil {
					return err
				}

				close(isHeld)
				<-release

				return nil
			})
		}()

		<-isHeld

		claimCtx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		claimed, err := files.repository.ClaimDue(claimCtx, files.newClaimRequest(nil))

		close(release)
		<-isReleased

		require.NoError(t, err, "the claim must skip a locked row rather than wait for it")
		assert.Equal(t, []StoredFileReference{free}, referencesOf(claimed))
	})
}

func (f *testFiles) newRegistration(reference StoredFileReference, dueIn time.Duration) Registration {
	return Registration{
		ID:        uuid.New(),
		Reference: reference,
		NotBefore: f.clock.Now().Add(dueIn),
		Now:       f.clock.Now(),
	}
}

func (f *testFiles) newClaimRequest(excludedIDs []uuid.UUID) ClaimRequest {
	return ClaimRequest{Limit: 100, AttemptLease: time.Minute, ExcludedIDs: excludedIDs, Now: f.clock.Now()}
}
