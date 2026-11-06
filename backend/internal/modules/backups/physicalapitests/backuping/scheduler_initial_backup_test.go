package backuping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhysicalInitialBackup(t *testing.T) {
	t.Parallel()

	t.Run("starts one full without a manual request when an enabled configuration has never run", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		team.setSchedules(t, false)

		team.App.RunJobPass(t, "backups_physical_scheduler")
		team.App.WaitForBackupRuns(t)
		team.App.RunJobPass(t, backupRequestJob)
		team.App.RunJobPass(t, "backups_physical_scheduler")
		team.App.WaitForBackupRuns(t)

		listedBackups := team.listBackups(t)
		require.Len(t, listedBackups, 1)
		assert.Equal(t, "FULL", listedBackups[0].Type)
		assert.Equal(t, "COMPLETED", listedBackups[0].Status)
		assert.False(t, team.hasFullBackupRequest(t))
	})

	t.Run("extends the second full when AUTO queues it during the scheduled initial full", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		team.setSchedules(t, false)
		releaseStorage := holdStorageWrites(t, team)
		team.App.RunJobPass(t, "backups_physical_scheduler")
		initialBackups := team.listBackups(t)
		require.Len(t, initialBackups, 1)
		require.Equal(t, "IN_PROGRESS", initialBackups[0].Status)

		team.trigger(t, "auto")
		team.App.RunJobPass(t, backupRequestJob)

		assert.Len(t, team.listBackups(t), 1)
		require.True(t, team.hasFullBackupRequest(t))

		releaseStorage()
		team.App.WaitForBackupRuns(t)
		team.App.Clock.Advance(time.Second)
		team.App.RunJobPass(t, backupRequestJob)
		team.App.WaitForBackupRuns(t)

		listedBackups := team.listBackups(t)
		require.Len(t, listedBackups, 2)
		assert.NotEqual(t, initialBackups[0].ID, listedBackups[0].ID)
		for _, backup := range listedBackups {
			assert.Equal(t, "FULL", backup.Type)
			assert.Equal(t, "COMPLETED", backup.Status)
		}
		assert.False(t, team.hasFullBackupRequest(t))

		team.App.Clock.Advance(time.Second)
		incrementalBackup := team.runBackup(t, "incremental")
		require.Equal(t, "COMPLETED", incrementalBackup.Status)
		require.NotNil(t, incrementalBackup.RootFullBackupID)
		assert.Equal(t, listedBackups[0].ID, *incrementalBackup.RootFullBackupID)
		assert.NotEqual(t, initialBackups[0].ID, *incrementalBackup.RootFullBackupID)
	})
}
