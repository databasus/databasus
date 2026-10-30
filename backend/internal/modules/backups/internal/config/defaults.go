package config

import (
	"slices"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/databases"
)

// The migration that created backups_configs repairs unknown stored values to these defaults and
// writes them as literals; a change here does not reach rows repaired before.
const (
	defaultTimeOfDay                 = "04:00"
	defaultLogicalRetentionPeriod    = RetentionPeriod3Months
	defaultMaxFailedTriesCount       = 3
	defaultChainsRetentionCount      = 7
	defaultFullBackupsRetentionCount = 7
	defaultWalLagThresholdBytes      = 256 * 1024 * 1024
)

func newDefaultLogicalBackupConfig(databaseID uuid.UUID) LogicalBackupConfig {
	return LogicalBackupConfig{
		BackupConfig: BackupConfig{
			DatabaseID:       databaseID,
			IsBackupsEnabled: false,
			StorageID:        nil,
			Encryption:       EncryptionNone,
			NotificationTriggers: []NotificationTrigger{
				NotificationTriggerBackupFailed,
				NotificationTriggerBackupSuccess,
			},
		},
		Schedule: newDailySchedule(),
		Retention: LogicalRetention{
			Policy: LogicalRetentionPolicyTimePeriod,
			Period: defaultLogicalRetentionPeriod,
			Count:  0,
			Gfs:    GfsRetention{},
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: defaultMaxFailedTriesCount,
	}
}

// newDefaultPhysicalBackupConfig fits the default to the database's backup type, as a type change
// fits a saved config.
func newDefaultPhysicalBackupConfig(
	databaseID uuid.UUID,
	backupType databases.PhysicalBackupType,
) PhysicalBackupConfig {
	defaultConfig := PhysicalBackupConfig{
		BackupConfig: BackupConfig{
			DatabaseID:           databaseID,
			IsBackupsEnabled:     false,
			StorageID:            nil,
			Encryption:           EncryptionNone,
			NotificationTriggers: slices.Clone(getPhysicalNotificationTriggers()),
		},
		BackupType:                backupType,
		FullBackupSchedule:        newDailySchedule(),
		IncrementalBackupSchedule: nil,
		Retention: PhysicalRetention{
			Policy:      PhysicalRetentionPolicyFullBackups,
			ChainsCount: 0,
			FullBackups: defaultFullBackupsRetention(),
		},
		WalLagThresholdBytes:        0,
		ForceFullRequestedAt:        nil,
		ForceIncrementalRequestedAt: nil,
	}

	return defaultConfig.coerceToBackupType(backupType)
}

func newDailySchedule() schedule.Schedule {
	timeOfDay := defaultTimeOfDay

	return schedule.Schedule{
		Type:           schedule.TypeDaily,
		TimeOfDay:      &timeOfDay,
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: nil,
	}
}

func defaultFullBackupsRetention() FullBackupsRetention {
	return FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyLastN,
		Count:  defaultFullBackupsRetentionCount,
		Gfs:    GfsRetention{},
	}
}
