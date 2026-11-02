package backups_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const logicalBackupSchedulerJobName = "backups_logical_scheduler"

var errScheduledBackupNotFailed = errors.New("the scheduled backup has not failed yet")

//nolint:tparallel // The subtests share one source server.
func TestLogicalBackupScheduler(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("starts a backup at the next check once the daily time passes", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, newDailyBackupSettings(),
			apiclient.BackupEncryptionNONE)
		yesterdaySlot := getNextDailySlot()
		tm.app.Clock.Set(yesterdaySlot.Add(30 * time.Second))
		yesterdayBackup := completeRestorableBackup(t, tm.owner, source.databaseID)

		tm.app.Clock.Set(yesterdaySlot.Add(24*time.Hour - time.Minute))
		tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

		require.Equal(t, []uuid.UUID{yesterdayBackup.Id}, listLogicalBackupIDs(t, tm.owner, source.databaseID))

		tm.app.Clock.Set(yesterdaySlot.Add(24*time.Hour + 30*time.Second))
		tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

		listedBackupIDs := listLogicalBackupIDs(t, tm.owner, source.databaseID)
		require.Len(t, listedBackupIDs, 2)
		assert.Equal(t, yesterdayBackup.Id, listedBackupIDs[1])
		waitForRestorableBackupStatus(t, tm.owner, listedBackupIDs[0], apiclient.LogicalBackupStatusCOMPLETED)
	})

	t.Run("starts no backup when a stored cron expression has no date to run on", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		tm.app.ExecuteSQL(t, `UPDATE backups_configs_logical
			SET interval_type = 'CRON', cron_expression = ? WHERE database_id = ?`,
			cronWithoutRunDate, source.databaseID)

		tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

		assert.Empty(t, listLogicalBackupIDs(t, tm.owner, source.databaseID))
	})

	t.Run("stops retrying after the configured tries until the schedule makes the next backup due", func(t *testing.T) {
		tm := newTeam(t)
		databaseID, storageID := createDroppedSourceDatabase(t, tm, server)
		retryingSettings := newDailyBackupSettings()
		retryingSettings.IsRetryIfFailed = true
		retryingSettings.MaxFailedTriesCount = 3
		saveLogicalBackupSettings(t, tm.owner, databaseID, storageID, retryingSettings, apiclient.BackupEncryptionNONE)
		firstSlot := getNextDailySlot()
		tm.app.Clock.Set(firstSlot.Add(30 * time.Second))

		for tryIndex := range 3 {
			runSchedulerPassUntilBackupFails(t, tm, databaseID, tryIndex+1)
			tm.app.Clock.Advance(time.Minute)
		}

		tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

		assert.Len(t, listLogicalBackupIDs(t, tm.owner, databaseID), 3)
		assert.Zero(t, countRunningLogicalBackups(t, tm.owner, databaseID))

		tm.app.Clock.Set(firstSlot.Add(24*time.Hour + 30*time.Second))
		runSchedulerPassUntilBackupFails(t, tm, databaseID, 4)
	})
}

// newDailyBackupSettings backs up every day at 04:00 UTC and never retries a failed backup.
func newDailyBackupSettings() apiclient.LogicalBackupSettings {
	dailySettings := newLogicalBackupSettings()
	dailySettings.Schedule = apiclient.Schedule{
		Type:           apiclient.ScheduleTypeDAILY,
		TimeOfDay:      new("04:00"),
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: nil,
	}

	return dailySettings
}

// getNextDailySlot is 04:00 UTC of a day after the real time, so moving the clock there only moves it
// forward.
func getNextDailySlot() time.Time {
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)

	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 4, 0, 0, 0, time.UTC)
}

// createDroppedSourceDatabase creates a database through the API and then drops it on the server, so
// every backup of it fails.
func createDroppedSourceDatabase(t *testing.T, tm team, server restoreServer) (uuid.UUID, uuid.UUID) {
	t.Helper()

	sourceName := server.createSourceDatabase(t)
	createdDatabase := apptest.CreateDatabase(t, tm.owner, tm.workspace.Id, server.newDatabaseSettings(t, sourceName))
	server.exec(t, containers.PostgresDatabase, "DROP DATABASE "+sourceName+" WITH (FORCE)")

	return createdDatabase.Id, tm.createLocalStorage(t)
}

// runSchedulerPassUntilBackupFails runs a scheduler pass, checks that it started the database's
// backupCount-th backup and waits until that backup fails.
func runSchedulerPassUntilBackupFails(t *testing.T, tm team, databaseID uuid.UUID, backupCount int) {
	t.Helper()

	tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

	require.Len(t, listLogicalBackupIDs(t, tm.owner, databaseID), backupCount)

	testwait.Until(t, "the scheduled backup fails", func(context.Context) error {
		newestBackup := getLogicalBackupPage(t, tm.owner, newLogicalBackupListParams(databaseID)).Items[0]
		if newestBackup.Status != apiclient.LogicalBackupStatusFAILED {
			return fmt.Errorf("%w: %s", errScheduledBackupNotFailed, newestBackup.Status)
		}

		return nil
	})
}
