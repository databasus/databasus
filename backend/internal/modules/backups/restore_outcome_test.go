package backups_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestRestoreOutcome(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("fails the restore with the storage error when the storage refuses the read", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		hideBackupFile(t, tm, database)

		startedRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		failedRestore := waitForRestoreStatus(t, tm.owner, startedRestore.Id, apiclient.RestoreStatusFAILED)

		require.NotNil(t, failedRestore.FailMessage)
		assert.Contains(t, *failedRestore.FailMessage, "failed to get backup file from storage")
		requireStartedRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, server.newTargetSpec(t)))
	})

	t.Run("lists the restores of a backup newest first after a failed restore", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		showBackupFile := hideBackupFile(t, tm, database)
		failedRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, failedRestore.Id, apiclient.RestoreStatusFAILED)

		showBackupFile()
		tm.app.Clock.Advance(time.Second)
		completedRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, completedRestore.Id, apiclient.RestoreStatusCOMPLETED)

		listedRestores := listRestores(t, tm.owner, database.backup.Id)

		require.Len(t, listedRestores, 2)
		assert.Equal(t, completedRestore.Id, listedRestores[0].Id)
		assert.Equal(t, apiclient.RestoreStatusCOMPLETED, listedRestores[0].Status)
		assert.Nil(t, listedRestores[0].FailMessage)
		assert.Equal(t, failedRestore.Id, listedRestores[1].Id)
		assert.Equal(t, apiclient.RestoreStatusFAILED, listedRestores[1].Status)
		assert.NotNil(t, listedRestores[1].FailMessage)
	})
}

// hideBackupFile moves the backup's file away from the local storage, so the storage refuses to read
// it, and returns the function that moves it back.
func hideBackupFile(t *testing.T, tm team, database backedUpDatabase) (showBackupFile func()) {
	t.Helper()

	backupFilePath := getBackupFilePath(t, tm, database.backup.Id)
	hiddenFilePath := backupFilePath + ".hidden"
	require.NoError(t, os.Rename(backupFilePath, hiddenFilePath))

	return func() {
		require.NoError(t, os.Rename(hiddenFilePath, backupFilePath))
	}
}
