package backuping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const storageChangedFailMessage = "stopped by the storage change"

func TestStaleStoragePublication(t *testing.T) {
	t.Parallel()

	t.Run("does not publish the backup when its storage publication went stale", func(t *testing.T) {
		t.Parallel()

		finishedRun := finishBackupAfterStorageChange(t, func(*backupTeam, backupRow) {})

		runningBackup := findPhysicalBackup(t, finishedRun.team, finishedRun.backup)
		assert.Equal(t, "IN_PROGRESS", runningBackup.Status)
		finishedRun.assertUnpublished(t)

		cancelPhysicalBackup(t, finishedRun.team, finishedRun.backup)

		canceledBackup := findPhysicalBackup(t, finishedRun.team, finishedRun.backup)
		assert.Equal(t, "CANCELED", canceledBackup.Status)
	})

	t.Run("keeps a removed backup removed when its storage publication went stale", func(t *testing.T) {
		t.Parallel()

		finishedRun := finishBackupAfterStorageChange(t, func(team *backupTeam, backup backupRow) {
			team.App.ExecuteSQL(t, "DELETE FROM backups_physical WHERE id=?", backup.ID)
		})

		for _, listedBackup := range finishedRun.team.listBackups(t) {
			assert.NotEqual(t, finishedRun.backup.ID, listedBackup.ID)
		}

		finishedRun.assertUnpublished(t)
	})

	t.Run("keeps a cancelled backup's message when its storage publication went stale", func(t *testing.T) {
		t.Parallel()

		finishedRun := finishBackupAfterStorageChange(t, func(team *backupTeam, backup backupRow) {
			team.App.ExecuteSQL(t, "UPDATE backups_physical SET status='CANCELED', fail_message=? WHERE id=?",
				storageChangedFailMessage, backup.ID)
		})

		canceledBackup := findPhysicalBackup(t, finishedRun.team, finishedRun.backup)
		assert.Equal(t, "CANCELED", canceledBackup.Status)
		require.NotNil(t, canceledBackup.FailMessage)
		assert.Equal(t, storageChangedFailMessage, *canceledBackup.FailMessage)
		finishedRun.assertUnpublished(t)
	})
}

type staleStorageRun struct {
	team      *backupTeam
	backup    backupRow
	fileName  string
	failures  *webhookInbox
	successes *webhookInbox
}

// finishBackupAfterStorageChange holds a full backup at its last write, moves the storage generation
// by SQL and lets the run finish. A real storage change would also stop the run before it publishes,
// so that race needs this state. changeBackupRow sets what the storage change did to the row.
func finishBackupAfterStorageChange(
	t *testing.T,
	changeBackupRow func(team *backupTeam, backup backupRow),
) staleStorageRun {
	t.Helper()

	team := newBackupTeam(t, backupTeamSpec{})
	failures, successes := &webhookInbox{}, &webhookInbox{}
	team.attachWebhook(t, failures, apiclient.NotificationCategoryBACKUPFAILED)
	team.attachWebhook(t, successes, apiclient.NotificationCategoryBACKUPSUCCESS)
	team.setNotifications(t, apiclient.NotificationTriggerBACKUPFAILED, apiclient.NotificationTriggerBACKUPSUCCESS)

	metadata := holdBackupMetadata(t, team)
	team.trigger(t, "full")
	team.App.RunJobPass(t, "backups_physical_scheduler")
	metadata.Wait(t)

	var runningBackup backupRow
	for _, listedBackup := range team.listBackups(t) {
		if listedBackup.Status == "IN_PROGRESS" {
			runningBackup = listedBackup
		}
	}

	require.NotEmpty(t, runningBackup.ID)

	var copiedBackup apptest.PhysicalFullBackupSeed
	team.App.QuerySQL(t, &copiedBackup, "SELECT * FROM backups_physical WHERE id=?", runningBackup.ID)
	require.NotNil(t, copiedBackup.FileName)

	team.App.ExecuteSQL(t,
		"UPDATE backups_configs SET storage_generation=storage_generation+1 WHERE database_id=?",
		team.Database.Id)
	changeBackupRow(team, runningBackup)

	metadata.Release(t)
	team.App.WaitForBackupRuns(t)

	return staleStorageRun{
		team:      team,
		backup:    runningBackup,
		fileName:  *copiedBackup.FileName,
		failures:  failures,
		successes: successes,
	}
}

func (run staleStorageRun) assertUnpublished(t *testing.T) {
	t.Helper()

	var lastBackupCount int64
	run.team.App.QuerySQL(t, &lastBackupCount,
		"SELECT count(*) FROM databases WHERE id=? AND last_backup_time IS NOT NULL", run.team.Database.Id)
	assert.Zero(t, lastBackupCount)

	// An unconfirmed write also leaves a pending deletion, so only a raised generation tells that the
	// run asked for the deletion itself.
	var requestedDeletionCount int64
	run.team.App.QuerySQL(t, &requestedDeletionCount,
		"SELECT count(*) FROM storages_pending_file_deletions WHERE file_name=? AND generation>0", run.fileName)
	assert.Equal(t, int64(1), requestedDeletionCount)

	run.team.App.Clock.Advance(3 * time.Second)
	run.team.App.RunJobPass(t, "storage_file_deletion")

	for _, suffix := range []string{"", ".manifest", ".metadata"} {
		assert.False(t, run.team.App.IsLocalStorageFilePresent(t, run.fileName+suffix), suffix)
	}

	run.team.App.WaitForNotificationDeliveries(t)
	assert.Empty(t, run.failures.getMessages())
	assert.Empty(t, run.successes.getMessages())
}
