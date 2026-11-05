package backups_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

	t.Run(
		"retries only the database whose newest backups stay under its tries with long histories",
		func(t *testing.T) {
			tm := newTeam(t)
			exhaustedDatabaseID, exhaustedStorageID := createDroppedSourceDatabase(t, tm, server)
			retriedDatabaseID, retriedStorageID := createDroppedSourceDatabase(t, tm, server)
			retryingSettings := newDailyBackupSettings()
			retryingSettings.IsRetryIfFailed = true
			retryingSettings.MaxFailedTriesCount = 3
			saveLogicalBackupSettings(t, tm.owner, exhaustedDatabaseID, exhaustedStorageID, retryingSettings,
				apiclient.BackupEncryptionNONE)
			saveLogicalBackupSettings(t, tm.owner, retriedDatabaseID, retriedStorageID, retryingSettings,
				apiclient.BackupEncryptionNONE)

			slot := getNextDailySlot()
			tm.app.Clock.Set(slot.Add(time.Hour))
			newestBackupTime := slot.Add(30 * time.Minute)

			failed := []apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusFAILED}
			completed := []apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusCOMPLETED}
			exhaustedNewestBackupID := insertLogicalBackupHistory(t, tm, logicalBackupHistory{
				databaseID:       exhaustedDatabaseID,
				storageID:        exhaustedStorageID,
				newestBackupTime: newestBackupTime,
				statuses:         slices.Concat(slices.Repeat(failed, 3), slices.Repeat(completed, 40)),
			})
			retriedNewestBackupID := insertLogicalBackupHistory(t, tm, logicalBackupHistory{
				databaseID:       retriedDatabaseID,
				storageID:        retriedStorageID,
				newestBackupTime: newestBackupTime,
				statuses:         slices.Concat(failed, slices.Repeat(completed, 2), slices.Repeat(failed, 40)),
			})

			tm.app.RunJobPass(t, logicalBackupSchedulerJobName)

			assert.Equal(t, exhaustedNewestBackupID, listLogicalBackupIDs(t, tm.owner, exhaustedDatabaseID)[0])
			assert.Zero(t, countRunningLogicalBackups(t, tm.owner, exhaustedDatabaseID))
			assert.NotEqual(t, retriedNewestBackupID, listLogicalBackupIDs(t, tm.owner, retriedDatabaseID)[0])
			waitForNewestBackupToFail(t, tm, retriedDatabaseID)
		},
	)
}

// logicalBackupHistory lists statuses newest first, one minute apart from newestBackupTime.
type logicalBackupHistory struct {
	databaseID       uuid.UUID
	storageID        uuid.UUID
	newestBackupTime time.Time
	statuses         []apiclient.LogicalBackupStatus
}

// insertLogicalBackupHistory writes finished backups directly, because the API cannot create past ones,
// and returns the ID of the newest. It writes the oldest first, as time would, so a read that forgets to
// order by time gets the oldest rows.
func insertLogicalBackupHistory(t *testing.T, tm team, history logicalBackupHistory) uuid.UUID {
	t.Helper()

	var newestBackupID uuid.UUID

	for position, status := range slices.Backward(history.statuses) {
		newestBackupID = uuid.New()
		tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			newestBackupID, history.databaseID, history.storageID, string(status), "history-"+newestBackupID.String(),
			history.newestBackupTime.Add(-time.Duration(position)*time.Minute))
	}

	return newestBackupID
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

	waitForNewestBackupToFail(t, tm, databaseID)
}

func waitForNewestBackupToFail(t *testing.T, tm team, databaseID uuid.UUID) {
	t.Helper()

	testwait.Until(t, "the scheduled backup fails", func(context.Context) error {
		newestBackup := getLogicalBackupPage(t, tm.owner, newLogicalBackupListParams(databaseID)).Items[0]
		if newestBackup.Status != apiclient.LogicalBackupStatusFAILED {
			return fmt.Errorf("%w: %s", errScheduledBackupNotFailed, newestBackup.Status)
		}

		return nil
	})
}
