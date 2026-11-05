package httpapi

import (
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
)

// toBackupConfig picks the method by the discriminator of the settings. The force requests are not
// part of the contract; the save keeps the stored ones.
func toBackupConfig(databaseID uuid.UUID, body BackupConfigInput) (config.AnyBackupConfig, error) {
	settingsBody, err := body.Settings.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownSettingsValue
	}

	sharedConfig := config.BackupConfig{
		DatabaseID:           databaseID,
		IsBackupsEnabled:     body.IsBackupsEnabled,
		StorageID:            body.StorageId,
		Encryption:           config.Encryption(body.Encryption),
		NotificationTriggers: nil,
	}

	switch typedSettings := settingsBody.(type) {
	case LogicalBackupSettings:
		return toLogicalBackupConfig(sharedConfig, typedSettings), nil
	case PhysicalBackupSettings:
		return toPhysicalBackupConfig(sharedConfig, typedSettings), nil
	default:
		return nil, errUnknownSettingsValue
	}
}

func toBackupConfigBody(savedConfig config.AnyBackupConfig) (BackupConfig, error) {
	var settings BackupSettings

	var err error

	switch typedConfig := savedConfig.(type) {
	case config.LogicalBackupConfig:
		var logicalSettings LogicalBackupSettings

		logicalSettings, err = fromLogicalBackupConfig(typedConfig)
		if err == nil {
			err = settings.FromLogicalBackupSettings(logicalSettings)
		}
	case config.PhysicalBackupConfig:
		err = settings.FromPhysicalBackupSettings(fromPhysicalBackupConfig(typedConfig))
	default:
		return BackupConfig{}, fmt.Errorf("%w: %T", errUnknownConfigType, savedConfig)
	}

	if err != nil {
		return BackupConfig{}, fmt.Errorf("encode the backup settings: %w", err)
	}

	sharedConfig := savedConfig.GetBackupConfig()

	return BackupConfig{
		Cleanup: &BackupCleanup{
			PendingFileCount: sharedConfig.Cleanup.PendingFileCount,
			FailedFileCount:  sharedConfig.Cleanup.FailedFileCount,
		},
		DatabaseId:       sharedConfig.DatabaseID,
		IsBackupsEnabled: sharedConfig.IsBackupsEnabled,
		StorageId:        sharedConfig.StorageID,
		Encryption:       BackupEncryption(sharedConfig.Encryption),
		Settings:         settings,
	}, nil
}

func toLogicalBackupConfig(sharedConfig config.BackupConfig, body LogicalBackupSettings) config.LogicalBackupConfig {
	sharedConfig.NotificationTriggers = make([]config.NotificationTrigger, 0, len(body.NotificationTriggers))
	for _, trigger := range body.NotificationTriggers {
		sharedConfig.NotificationTriggers = append(
			sharedConfig.NotificationTriggers,
			config.NotificationTrigger(trigger),
		)
	}

	return config.LogicalBackupConfig{
		BackupConfig: sharedConfig,
		Schedule:     toSchedule(body.Schedule),
		Retention: config.LogicalRetention{
			Policy: config.LogicalRetentionPolicy(body.Retention.Policy),
			Period: toRetentionPeriod(body.Retention.Period),
			Count:  body.Retention.Count,
			Gfs: config.GfsRetention{
				Hours:  body.Retention.GfsHours,
				Days:   body.Retention.GfsDays,
				Weeks:  body.Retention.GfsWeeks,
				Months: body.Retention.GfsMonths,
				Years:  body.Retention.GfsYears,
			},
		},
		IsRetryIfFailed:     body.IsRetryIfFailed,
		MaxFailedTriesCount: body.MaxFailedTriesCount,
	}
}

func fromLogicalBackupConfig(savedConfig config.LogicalBackupConfig) (LogicalBackupSettings, error) {
	period, err := fromRetentionPeriod(savedConfig.Retention.Period)
	if err != nil {
		return LogicalBackupSettings{}, err
	}

	triggers := make([]LogicalNotificationTrigger, 0, len(savedConfig.NotificationTriggers))
	for _, trigger := range savedConfig.NotificationTriggers {
		triggers = append(triggers, LogicalNotificationTrigger(trigger))
	}

	return LogicalBackupSettings{
		Type:     LogicalBackupSettingsTypeLOGICAL,
		Schedule: fromSchedule(savedConfig.Schedule),
		Retention: LogicalRetention{
			Policy:    LogicalRetentionPolicy(savedConfig.Retention.Policy),
			Period:    period,
			Count:     savedConfig.Retention.Count,
			GfsHours:  savedConfig.Retention.Gfs.Hours,
			GfsDays:   savedConfig.Retention.Gfs.Days,
			GfsWeeks:  savedConfig.Retention.Gfs.Weeks,
			GfsMonths: savedConfig.Retention.Gfs.Months,
			GfsYears:  savedConfig.Retention.Gfs.Years,
		},
		IsRetryIfFailed:      savedConfig.IsRetryIfFailed,
		MaxFailedTriesCount:  savedConfig.MaxFailedTriesCount,
		NotificationTriggers: triggers,
	}, nil
}

func toPhysicalBackupConfig(sharedConfig config.BackupConfig, body PhysicalBackupSettings) config.PhysicalBackupConfig {
	sharedConfig.NotificationTriggers = make([]config.NotificationTrigger, 0, len(body.NotificationTriggers))
	for _, trigger := range body.NotificationTriggers {
		sharedConfig.NotificationTriggers = append(
			sharedConfig.NotificationTriggers,
			config.NotificationTrigger(trigger),
		)
	}

	physicalConfig := config.PhysicalBackupConfig{
		BackupConfig:              sharedConfig,
		BackupType:                "",
		FullBackupSchedule:        toSchedule(body.FullBackupSchedule),
		IncrementalBackupSchedule: nil,
		Retention: config.PhysicalRetention{
			Policy:      config.PhysicalRetentionPolicy(body.Retention.Policy),
			ChainsCount: body.Retention.ChainsCount,
			FullBackups: config.FullBackupsRetention{
				Policy: "",
				Count:  0,
				Gfs:    config.GfsRetention{Hours: 0, Days: 0, Weeks: 0, Months: 0, Years: 0},
			},
		},
		WalLagThresholdBytes:        body.WalLagThresholdBytes,
		ForceFullRequestedAt:        nil,
		ForceIncrementalRequestedAt: nil,
	}

	if body.IncrementalBackupSchedule != nil {
		incrementalSchedule := toSchedule(*body.IncrementalBackupSchedule)
		physicalConfig.IncrementalBackupSchedule = &incrementalSchedule
	}

	if body.Retention.FullBackups != nil {
		physicalConfig.Retention.FullBackups = config.FullBackupsRetention{
			Policy: config.FullBackupsRetentionPolicy(body.Retention.FullBackups.Policy),
			Count:  body.Retention.FullBackups.Count,
			Gfs: config.GfsRetention{
				Hours:  body.Retention.FullBackups.GfsHours,
				Days:   body.Retention.FullBackups.GfsDays,
				Weeks:  body.Retention.FullBackups.GfsWeeks,
				Months: body.Retention.FullBackups.GfsMonths,
				Years:  body.Retention.FullBackups.GfsYears,
			},
		}
	}

	return physicalConfig
}

func fromPhysicalBackupConfig(savedConfig config.PhysicalBackupConfig) PhysicalBackupSettings {
	triggers := make([]NotificationTrigger, 0, len(savedConfig.NotificationTriggers))
	for _, trigger := range savedConfig.NotificationTriggers {
		triggers = append(triggers, NotificationTrigger(trigger))
	}

	settings := PhysicalBackupSettings{
		Type:                      PhysicalBackupSettingsTypePHYSICAL,
		FullBackupSchedule:        fromSchedule(savedConfig.FullBackupSchedule),
		IncrementalBackupSchedule: nil,
		Retention: PhysicalRetention{
			Policy:      PhysicalRetentionPolicy(savedConfig.Retention.Policy),
			ChainsCount: savedConfig.Retention.ChainsCount,
			FullBackups: nil,
		},
		WalLagThresholdBytes: savedConfig.WalLagThresholdBytes,
		NotificationTriggers: triggers,
	}

	if savedConfig.IncrementalBackupSchedule != nil {
		incrementalSchedule := fromSchedule(*savedConfig.IncrementalBackupSchedule)
		settings.IncrementalBackupSchedule = &incrementalSchedule
	}

	if !savedConfig.Retention.FullBackups.IsZero() {
		fullBackups := savedConfig.Retention.FullBackups
		settings.Retention.FullBackups = &FullBackupsRetention{
			Policy:    FullBackupsRetentionPolicy(fullBackups.Policy),
			Count:     fullBackups.Count,
			GfsHours:  fullBackups.Gfs.Hours,
			GfsDays:   fullBackups.Gfs.Days,
			GfsWeeks:  fullBackups.Gfs.Weeks,
			GfsMonths: fullBackups.Gfs.Months,
			GfsYears:  fullBackups.Gfs.Years,
		}
	}

	return settings
}
