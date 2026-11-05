package logical

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/backups/internal/config"
)

func TestGetExpiredBackups(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	databaseID := uuid.New()

	t.Run("drops every backup older than the time period, whatever its status", func(t *testing.T) {
		t.Parallel()
		weekOldFailedBackup := newBackup(databaseID, StatusFailed, now.Add(-8*24*time.Hour))
		weekOldCompletedBackup := newBackup(databaseID, StatusCompleted, now.Add(-9*24*time.Hour))
		freshBackup := newBackup(databaseID, StatusCompleted, now.Add(-2*24*time.Hour))

		expiredBackups := getExpiredBackups(newTimePeriodRetention(config.RetentionPeriodWeek),
			[]LogicalBackup{freshBackup, weekOldFailedBackup, weekOldCompletedBackup}, now)

		assert.Equal(t, []uuid.UUID{weekOldFailedBackup.ID, weekOldCompletedBackup.ID}, getBackupIDs(expiredBackups))
	})

	t.Run("keeps every backup when the time period is forever", func(t *testing.T) {
		t.Parallel()
		yearOldBackup := newBackup(databaseID, StatusCompleted, now.Add(-400*24*time.Hour))

		expiredBackups := getExpiredBackups(newTimePeriodRetention(config.RetentionPeriodForever),
			[]LogicalBackup{yearOldBackup}, now)

		assert.Empty(t, expiredBackups)
	})

	t.Run("keeps the newest five completed backups of seven under the count policy", func(t *testing.T) {
		t.Parallel()
		completedBackups := newHourlyBackups(databaseID, StatusCompleted, now.Add(-2*time.Hour), 7)

		expiredBackups := getExpiredBackups(newCountRetention(5), completedBackups, now)

		assert.Equal(t, getBackupIDs(completedBackups[5:]), getBackupIDs(expiredBackups))
	})

	t.Run("keeps failed and cancelled backups under the count policy", func(t *testing.T) {
		t.Parallel()
		failedBackup := newBackup(databaseID, StatusFailed, now.Add(-10*time.Hour))
		cancelledBackup := newBackup(databaseID, StatusCanceled, now.Add(-11*time.Hour))
		completedBackups := newHourlyBackups(databaseID, StatusCompleted, now.Add(-2*time.Hour), 2)

		expiredBackups := getExpiredBackups(newCountRetention(1),
			append(completedBackups, failedBackup, cancelledBackup), now)

		assert.Equal(t, []uuid.UUID{completedBackups[1].ID}, getBackupIDs(expiredBackups))
	})

	t.Run("keeps the newest backup of each day the GFS policy names and drops the rest", func(t *testing.T) {
		t.Parallel()
		todayLateBackup := newBackup(databaseID, StatusCompleted, now.Add(-2*time.Hour))
		todayEarlyBackup := newBackup(databaseID, StatusCompleted, now.Add(-5*time.Hour))
		yesterdayBackup := newBackup(databaseID, StatusCompleted, now.Add(-26*time.Hour))
		weekOldBackup := newBackup(databaseID, StatusCompleted, now.Add(-7*24*time.Hour))

		expiredBackups := getExpiredBackups(newGfsRetention(config.GfsRetention{Days: 2}),
			[]LogicalBackup{todayLateBackup, todayEarlyBackup, yesterdayBackup, weekOldBackup}, now)

		assert.ElementsMatch(t, []uuid.UUID{todayEarlyBackup.ID, weekOldBackup.ID}, getBackupIDs(expiredBackups))
	})

	t.Run("keeps failed and cancelled backups under the GFS policy", func(t *testing.T) {
		t.Parallel()
		completedBackup := newBackup(databaseID, StatusCompleted, now.Add(-2*time.Hour))
		failedBackup := newBackup(databaseID, StatusFailed, now.Add(-30*24*time.Hour))
		cancelledBackup := newBackup(databaseID, StatusCanceled, now.Add(-31*24*time.Hour))

		expiredBackups := getExpiredBackups(newGfsRetention(config.GfsRetention{Days: 1}),
			[]LogicalBackup{completedBackup, failedBackup, cancelledBackup}, now)

		assert.Empty(t, expiredBackups)
	})

	t.Run("keeps a backup younger than one hour that the policy drops", func(t *testing.T) {
		t.Parallel()
		newestBackup := newBackup(databaseID, StatusCompleted, now.Add(-10*time.Minute))
		halfHourOldBackup := newBackup(databaseID, StatusCompleted, now.Add(-40*time.Minute))
		twoHoursOldBackup := newBackup(databaseID, StatusCompleted, now.Add(-2*time.Hour))

		expiredBackups := getExpiredBackups(newCountRetention(1),
			[]LogicalBackup{newestBackup, halfHourOldBackup, twoHoursOldBackup}, now)

		assert.Equal(t, []uuid.UUID{twoHoursOldBackup.ID}, getBackupIDs(expiredBackups))
	})
}

// newHourlyBackups makes count backups an hour apart, newest first, starting at newestAt.
func newHourlyBackups(databaseID uuid.UUID, status Status, newestAt time.Time, count int) []LogicalBackup {
	hourlyBackups := make([]LogicalBackup, 0, count)
	for position := range count {
		hourlyBackups = append(hourlyBackups,
			newBackup(databaseID, status, newestAt.Add(-time.Duration(position)*time.Hour)))
	}

	return hourlyBackups
}

func newTimePeriodRetention(period config.RetentionPeriod) config.LogicalRetention {
	return config.LogicalRetention{
		Policy: config.LogicalRetentionPolicyTimePeriod,
		Period: period,
		Count:  0,
		Gfs:    config.GfsRetention{},
	}
}

func newCountRetention(count int) config.LogicalRetention {
	return config.LogicalRetention{
		Policy: config.LogicalRetentionPolicyCount,
		Period: config.RetentionPeriodForever,
		Count:  count,
		Gfs:    config.GfsRetention{},
	}
}

func newGfsRetention(gfsRetention config.GfsRetention) config.LogicalRetention {
	return config.LogicalRetention{
		Policy: config.LogicalRetentionPolicyGfs,
		Period: config.RetentionPeriodForever,
		Count:  0,
		Gfs:    gfsRetention,
	}
}

func getBackupIDs(listedBackups []LogicalBackup) []uuid.UUID {
	backupIDs := make([]uuid.UUID, 0, len(listedBackups))
	for _, listedBackup := range listedBackups {
		backupIDs = append(backupIDs, listedBackup.ID)
	}

	return backupIDs
}
