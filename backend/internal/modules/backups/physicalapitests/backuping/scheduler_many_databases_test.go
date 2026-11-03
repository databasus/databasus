package backuping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhysicalScheduler(t *testing.T) {
	t.Parallel()

	t.Run("decides each database from its own history when one pass reads many databases", func(t *testing.T) {
		t.Parallel()

		dueTeam := newBackupState(t)
		dueTeam.setSchedules(t, true)
		dueTeam.seedFull(t, schedulerFullSpec{Age: 72 * time.Hour})
		listedDueBackups := dueTeam.listBackups(t)

		mismatchTeam := dueTeam.addDatabase(t)
		mismatchTeam.setSchedules(t, true)
		mismatchTeam.seedUnresolvedMismatch(t)
		listedMismatchBackups := mismatchTeam.listBackups(t)

		notDueTeam := dueTeam.addIdleDatabase(t)
		listedNotDueBackups := notDueTeam.listBackups(t)

		dueTeam.App.RunJobPass(t, "backups_physical_scheduler")

		startedBackups := dueTeam.listNewBackups(t, listedDueBackups)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
		assert.Empty(t, mismatchTeam.listNewBackups(t, listedMismatchBackups))
		assert.Empty(t, notDueTeam.listNewBackups(t, listedNotDueBackups))
	})

	t.Run("starts a full backup only for the database whose mismatch a completed full backup resolved",
		func(t *testing.T) {
			t.Parallel()

			resolvedTeam := newBackupState(t)
			resolvedTeam.setSchedules(t, true)
			resolvedTeam.seedUnresolvedMismatch(t)
			resolvedTeam.seedFull(t, schedulerFullSpec{Age: 72 * time.Hour, Segment: 2})
			listedResolvedBackups := resolvedTeam.listBackups(t)

			unresolvedTeam := resolvedTeam.addDatabase(t)
			unresolvedTeam.setSchedules(t, true)
			unresolvedTeam.seedUnresolvedMismatch(t)
			listedUnresolvedBackups := unresolvedTeam.listBackups(t)

			resolvedTeam.App.RunJobPass(t, "backups_physical_scheduler")

			startedBackups := resolvedTeam.listNewBackups(t, listedResolvedBackups)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "FULL", startedBackups[0].Type)
			assert.Empty(t, unresolvedTeam.listNewBackups(t, listedUnresolvedBackups))
		},
	)
}

func (team *backupTeam) seedUnresolvedMismatch(t *testing.T) {
	t.Helper()
	team.seedFull(
		t,
		schedulerFullSpec{Status: "CHAIN_BROKEN", Reason: new("SYSTEM_IDENTIFIER_MISMATCH"), Age: 96 * time.Hour},
	)
}
