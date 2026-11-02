package backuping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSchedulePhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run(
		"starts a full backup when AUTO is requested for a full-only database with a completed full",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.setSchedules(t, true)
			team.seedFull(t, schedulerFullSpec{Age: time.Minute})
			team.trigger(t, "auto")

			var requests struct {
				IsFullRequested        bool
				IsIncrementalRequested bool
			}
			team.App.QuerySQL(t, &requests,
				`SELECT force_full_requested_at IS NOT NULL AS is_full_requested,
                        force_incremental_requested_at IS NOT NULL AS is_incremental_requested
                 FROM backups_configs_physical WHERE database_id=?`, team.Database.Id)

			assert.True(t, requests.IsFullRequested)
			assert.False(t, requests.IsIncrementalRequested)

			backup := team.runAutomaticBackup(t)
			assert.Equal(t, "FULL", backup.Type)
		},
	)

	t.Run("starts a replacement full immediately when an incremental detected a timeline switch", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		full := team.seedFull(t, schedulerFullSpec{Age: 3 * time.Hour})
		team.seedIncremental(t, full.ID, "TIMELINE_SWITCH_DETECTED")
		team.setSchedules(t, false)
		backup := team.runAutomaticBackup(t)
		assert.Equal(t, "FULL", backup.Type)
	})
	t.Run(
		"suppresses automatic work when a system identifier mismatch has no completed replacement",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(
				t,
				schedulerFullSpec{
					Status: "CHAIN_BROKEN",
					Reason: new("SYSTEM_IDENTIFIER_MISMATCH"),
					Age:    48 * time.Hour,
				},
			)
			team.assertNoScheduledBackup(t)
		},
	)
	t.Run("allows a requested full when a system identifier mismatch suppresses automatic work", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(
			t,
			schedulerFullSpec{Status: "CHAIN_BROKEN", Reason: new("SYSTEM_IDENTIFIER_MISMATCH"), Age: 48 * time.Hour},
		)
		backup := team.runBackup(t, "full")
		assert.Equal(t, "FULL", backup.Type)
	})
	t.Run(
		"resumes automatic scheduling when a completed full follows a system identifier mismatch",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(
				t,
				schedulerFullSpec{
					Status: "CHAIN_BROKEN",
					Reason: new("SYSTEM_IDENTIFIER_MISMATCH"),
					Age:    96 * time.Hour,
				},
			)
			team.seedFull(t, schedulerFullSpec{Age: 72 * time.Hour, Segment: 2})
			backup := team.runAutomaticBackup(t)
			assert.Equal(t, "FULL", backup.Type)
		},
	)
	t.Run(
		"suppresses automatic scheduling when a later full failed after a system identifier mismatch",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(
				t,
				schedulerFullSpec{
					Status: "CHAIN_BROKEN",
					Reason: new("SYSTEM_IDENTIFIER_MISMATCH"),
					Age:    96 * time.Hour,
				},
			)
			team.seedFull(t, schedulerFullSpec{Status: "ERROR", Age: 72 * time.Hour, Segment: 2})
			team.assertNoScheduledBackup(t)
		},
	)
	t.Run(
		"suppresses automatic scheduling when a later full was cancelled after a system identifier mismatch",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(
				t,
				schedulerFullSpec{
					Status: "CHAIN_BROKEN",
					Reason: new("SYSTEM_IDENTIFIER_MISMATCH"),
					Age:    96 * time.Hour,
				},
			)
			team.seedFull(t, schedulerFullSpec{Status: "CANCELED", Age: 72 * time.Hour, Segment: 2})
			team.assertNoScheduledBackup(t)
		},
	)

	t.Run("starts an incremental once the built daily slot passes when the stored cron has no date to run on",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(t, schedulerFullSpec{Age: 3 * time.Hour, ManifestFileName: new("root.manifest")})
			team.setSchedules(t, false)
			team.storeUndatedIncrementalCronWithDailySlotTwoHoursAgo(t)

			backup := team.runAutomaticBackup(t)

			assert.Equal(t, "INCREMENTAL", backup.Type)
		},
	)

	t.Run("schedules nothing before the built daily slot when the stored cron has no date to run on",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(t, schedulerFullSpec{Age: time.Hour, ManifestFileName: new("root.manifest")})
			team.setSchedules(t, false)
			team.storeUndatedIncrementalCronWithDailySlotTwoHoursAgo(t)

			team.assertNoScheduledBackup(t)
		},
	)

	t.Run("schedules nothing when the stored incremental cron has no date under an hourly full schedule",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(t, schedulerFullSpec{Age: 10 * time.Minute, ManifestFileName: new("root.manifest")})
			team.setSchedules(t, false)
			team.App.ExecuteSQL(t, `UPDATE backups_configs_physical
				SET full_interval_type = 'HOURLY',
					incremental_interval_type = 'CRON', incremental_cron_expression = '0 0 31 2 *'
				WHERE database_id = ?`, team.Database.Id)

			team.assertNoScheduledBackup(t)
		},
	)

	t.Run("schedules nothing when a full only database has a recent full", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.setSchedules(t, true)
		team.seedFull(t, schedulerFullSpec{Age: 2 * time.Hour})
		team.assertNoScheduledBackup(t)
	})
	t.Run("keeps the running backup without inserting another when a backup is running", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Status: "IN_PROGRESS"})

		team.assertNoScheduledBackup(t)
	})
	t.Run("chooses a requested full before its cadence", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute})
		team.setSchedules(t, false)
		backup := team.runBackup(t, "full")
		assert.Equal(t, "FULL", backup.Type)
	})
	t.Run("chooses a requested incremental before its cadence", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute, ManifestFileName: new("root.manifest")})
		team.setSchedules(t, false)
		backup := team.runBackup(t, "incremental")
		assert.Equal(t, "INCREMENTAL", backup.Type)
	})
	t.Run("clears an incremental request and falls back to full when no chain is extendable", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_configs_physical SET force_incremental_requested_at=? WHERE database_id=?",
			time.Now().UTC(),
			team.Database.Id,
		)

		backup := team.runAutomaticBackup(t)
		assert.Equal(t, "FULL", backup.Type)

		var pendingRequests int64
		team.App.QuerySQL(
			t,
			&pendingRequests,
			"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_incremental_requested_at IS NOT NULL",
			team.Database.Id,
		)
		assert.Zero(t, pendingRequests)
	})
	t.Run("clears an incremental request when incrementals are disabled", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.setSchedules(t, true)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute})

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_configs_physical SET force_incremental_requested_at=? WHERE database_id=?",
			time.Now().UTC(),
			team.Database.Id,
		)

		team.assertNoScheduledBackup(t)

		var pendingRequests int64
		team.App.QuerySQL(
			t,
			&pendingRequests,
			"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_incremental_requested_at IS NOT NULL",
			team.Database.Id,
		)
		assert.Zero(t, pendingRequests)
	})
}
