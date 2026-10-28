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

func SaveLogicalTestBackupConfig(ctx context.Context, config backups.LogicalBackupConfig) backups.LogicalBackupConfig {
	savedConfig, err := GetModule().SaveBackupConfig(ctx, testHelperActor, config)
	if err != nil {
		panic(fmt.Sprintf("save the logical test backup config: %v", err))
	}
	logicalConfig, isLogical := savedConfig.(backups.LogicalBackupConfig)
	if !isLogical {
		panic(fmt.Sprintf("the save returned a %T config for a logical one", savedConfig))
	}
	return logicalConfig
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

func NewDailyTestSchedule(timeOfDay string) schedule.Schedule {
	return schedule.Schedule{
		Type:           schedule.TypeDaily,
		TimeOfDay:      &timeOfDay,
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: nil,
	}
}
