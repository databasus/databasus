package storage_files

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	db "databasus-backend/internal/storage"
)

// The rows go in through raw SQL rather than storages.CreateTestStorage: the
// storages package imports this one, so an in-package test cannot reach its
// fixtures without giving up access to the unexported protocol under test.
func createStorageRow(t *testing.T) uuid.UUID {
	t.Helper()

	workspaceID := uuid.New()
	storageID := uuid.New()

	require.NoError(t, db.GetDb().Exec(
		"INSERT INTO workspaces (id, name, created_at) VALUES (?, ?, now())",
		workspaceID, "storage-files-"+workspaceID.String(),
	).Error)

	require.NoError(t, db.GetDb().Exec(
		"INSERT INTO storages (id, workspace_id, type, name) VALUES (?, ?, 'LOCAL', ?)",
		storageID, workspaceID, "storage-files-"+storageID.String(),
	).Error)

	// The reference from storages to workspaces is RESTRICT, so the storage goes first; its pending
	// deletions go with it.
	t.Cleanup(func() {
		db.GetDb().Exec("DELETE FROM storages WHERE workspace_id = ?", workspaceID)
		db.GetDb().Exec("DELETE FROM workspaces WHERE id = ?", workspaceID)
	})

	return storageID
}

func Test_InsertIfAbsent_WhenNameIsFree_RegistersObligation(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	pending, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, time.Hour)

	require.NoError(t, err)
	assert.Equal(t, 0, pending.Generation)
	assert.Equal(t, 0, pending.AttemptCount)
	assert.Equal(t, reference, pending.GetReference())
}

func Test_InsertIfAbsent_WhenNameAlreadyRegistered_Refuses(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, 0)
	require.NoError(t, err)

	_, err = repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, 0)

	assert.ErrorIs(t, err, ErrFileNameAlreadyRegistered)
}

func Test_InsertIfAbsent_WhenTransactionRollsBack_LeavesNoRow(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	err := db.GetDb().Transaction(func(tx *gorm.DB) error {
		_, insertErr := repository.InsertIfAbsent(tx, uuid.New(), reference, 0)
		require.NoError(t, insertErr)

		return assert.AnError
	})
	require.ErrorIs(t, err, assert.AnError)

	found, err := repository.FindByReference(db.GetDb(), reference)

	require.NoError(t, err)
	assert.Nil(t, found)
}

func Test_InsertOrTake_WhenRequestedTwice_KeepsOneRowAndRaisesGeneration(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))
	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

	found, err := repository.FindByReference(db.GetDb(), reference)

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 1, found.Generation)
	assert.Equal(t, 0, found.AttemptCount,
		"asking for a file again is not a deletion attempt and must not advance the backoff")
}

func Test_InsertOrTake_WhenBatchRepeatsAReference_DoesNotFail(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	err := repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference, reference})

	require.NoError(t, err)

	found, findErr := repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, findErr)
	require.NotNil(t, found)
	assert.Equal(t, 0, found.Generation)
}

func Test_InsertOrTake_WhenBatchIsEmpty_DoesNothing(t *testing.T) {
	repository := &PendingDeletionRepository{}

	assert.NoError(t, repository.InsertOrTake(db.GetDb(), nil))
}

func Test_InsertOrTake_WhenWriterHoldsTheRow_TakesItByRaisingGeneration(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	pending, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, time.Hour)
	require.NoError(t, err)

	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

	completed, err := repository.CompleteIfGeneration(db.GetDb(), pending.ID, pending.Generation)

	require.NoError(t, err)
	assert.False(t, completed, "the writer's receipt must not release an obligation someone else took")
}

func Test_CompleteIfGeneration_WhenGenerationMatches_ReleasesObligation(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	pending, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, 0)
	require.NoError(t, err)

	completed, err := repository.CompleteIfGeneration(db.GetDb(), pending.ID, pending.Generation)
	require.NoError(t, err)
	assert.True(t, completed)

	found, err := repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func Test_RescheduleIfGeneration_WhenGenerationIsStale_ChangesNothing(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	pending, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, 0)
	require.NoError(t, err)

	rescheduled, err := repository.RescheduleIfGeneration(
		db.GetDb(), pending.ID, pending.Generation+1, time.Hour, "stale",
	)

	require.NoError(t, err)
	assert.False(t, rescheduled)

	found, err := repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Nil(t, found.LastError)
}

func Test_ClaimDue_WhenNothingIsDue_ReturnsEmpty(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, time.Hour)
	require.NoError(t, err)

	claimed, err := repository.ClaimDue(db.GetDb(), ClaimRequest{Limit: 10, AttemptLease: time.Minute})

	require.NoError(t, err)
	assert.NotContains(t, referencesOf(claimed), reference)
}

func Test_ClaimDue_WhenRowIsDue_TakesItAndRaisesGeneration(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

	claimed, err := repository.ClaimDue(db.GetDb(), ClaimRequest{Limit: 100, AttemptLease: time.Minute})
	require.NoError(t, err)
	require.Contains(t, referencesOf(claimed), reference)

	found, err := repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 1, found.Generation)
	assert.Equal(t, 1, found.AttemptCount)
	assert.False(t, isDue(t, reference), "a claimed row is not due again until its lease runs out")
}

func Test_ClaimDue_WhenWriteIsLive_SkipsIt(t *testing.T) {
	repository := &PendingDeletionRepository{}
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

	live, err := repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)

	claimed, err := repository.ClaimDue(db.GetDb(), ClaimRequest{
		Limit: 100, AttemptLease: time.Minute, ExcludedIDs: []uuid.UUID{live.ID},
	})

	require.NoError(t, err)
	assert.NotContains(t, referencesOf(claimed), reference)
}

func Test_ClaimDue_WhenAnotherTransactionHoldsTheRow_SkipsItInsteadOfWaiting(t *testing.T) {
	repository := &PendingDeletionRepository{}
	storageID := createStorageRow(t)
	held := StoredFileReference{StorageID: storageID, FileName: "held"}
	free := StoredFileReference{StorageID: storageID, FileName: "free"}

	require.NoError(t, repository.InsertOrTake(db.GetDb(), []StoredFileReference{held, free}))

	blocked := make(chan struct{})
	released := make(chan struct{})

	go func() {
		defer close(released)

		_ = db.GetDb().Transaction(func(tx *gorm.DB) error {
			if err := repository.InsertOrTake(tx, []StoredFileReference{held}); err != nil {
				return err
			}

			close(blocked)
			time.Sleep(2 * time.Second)

			return nil
		})
	}()

	<-blocked

	start := time.Now()
	claimed, err := repository.ClaimDue(db.GetDb(), ClaimRequest{Limit: 100, AttemptLease: time.Minute})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 2*time.Second, "the claim must skip a locked row rather than wait for it")
	assert.NotContains(t, referencesOf(claimed), held)
	assert.Contains(t, referencesOf(claimed), free)

	<-released
}

func referencesOf(rows []PendingDeletion) []StoredFileReference {
	references := make([]StoredFileReference, 0, len(rows))
	for _, row := range rows {
		references = append(references, row.GetReference())
	}

	return references
}
