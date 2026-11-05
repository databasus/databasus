package config

import (
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/databases"
)

// coerceToBackupType reshapes the config for the backup type of its database, keeping every value the
// type allows. The database owns the type, so a switch there would otherwise leave a config that the
// validation refuses and that cannot be edited back into shape.
func (c PhysicalBackupConfig) coerceToBackupType(backupType databases.PhysicalBackupType) PhysicalBackupConfig {
	coercedConfig := c
	coercedConfig.BackupType = backupType

	switch backupType {
	case databases.PhysicalBackupTypeFull:
		coercedConfig.coerceForFullOnly()

	case databases.PhysicalBackupTypeFullAndIncremental:
		coercedConfig.coerceForIncremental()
		coercedConfig.WalLagThresholdBytes = 0

	case databases.PhysicalBackupTypeFullIncrementalWalStream:
		coercedConfig.coerceForIncremental()

		if coercedConfig.WalLagThresholdBytes <= 0 {
			coercedConfig.WalLagThresholdBytes = defaultWalLagThresholdBytes
		}
	}

	return coercedConfig
}

func (c *PhysicalBackupConfig) coerceForFullOnly() {
	c.IncrementalBackupSchedule = nil
	c.WalLagThresholdBytes = 0
	c.Retention.Policy = PhysicalRetentionPolicyFullBackups
	c.Retention.ChainsCount = 0

	if c.Retention.FullBackups.validate() != nil {
		c.Retention.FullBackups = defaultFullBackupsRetention()
	}
}

func (c *PhysicalBackupConfig) coerceForIncremental() {
	if c.Retention.Policy != PhysicalRetentionPolicyChains &&
		c.Retention.Policy != PhysicalRetentionPolicyChainsAndFullBackups {
		c.Retention.Policy = PhysicalRetentionPolicyChains
		if c.Retention.FullBackups.validate() == nil {
			c.Retention.Policy = PhysicalRetentionPolicyChainsAndFullBackups
		}
	}

	if c.Retention.Policy == PhysicalRetentionPolicyChains {
		c.Retention.FullBackups = FullBackupsRetention{}
	}

	if c.Retention.ChainsCount <= 0 {
		c.Retention.ChainsCount = defaultChainsRetentionCount
	}

	if c.IncrementalBackupSchedule != nil &&
		c.IncrementalBackupSchedule.Validate() == nil &&
		isRunningMoreOften(*c.IncrementalBackupSchedule, c.FullBackupSchedule) {
		return
	}

	c.IncrementalBackupSchedule = buildScheduleMoreOftenThan(c.FullBackupSchedule)
}

// buildScheduleMoreOftenThan returns nil for an hourly full schedule: no fixed schedule runs more often,
// so the owner has to slow the full backups down first, and the validation says so.
func buildScheduleMoreOftenThan(fullSchedule schedule.Schedule) *schedule.Schedule {
	switch fullSchedule.Type {
	case schedule.TypeDaily:
		return &schedule.Schedule{
			Type:           schedule.TypeHourly,
			TimeOfDay:      nil,
			Weekday:        nil,
			DayOfMonth:     nil,
			CronExpression: nil,
		}

	case schedule.TypeWeekly, schedule.TypeMonthly, schedule.TypeCron:
		timeOfDay := defaultTimeOfDay
		if fullSchedule.TimeOfDay != nil {
			timeOfDay = *fullSchedule.TimeOfDay
		}

		return &schedule.Schedule{
			Type:           schedule.TypeDaily,
			TimeOfDay:      &timeOfDay,
			Weekday:        nil,
			DayOfMonth:     nil,
			CronExpression: nil,
		}

	case schedule.TypeHourly:
		return nil

	default:
		return nil
	}
}
