package recovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishAccepted(t *testing.T) {
	t.Parallel()

	t.Run("publishes the accepted upload and leaves the lock shared", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))
		uploadTarget.acceptApply(t)
		lock := uploadTarget.openSharedLock(t)

		err := uploadTarget.newStartup(t, sourceArchive{}).PublishAccepted(t.Context(), lock, true)

		require.NoError(t, err)
		journal := uploadTarget.readJournal(t)
		assert.Nil(t, journal.Operation)
		require.NotNil(t, journal.LastCompleted)
		assert.Equal(t, preparedRecovery.ID, journal.LastCompleted.OperationID)
		assert.Equal(t, source.keyContent, uploadTarget.readKey(t))
		assert.Equal(t, []string{sourceWorkspaceName}, uploadTarget.listWorkspaceNames(t))
		assert.Equal(t, &preparedRecovery.ID, uploadTarget.getRecoveredByOperationID(t))
		requireSharedOnly(t, uploadTarget)
	})

	t.Run(
		"ends the recovery as QUIESCENCE_FAILED and changes nothing when the drain did not finish",
		func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			uploadTarget := newUploadTarget(t)
			activeKey := uploadTarget.readKey(t)
			preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))
			uploadTarget.acceptApply(t)
			lock := uploadTarget.openSharedLock(t)

			err := uploadTarget.newStartup(t, sourceArchive{}).PublishAccepted(t.Context(), lock, false)

			require.NoError(t, err)
			requireQuiescenceFailure(t, uploadTarget, preparedRecovery.ID.String())
			assert.Equal(t, activeKey, uploadTarget.readKey(t))
			assert.Empty(t, uploadTarget.listWorkspaceNames(t))
			requireSharedOnly(t, uploadTarget)
		},
	)

	t.Run("ends the recovery as QUIESCENCE_FAILED and changes nothing while another process holds the lock",
		func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			uploadTarget := newUploadTarget(t)
			activeKey := uploadTarget.readKey(t)
			preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))
			uploadTarget.acceptApply(t)
			lock := uploadTarget.openSharedLock(t)
			uploadTarget.openSharedLock(t)

			err := uploadTarget.newStartup(t, sourceArchive{}).PublishAccepted(t.Context(), lock, true)

			require.NoError(t, err)
			requireQuiescenceFailure(t, uploadTarget, preparedRecovery.ID.String())
			assert.Equal(t, activeKey, uploadTarget.readKey(t))
			assert.Empty(t, uploadTarget.listWorkspaceNames(t))
			requireSharedOnly(t, uploadTarget)
		})

	t.Run("keeps a prepared recovery whose apply was not accepted", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadTarget := newUploadTarget(t)
		preparedRecovery := uploadTarget.uploadAndPrepare(t, newUploadParts(t, source.path, source.keyContent))
		lock := uploadTarget.openSharedLock(t)

		err := uploadTarget.newStartup(t, sourceArchive{}).PublishAccepted(t.Context(), lock, true)

		require.NoError(t, err)
		journal := uploadTarget.readJournal(t)
		require.NotNil(t, journal.Operation)
		assert.Equal(t, preparedRecovery.ID, journal.Operation.ID)
		assert.Equal(t, PhaseImported, journal.Operation.Phase)
		assert.Empty(t, uploadTarget.listWorkspaceNames(t))
	})
}

// acceptApply records an accepted apply as Service.Apply does after root admin's proof.
func (ut uploadTarget) acceptApply(t *testing.T) {
	t.Helper()

	journalStore := NewJournalStore(ut.getDirectory())
	journal, err := journalStore.Read()
	require.NoError(t, err)
	require.NotNil(t, journal.Operation)

	journal.Operation.IsApplyAccepted = true
	require.NoError(t, journalStore.Write(journal))
}

// openSharedLock holds the lifecycle lock shared, as a running server or console command does.
func (ut uploadTarget) openSharedLock(t *testing.T) *LifecycleLock {
	t.Helper()

	lock, err := OpenLifecycleLock(ut.getDirectory())
	require.NoError(t, err)
	t.Cleanup(func() { _ = lock.Close() })

	isShared, err := lock.TryShared()
	require.NoError(t, err)
	require.True(t, isShared)

	return lock
}

func requireQuiescenceFailure(t *testing.T, uploadTarget uploadTarget, operationID string) {
	t.Helper()

	journal := uploadTarget.readJournal(t)
	assert.Nil(t, journal.Operation)
	require.NotNil(t, journal.LastEnded)
	assert.Equal(t, operationID, journal.LastEnded.OperationID.String())
	require.NotNil(t, journal.LastEnded.Failure)
	assert.Equal(t, FailureQuiescenceFailed, *journal.LastEnded.Failure)
	assert.Empty(t, uploadTarget.listRecoveryDatabases(t))
	assert.Contains(t, uploadTarget.getLogs(), "the accepted recovery was not published")
}

// requireSharedOnly checks that the lock is still held shared: another process can share it but
// cannot take it exclusive.
func requireSharedOnly(t *testing.T, uploadTarget uploadTarget) {
	t.Helper()

	otherLock, err := OpenLifecycleLock(uploadTarget.getDirectory())
	require.NoError(t, err)
	defer func() { _ = otherLock.Close() }()

	isExclusive, err := otherLock.TryExclusive()
	require.NoError(t, err)
	assert.False(t, isExclusive)
}
