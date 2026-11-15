package recovery

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterruptedPublication(t *testing.T) {
	t.Parallel()

	for _, interruption := range []struct {
		name      string
		interrupt func(t *testing.T, startup *Startup, journal Journal)
	}{
		{
			name: "finishes a publication stopped right after it was recorded",
			interrupt: func(t *testing.T, startup *Startup, journal Journal) {
				t.Helper()
				recordPublishing(t, startup, journal)
			},
		},
		{
			name: "finishes a publication stopped after the database swap",
			interrupt: func(t *testing.T, startup *Startup, journal Journal) {
				t.Helper()
				recordPublishing(t, startup, journal)
				require.NoError(t, startup.publisher.retainPreviousKey(journal))
				require.NoError(t, startup.publisher.swapDatabasesOnce(t.Context(), journal))
			},
		},
		{
			name: "finishes a publication stopped after the key was installed",
			interrupt: func(t *testing.T, startup *Startup, journal Journal) {
				t.Helper()
				recordPublishing(t, startup, journal)
				require.NoError(t, startup.publisher.retainPreviousKey(journal))
				require.NoError(t, startup.publisher.swapDatabasesOnce(t.Context(), journal))
				installStagedKey(t, startup, journal)
			},
		},
		{
			name: "releases what a publication stopped after recording completion left behind",
			interrupt: func(t *testing.T, startup *Startup, journal Journal) {
				t.Helper()
				recordPublishing(t, startup, journal)
				require.NoError(t, startup.publisher.retainPreviousKey(journal))
				require.NoError(t, startup.publisher.swapDatabasesOnce(t.Context(), journal))
				keyContent := installStagedKey(t, startup, journal)
				require.NoError(t, startup.journal.Write(startup.publisher.recordCompletion(journal, hashBytes(keyContent))))
			},
		},
	} {
		t.Run(interruption.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			initializedTarget := newInitializedTarget(t)
			startup := initializedTarget.newStartup(t, source)
			journal := prepareStartupOperation(t, startup, source)
			operationID := journal.Operation.ID
			interruption.interrupt(t, startup, journal)

			_, consoleErr := initializedTarget.newStartup(t, sourceArchive{}).OpenForConsole()
			if initializedTarget.readJournal(t).IsPublicationUnfinished() {
				require.ErrorIs(t, consoleErr, ErrPublicationUnfinished)
			}
			err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

			require.NoError(t, err)
			archiveSha256, err := hashFile(source.path)
			require.NoError(t, err)
			finishedJournal := initializedTarget.readJournal(t)
			assert.Equal(t, []string{sourceWorkspaceName}, initializedTarget.listWorkspaceNames(t))
			assert.Equal(t, source.keyContent, initializedTarget.readKey(t))
			assert.Equal(t, &operationID, initializedTarget.getPausedOperationID(t))
			assert.Equal(t, []string{archiveSha256}, finishedJournal.ConsumedStartupArchives)
			assert.Equal(t, int64(1), finishedJournal.PublicationGeneration)
			assert.Nil(t, finishedJournal.Operation)
			assert.Empty(t, initializedTarget.listRecoveryDatabases(t))
			assertNoOperationFiles(t, initializedTarget)
		})
	}

	t.Run("discards an imported operation whose apply was never accepted", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		generatedKey := initializedTarget.readKey(t)
		prepareStartupOperation(t, initializedTarget.newStartup(t, source), source)

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

		require.NoError(t, err)
		assertTargetUnchanged(t, initializedTarget, generatedKey)
		assert.Empty(t, initializedTarget.readJournal(t).ConsumedStartupArchives)
	})

	t.Run("publishes an imported operation whose apply was accepted", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		startup := initializedTarget.newStartup(t, source)
		journal := prepareStartupOperation(t, startup, source)
		journal.Operation.Origin = OriginUpload
		journal.Operation.IsApplyAccepted = true
		require.NoError(t, startup.journal.Write(journal))

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

		require.NoError(t, err)
		assert.Equal(t, []string{sourceWorkspaceName}, initializedTarget.listWorkspaceNames(t))
		assert.Equal(t, source.keyContent, initializedTarget.readKey(t))
		assert.Empty(t, initializedTarget.readJournal(t).ConsumedStartupArchives)
	})

	t.Run("refuses a second process while a publication is unfinished", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		startup := initializedTarget.newStartup(t, source)
		recordPublishing(t, startup, prepareStartupOperation(t, startup, source))
		otherProcess := holdSharedLock(t, initializedTarget)
		defer func() { _ = otherProcess.Close() }()

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

		require.ErrorIs(t, err, ErrPublicationUnfinished)
	})
}

func TestPublishedState(t *testing.T) {
	t.Parallel()

	t.Run("keeps an upload import current when the consumed startup archive stays configured", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		uploadedSource := newSourceArchiveWith(
			t,
			archiveSpec{release: testRelease, bundleDigest: "", workspaceName: "Uploaded"},
		)
		recoveredTarget := newFreshTarget(t)
		require.NoError(t, recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source)))
		publishAsUpload(t, recoveredTarget, uploadedSource)

		err := recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source))

		require.NoError(t, err)
		archiveSha256, err := hashFile(source.path)
		require.NoError(t, err)
		journal := recoveredTarget.readJournal(t)
		assert.Equal(t, []string{"Uploaded"}, recoveredTarget.listWorkspaceNames(t))
		assert.Equal(t, uploadedSource.keyContent, recoveredTarget.readKey(t))
		assert.Equal(t, []string{archiveSha256}, journal.ConsumedStartupArchives)
		assert.Equal(t, OriginUpload, journal.LastCompleted.Origin)
		assert.Equal(t, int64(2), journal.PublicationGeneration)
	})

	t.Run("refuses to start when the key file is not the one published with the database", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		recoveredTarget := newFreshTarget(t)
		require.NoError(t, recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source)))
		require.NoError(
			t,
			os.WriteFile(recoveredTarget.secretKeyPath, []byte(uuid.NewString()+uuid.NewString()), 0o600),
		)

		serverErr := recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, sourceArchive{}))
		_, consoleErr := recoveredTarget.newStartup(t, sourceArchive{}).OpenForConsole()

		require.ErrorIs(t, serverErr, ErrPublishedKeyMismatch)
		require.ErrorIs(t, consoleErr, ErrPublishedKeyMismatch)
	})
}

func recordPublishing(t *testing.T, startup *Startup, journal Journal) {
	t.Helper()

	journal.Operation.Phase = PhasePublishing
	require.NoError(t, startup.journal.Write(journal))
}

func installStagedKey(t *testing.T, startup *Startup, journal Journal) []byte {
	t.Helper()

	keyContent, err := os.ReadFile(startup.directory.getStagedKeyPath(journal.Operation.ID))
	require.NoError(t, err)
	require.NoError(t, writeFileDurably(startup.secretKeyPath, keyContent))

	return keyContent
}

// publishAsUpload stands in for a confirmed UI import: the same staging and publication with the
// upload origin.
func publishAsUpload(t *testing.T, tg target, source sourceArchive) {
	t.Helper()

	startup := tg.newStartup(t, source)
	journal := prepareStartupOperation(t, startup, source)
	journal.Operation.Origin = OriginUpload
	journal.Operation.IsApplyAccepted = true
	require.NoError(t, startup.journal.Write(journal))

	_, err := startup.publisher.Publish(t.Context(), journal)
	require.NoError(t, err)
}
