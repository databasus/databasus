package backups_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestRestoreAcrossApplicationRestart(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("fails a restore that runs when the application stops", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		runningRestore, _ := startBlockedRestore(t, tm.owner, server, database.backup.Id)

		tm.app.RestartModules(t)

		failedRestore := waitForRestoreStatus(t, tm.owner, runningRestore.Id, apiclient.RestoreStatusFAILED)
		require.NotNil(t, failedRestore.FailMessage)
		assert.Contains(t, *failedRestore.FailMessage, "restore interrupted by application shutdown")
	})

	t.Run("fails a restore left in progress and accepts a new restore of the database", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		leftRestoreID := uuid.New()
		tm.app.ExecuteSQL(t, `INSERT INTO backups_restores
			(id, backup_id, database_id, status, restore_duration_ms, created_at)
			VALUES (?, ?, ?, 'IN_PROGRESS', 0, ?)`,
			leftRestoreID, database.backup.Id, database.databaseID, tm.app.Clock.Now().Add(-time.Minute))

		tm.app.RestartModules(t)

		leftRestore := getRestore(t, tm.owner, leftRestoreID)
		assert.Equal(t, apiclient.RestoreStatusFAILED, leftRestore.Status)
		require.NotNil(t, leftRestore.FailMessage)
		assert.Contains(t, *leftRestore.FailMessage, "Restore failed due to application restart")

		newRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, newRestore.Id, apiclient.RestoreStatusCOMPLETED)
	})
}
