package logical

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups/internal/config"
)

func TestGetDueBackups(t *testing.T) {
	t.Parallel()

	todayAt0430 := time.Date(2026, 10, 27, 4, 30, 0, 0, time.UTC)
	yesterdayAt0400 := time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC)
	todayAt0400 := time.Date(2026, 10, 27, 4, 0, 0, 0, time.UTC)

	t.Run("starts a daily backup once the time of day passes since yesterday's backup", func(t *testing.T) {
		t.Parallel()
		dailyConfig := newDailyConfig("04:00")
		lastBackups := []LogicalBackup{newBackup(dailyConfig.DatabaseID, StatusCompleted, yesterdayAt0400)}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{dailyConfig},
			map[uuid.UUID][]LogicalBackup{dailyConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Equal(t, []uuid.UUID{dailyConfig.DatabaseID}, dueDatabaseIDs)
	})

	t.Run("starts nothing when today's backup already ran", func(t *testing.T) {
		t.Parallel()
		dailyConfig := newDailyConfig("04:00")
		lastBackups := []LogicalBackup{newBackup(dailyConfig.DatabaseID, StatusCompleted, todayAt0400)}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{dailyConfig},
			map[uuid.UUID][]LogicalBackup{dailyConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})

	t.Run("starts the first backup of a database without backups", func(t *testing.T) {
		t.Parallel()
		dailyConfig := newDailyConfig("04:00")

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{dailyConfig},
			map[uuid.UUID][]LogicalBackup{}, todayAt0430)

		assert.Equal(t, []uuid.UUID{dailyConfig.DatabaseID}, dueDatabaseIDs)
	})

	t.Run("skips a database whose backup is running even when the schedule is due", func(t *testing.T) {
		t.Parallel()
		dailyConfig := newDailyConfig("04:00")
		lastBackups := []LogicalBackup{
			newBackup(dailyConfig.DatabaseID, StatusInProgress, todayAt0400.Add(-48*time.Hour)),
		}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{dailyConfig},
			map[uuid.UUID][]LogicalBackup{dailyConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})

	t.Run("skips a config that names no storage", func(t *testing.T) {
		t.Parallel()
		storagelessConfig := newDailyConfig("04:00")
		storagelessConfig.StorageID = nil

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{storagelessConfig},
			map[uuid.UUID][]LogicalBackup{}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})

	t.Run("retries a failed backup while fewer of the newest backups failed than the tries allow",
		func(t *testing.T) {
			t.Parallel()
			retryingConfig := newRetryingConfig(3)
			lastBackups := []LogicalBackup{
				newBackup(retryingConfig.DatabaseID, StatusFailed, todayAt0400.Add(10*time.Minute)),
				newBackup(retryingConfig.DatabaseID, StatusFailed, todayAt0400),
				newBackup(retryingConfig.DatabaseID, StatusCompleted, yesterdayAt0400),
			}

			dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{retryingConfig},
				map[uuid.UUID][]LogicalBackup{retryingConfig.DatabaseID: lastBackups}, todayAt0430)

			assert.Equal(t, []uuid.UUID{retryingConfig.DatabaseID}, dueDatabaseIDs)
		})

	t.Run("stops retrying once the last three backups failed with three tries", func(t *testing.T) {
		t.Parallel()
		retryingConfig := newRetryingConfig(3)
		lastBackups := []LogicalBackup{
			newBackup(retryingConfig.DatabaseID, StatusFailed, todayAt0400.Add(20*time.Minute)),
			newBackup(retryingConfig.DatabaseID, StatusFailed, todayAt0400.Add(10*time.Minute)),
			newBackup(retryingConfig.DatabaseID, StatusFailed, todayAt0400),
		}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{retryingConfig},
			map[uuid.UUID][]LogicalBackup{retryingConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})

	t.Run("does not retry a failed backup when the config does not retry", func(t *testing.T) {
		t.Parallel()
		nonRetryingConfig := newRetryingConfig(3)
		nonRetryingConfig.IsRetryIfFailed = false
		lastBackups := []LogicalBackup{newBackup(nonRetryingConfig.DatabaseID, StatusFailed, todayAt0400)}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{nonRetryingConfig},
			map[uuid.UUID][]LogicalBackup{nonRetryingConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})

	t.Run("does not retry when the last backup was cancelled", func(t *testing.T) {
		t.Parallel()
		retryingConfig := newRetryingConfig(3)
		lastBackups := []LogicalBackup{newBackup(retryingConfig.DatabaseID, StatusCanceled, todayAt0400)}

		dueDatabaseIDs := getDueBackups([]config.LogicalBackupConfig{retryingConfig},
			map[uuid.UUID][]LogicalBackup{retryingConfig.DatabaseID: lastBackups}, todayAt0430)

		assert.Empty(t, dueDatabaseIDs)
	})
}

func newDailyConfig(timeOfDay string) config.LogicalBackupConfig {
	storageID := uuid.New()

	return config.LogicalBackupConfig{
		BackupConfig: config.BackupConfig{
			DatabaseID:           uuid.New(),
			IsBackupsEnabled:     true,
			StorageID:            &storageID,
			Encryption:           config.EncryptionNone,
			NotificationTriggers: nil,
		},
		Schedule: schedule.Schedule{
			Type:           schedule.TypeDaily,
			TimeOfDay:      &timeOfDay,
			Weekday:        nil,
			DayOfMonth:     nil,
			CronExpression: nil,
		},
		Retention: config.LogicalRetention{
			Policy: config.LogicalRetentionPolicyTimePeriod,
			Period: config.RetentionPeriodForever,
			Count:  0,
			Gfs:    config.GfsRetention{},
		},
		IsRetryIfFailed:     false,
		MaxFailedTriesCount: 0,
	}
}

func newRetryingConfig(maxFailedTriesCount int) config.LogicalBackupConfig {
	retryingConfig := newDailyConfig("04:00")
	retryingConfig.IsRetryIfFailed = true
	retryingConfig.MaxFailedTriesCount = maxFailedTriesCount

	return retryingConfig
}

func newBackup(databaseID uuid.UUID, status Status, createdAt time.Time) LogicalBackup {
	return LogicalBackup{
		ID:                        uuid.New(),
		DatabaseID:                databaseID,
		StorageID:                 uuid.New(),
		FileName:                  "backup",
		Status:                    status,
		FailMessage:               nil,
		BackupSizeMb:              0,
		BackupRawDbSizeMb:         0,
		BackupDurationMs:          0,
		Encryption:                config.EncryptionNone,
		EncryptionSalt:            nil,
		EncryptionIV:              nil,
		TimescaledbVersion:        "",
		RestoreVerificationStatus: VerificationStatusNotVerified,
		CreatedAt:                 createdAt,
	}
}
