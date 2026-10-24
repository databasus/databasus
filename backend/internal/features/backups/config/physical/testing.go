//go:build testhelpers

package backups_config_physical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/storages"
)

func EnableBackupsForPhysicalTestDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
	storage *storages.Storage,
) *PhysicalBackupConfig {
	timeOfDay := "16:00"

	backupConfig := &PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: true,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Retention: RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  7,
		},
		StorageID: &storage.ID,
		Storage:   storage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
			NotificationChainBroken,
		},
	}

	backupConfig, err := GetBackupConfigService().SaveBackupConfig(ctx, backupConfig)
	if err != nil {
		panic(err)
	}

	return backupConfig
}
