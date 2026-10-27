//go:build testhelpers

package backups_bridge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/users"
)

// testHelperActor is an instance admin, whom every workspace lets through, so the old tests save
// configs without a member of the database's workspace.
var testHelperActor = users.Actor{UserID: uuid.Nil, Role: users.RoleAdmin}

// SaveTestBackupConfig saves the config through the module's save, with its checks, and returns its error.
func SaveTestBackupConfig[C backups.LogicalBackupConfig | backups.PhysicalBackupConfig](
	ctx context.Context,
	config C,
) (C, error) {
	var noConfig C

	savedConfig, err := GetModule().SaveBackupConfig(ctx, testHelperActor, toAnyBackupConfig(config))
	if err != nil {
		return noConfig, err
	}

	typedConfig, isSameMethod := savedConfig.(C)
	if !isSameMethod {
		return noConfig, fmt.Errorf("the save returned a %T config for a %T one", savedConfig, config)
	}

	return typedConfig, nil
}

// SaveLogicalTestBackupConfig saves the config through the module's save, with its checks.
func SaveLogicalTestBackupConfig(ctx context.Context, config backups.LogicalBackupConfig) backups.LogicalBackupConfig {
	savedConfig, err := SaveTestBackupConfig(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("save the logical test backup config: %v", err))
	}

	return savedConfig
}

// SavePhysicalTestBackupConfig saves the config through the module's save, with its checks.
func SavePhysicalTestBackupConfig(
	ctx context.Context,
	config backups.PhysicalBackupConfig,
) backups.PhysicalBackupConfig {
	savedConfig, err := SaveTestBackupConfig(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("save the physical test backup config: %v", err))
	}

	return savedConfig
}

func toAnyBackupConfig[C backups.LogicalBackupConfig | backups.PhysicalBackupConfig](config C) backups.AnyBackupConfig {
	switch typedConfig := any(config).(type) {
	case backups.LogicalBackupConfig:
		return typedConfig
	case backups.PhysicalBackupConfig:
		return typedConfig
	default:
		panic(fmt.Sprintf("unknown backup config type %T", config))
	}
}

// EnableBackupsForTestDatabase saves a logical config that backs up daily at 16:00 into the storage and
// keeps backups for a day.
func EnableBackupsForTestDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) backups.LogicalBackupConfig {
	return SaveLogicalTestBackupConfig(ctx, backups.LogicalBackupConfig{
		BackupConfig: backups.BackupConfig{
			DatabaseID:       databaseID,
			IsBackupsEnabled: true,
			StorageID:        &storageID,
			Encryption:       backups.EncryptionNone,
			NotificationTriggers: []backups.NotificationTrigger{
				backups.NotificationTriggerBackupFailed,
				backups.NotificationTriggerBackupSuccess,
			},
		},
		Schedule: NewDailyTestSchedule("16:00"),
		Retention: backups.LogicalRetention{
			Policy: backups.LogicalRetentionPolicyTimePeriod,
			Period: backups.RetentionPeriodDay,
			Count:  0,
			Gfs:    backups.GfsRetention{},
		},
		IsRetryIfFailed:     false,
		MaxFailedTriesCount: 0,
	})
}

// EnableBackupsForPhysicalTestDatabase saves a physical config of a database that takes full backups
// only, backing up daily at 16:00 into the storage and keeping the last seven full backups.
func EnableBackupsForPhysicalTestDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) backups.PhysicalBackupConfig {
	return SavePhysicalTestBackupConfig(ctx, backups.PhysicalBackupConfig{
		BackupConfig: backups.BackupConfig{
			DatabaseID:       databaseID,
			IsBackupsEnabled: true,
			StorageID:        &storageID,
			Encryption:       backups.EncryptionNone,
			NotificationTriggers: []backups.NotificationTrigger{
				backups.NotificationTriggerBackupFailed,
				backups.NotificationTriggerBackupSuccess,
				backups.NotificationTriggerChainBroken,
			},
		},
		BackupType:                "",
		FullBackupSchedule:        NewDailyTestSchedule("16:00"),
		IncrementalBackupSchedule: nil,
		Retention: backups.PhysicalRetention{
			Policy:      backups.PhysicalRetentionPolicyFullBackups,
			ChainsCount: 0,
			FullBackups: backups.FullBackupsRetention{
				Policy: backups.FullBackupsRetentionPolicyLastN,
				Count:  7,
				Gfs:    backups.GfsRetention{},
			},
		},
		WalLagThresholdBytes:        0,
		ForceFullRequestedAt:        nil,
		ForceIncrementalRequestedAt: nil,
	})
}

func NewDailyTestSchedule(timeOfDay string) schedule.Schedule {
	return schedule.Schedule{
		Type:           schedule.TypeDaily,
		TimeOfDay:      &timeOfDay,
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: nil,
	}
}
