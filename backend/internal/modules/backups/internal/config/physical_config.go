package config

import (
	"time"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/databases"
)

type PhysicalRetentionPolicy string

const (
	PhysicalRetentionPolicyChains               PhysicalRetentionPolicy = "CHAINS"
	PhysicalRetentionPolicyFullBackups          PhysicalRetentionPolicy = "FULL_BACKUPS"
	PhysicalRetentionPolicyChainsAndFullBackups PhysicalRetentionPolicy = "CHAINS_AND_FULL_BACKUPS"
)

type FullBackupsRetentionPolicy string

const (
	FullBackupsRetentionPolicyLastN FullBackupsRetentionPolicy = "LAST_N"
	FullBackupsRetentionPolicyGfs   FullBackupsRetentionPolicy = "GFS"
)

// FullBackupsRetention keeps the last Count full backups or keeps them by Gfs. Its zero value is the
// retention of a config that keeps only chains.
type FullBackupsRetention struct {
	Policy FullBackupsRetentionPolicy
	Count  int
	Gfs    GfsRetention
}

func (r FullBackupsRetention) IsZero() bool {
	return r.Policy == "" && r.Count == 0 && r.Gfs.IsZero()
}

func (r FullBackupsRetention) validate() error {
	switch r.Policy {
	case FullBackupsRetentionPolicyLastN:
		if r.Count <= 0 {
			return newInvalidSettingError("retention.fullBackups.count", "must be greater than 0")
		}

		if r.Gfs.HasAnyPeriod() {
			return newInvalidSettingError("retention.fullBackups", "must not keep GFS periods with LAST_N")
		}

	case FullBackupsRetentionPolicyGfs:
		if r.Count != 0 {
			return newInvalidSettingError("retention.fullBackups.count", "must be 0 with GFS")
		}

		if !r.Gfs.HasAnyPeriod() {
			return newInvalidSettingError("retention.fullBackups", "must keep at least one GFS period")
		}

	default:
		return newInvalidSettingError("retention.fullBackups.policy", "is not a full backups retention policy")
	}

	return nil
}

// PhysicalRetention keeps the last ChainsCount chains, full backups by FullBackups, or both, whichever
// Policy names.
type PhysicalRetention struct {
	Policy      PhysicalRetentionPolicy
	ChainsCount int
	FullBackups FullBackupsRetention
}

type PhysicalBackupConfig struct {
	BackupConfig
	// BackupType belongs to the database; the config is read with it and must fit it.
	BackupType         databases.PhysicalBackupType
	FullBackupSchedule schedule.Schedule
	// IncrementalBackupSchedule is nil for a backup type without incremental backups.
	IncrementalBackupSchedule *schedule.Schedule
	Retention                 PhysicalRetention
	// WalLagThresholdBytes is the WAL lag after which the replication slot is rebuilt, 0 without WAL
	// streaming.
	WalLagThresholdBytes int64
	// The force requests are the times a backup was asked for outside the schedule, nil when none waits.
	ForceFullRequestedAt        *time.Time
	ForceIncrementalRequestedAt *time.Time
}

func (c PhysicalBackupConfig) GetBackupConfig() BackupConfig {
	return c.BackupConfig
}

func (c PhysicalBackupConfig) GetMethod() Method {
	return MethodPhysical
}

// IsWalStreamingExpected is true exactly for the configs the WAL stream supervisor runs a streamer
// for; the WAL stream statistics call every other config "not streaming".
func (c PhysicalBackupConfig) IsWalStreamingExpected() bool {
	if !c.IsBackupsEnabled || c.StorageID == nil {
		return false
	}

	return c.BackupType.IsWalStreaming()
}

func getPhysicalNotificationTriggers() []NotificationTrigger {
	return []NotificationTrigger{
		NotificationTriggerBackupFailed,
		NotificationTriggerBackupSuccess,
		NotificationTriggerChainBroken,
		NotificationTriggerWalGap,
	}
}

func (c PhysicalBackupConfig) Validate() error {
	err := validateSchedule("fullBackupSchedule", c.FullBackupSchedule)
	if err != nil {
		return err
	}

	if c.WalLagThresholdBytes < 0 {
		return newInvalidSettingError("walLagThresholdBytes", "must not be negative")
	}

	err = c.validateRetention()
	if err != nil {
		return err
	}

	err = c.validateForBackupType()
	if err != nil {
		return err
	}

	return validateCommonFields(c.BackupConfig, getPhysicalNotificationTriggers())
}

func (c PhysicalBackupConfig) validateRetention() error {
	switch c.Retention.Policy {
	case PhysicalRetentionPolicyChains:
		if c.Retention.ChainsCount <= 0 {
			return newInvalidSettingError("retention.chainsCount", "must be greater than 0")
		}

		if !c.Retention.FullBackups.IsZero() {
			return newInvalidSettingError("retention.fullBackups", "must be empty with CHAINS")
		}

		return nil

	case PhysicalRetentionPolicyFullBackups:
		if c.Retention.ChainsCount != 0 {
			return newInvalidSettingError("retention.chainsCount", "must be 0 with FULL_BACKUPS")
		}

		return c.Retention.FullBackups.validate()

	case PhysicalRetentionPolicyChainsAndFullBackups:
		if c.Retention.ChainsCount <= 0 {
			return newInvalidSettingError("retention.chainsCount", "must be greater than 0")
		}

		return c.Retention.FullBackups.validate()

	default:
		return newInvalidSettingError("retention.policy", "is not a physical retention policy")
	}
}

func (c PhysicalBackupConfig) validateForBackupType() error {
	switch c.BackupType {
	case databases.PhysicalBackupTypeFull:
		if c.Retention.Policy != PhysicalRetentionPolicyFullBackups {
			return newInvalidSettingError("retention.policy", "must be FULL_BACKUPS without incremental backups")
		}

		if c.IncrementalBackupSchedule != nil {
			return newInvalidSettingError("incrementalBackupSchedule", "must be empty without incremental backups")
		}

		if c.WalLagThresholdBytes != 0 {
			return newInvalidSettingError("walLagThresholdBytes", "must be 0 without WAL streaming")
		}

		return nil

	case databases.PhysicalBackupTypeFullAndIncremental, databases.PhysicalBackupTypeFullIncrementalWalStream:
		if c.Retention.Policy == PhysicalRetentionPolicyFullBackups {
			return newInvalidSettingError("retention.policy", "must keep chains with incremental backups")
		}

		err := c.validateIncrementalSchedule()
		if err != nil {
			return err
		}

		isWalStreaming := c.BackupType.IsWalStreaming()
		if !isWalStreaming && c.WalLagThresholdBytes != 0 {
			return newInvalidSettingError("walLagThresholdBytes", "must be 0 without WAL streaming")
		}

		if isWalStreaming && c.WalLagThresholdBytes <= 0 {
			return newInvalidSettingError("walLagThresholdBytes", "must be greater than 0 with WAL streaming")
		}

		return nil

	default:
		return errUnknownBackupType
	}
}

func (c PhysicalBackupConfig) validateIncrementalSchedule() error {
	if c.IncrementalBackupSchedule == nil {
		return newInvalidSettingError("incrementalBackupSchedule", "is required with incremental backups")
	}

	err := validateSchedule("incrementalBackupSchedule", *c.IncrementalBackupSchedule)
	if err != nil {
		return err
	}

	if !isRunningMoreOften(*c.IncrementalBackupSchedule, c.FullBackupSchedule) {
		return newInvalidSettingError("incrementalBackupSchedule", "must run more often than the full backups")
	}

	return nil
}

// isRunningMoreOften cannot compare a cron schedule, so it lets one through.
func isRunningMoreOften(incrementalSchedule, fullSchedule schedule.Schedule) bool {
	if incrementalSchedule.Type == schedule.TypeCron || fullSchedule.Type == schedule.TypeCron {
		return true
	}

	incrementalPeriod := incrementalSchedule.GetApproxPeriod()
	fullPeriod := fullSchedule.GetApproxPeriod()

	if incrementalPeriod == 0 || fullPeriod == 0 {
		return false
	}

	return incrementalPeriod < fullPeriod
}
