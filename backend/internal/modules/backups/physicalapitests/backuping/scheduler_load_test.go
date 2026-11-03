package backuping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhysicalSchedulerLoad(t *testing.T) {
	t.Parallel()

	t.Run("reads the state of many databases with a fixed number of queries when nothing is due", func(t *testing.T) {
		t.Parallel()

		firstTeam := newIdleBackupState(t)
		idleTeams := []*backupTeam{firstTeam, firstTeam.addIdleDatabase(t)}

		queriesForTwoDatabases := firstTeam.App.CountInternalDatabaseQueries(t, func() {
			firstTeam.App.RunJobPass(t, "backups_physical_scheduler")
		})

		for range 3 {
			idleTeams = append(idleTeams, firstTeam.addIdleDatabase(t))
		}

		queriesForFiveDatabases := firstTeam.App.CountInternalDatabaseQueries(t, func() {
			firstTeam.App.RunJobPass(t, "backups_physical_scheduler")
		})

		assert.Equal(t, queriesForTwoDatabases, queriesForFiveDatabases)
		assert.Positive(t, queriesForTwoDatabases)

		for _, idleTeam := range idleTeams {
			assert.Len(t, idleTeam.listBackups(t), 1)
		}
	})

	t.Run("sends one query when no request is pending", func(t *testing.T) {
		t.Parallel()

		firstTeam := newIdleBackupState(t)
		idleTeams := []*backupTeam{firstTeam, firstTeam.addIdleDatabase(t)}

		queriesForTwoDatabases := firstTeam.App.CountInternalDatabaseQueries(t, func() {
			firstTeam.App.RunJobPass(t, backupRequestJob)
		})

		for range 3 {
			idleTeams = append(idleTeams, firstTeam.addIdleDatabase(t))
		}

		queriesForFiveDatabases := firstTeam.App.CountInternalDatabaseQueries(t, func() {
			firstTeam.App.RunJobPass(t, backupRequestJob)
		})

		assert.Equal(t, int64(1), queriesForTwoDatabases)
		assert.Equal(t, int64(1), queriesForFiveDatabases)

		for _, idleTeam := range idleTeams {
			assert.Len(t, idleTeam.listBackups(t), 1)
		}
	})
}
