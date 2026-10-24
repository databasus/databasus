//go:build testhelpers

package backups_config_logical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/period"
)

func EnableBackupsForTestDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
	storage *storages.Storage,
) *LogicalBackupConfig {
	timeOfDay := "16:00"

	backupConfig := &LogicalBackupConfig{
		DatabaseID:          databaseID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodDay,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		StorageID: &storage.ID,
		Storage:   storage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
	}

	backupConfig, err := GetBackupConfigService().SaveBackupConfig(ctx, backupConfig)
	if err != nil {
		panic(err)
	}

	return backupConfig
}
