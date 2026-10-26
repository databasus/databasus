package backups_config_physical

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
)

type PhysicalBackupConfig struct {
	DatabaseID         uuid.UUID                                       `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey;not null"`
	PostgresqlPhysical *postgresql_physical.PostgresqlPhysicalDatabase `json:"-"          gorm:"foreignKey:DatabaseID;references:DatabaseID"`

	IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`

	FullBackupInterval        intervals.Interval `json:"fullBackupInterval"        gorm:"embedded;embeddedPrefix:full_"`
	IncrementalBackupInterval intervals.Interval `json:"incrementalBackupInterval" gorm:"embedded;embeddedPrefix:incremental_"`

	Retention            Retention            `json:"retention"            gorm:"column:retention;type:text;not null;default:'FULL_BACKUPS'"`
	ChainsRetention      ChainsRetention      `json:"chainsRetention"      gorm:"embedded;embeddedPrefix:chains_retention_"`
	FullBackupsRetention FullBackupsRetention `json:"fullBackupsRetention" gorm:"embedded;embeddedPrefix:full_backups_retention_"`

	WalLagThresholdBytes int64 `json:"walLagThresholdBytes" gorm:"column:wal_lag_threshold_bytes;type:bigint;not null;default:0"`

	ForceFullRequestedAt        *time.Time `json:"-" gorm:"column:force_full_requested_at;type:timestamptz"`
	ForceIncrementalRequestedAt *time.Time `json:"-" gorm:"column:force_incremental_requested_at;type:timestamptz"`

	Storage   *BackupConfigStorage `json:"storage"   gorm:"-"`
	StorageID *uuid.UUID           `json:"storageId" gorm:"column:storage_id;type:uuid"`

	Encryption backups_core_enums.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`

	SendNotificationsOn       []BackupNotificationType `json:"sendNotificationsOn" gorm:"-"`
	SendNotificationsOnString string                   `json:"-"                   gorm:"column:send_notifications_on;type:text;not null"`
}

func (b *PhysicalBackupConfig) TableName() string {
	return "physical_backup_configs"
}

func (b *PhysicalBackupConfig) BeforeSave(_ *gorm.DB) error {
	if len(b.SendNotificationsOn) > 0 {
		notificationTypes := make([]string, len(b.SendNotificationsOn))
		for i, t := range b.SendNotificationsOn {
			notificationTypes[i] = string(t)
		}
		b.SendNotificationsOnString = strings.Join(notificationTypes, ",")
	} else {
		b.SendNotificationsOnString = ""
	}

	return nil
}

func (b *PhysicalBackupConfig) AfterFind(_ *gorm.DB) error {
	if b.SendNotificationsOnString != "" {
		parts := strings.Split(b.SendNotificationsOnString, ",")
		b.SendNotificationsOn = make([]BackupNotificationType, len(parts))
		for i, p := range parts {
			b.SendNotificationsOn[i] = BackupNotificationType(p)
		}
	} else {
		b.SendNotificationsOn = []BackupNotificationType{}
	}

	return nil
}

func (b *PhysicalBackupConfig) Validate() error {
	if b.PostgresqlPhysical == nil {
		return errors.New("PostgresqlPhysical must be preloaded before Validate")
	}

	if err := b.FullBackupInterval.Validate(); err != nil {
		return fmt.Errorf("full backup interval: %w", err)
	}

	switch b.Encryption {
	case "", backups_core_enums.BackupEncryptionNone, backups_core_enums.BackupEncryptionEncrypted:
	default:
		return errors.New("encryption must be NONE or AES_256_GCM")
	}

	if b.WalLagThresholdBytes < 0 {
		return errors.New("wal lag threshold must be non-negative")
	}

	backupType := b.PostgresqlPhysical.BackupType

	if err := validateRetentionAllowedForBackupType(b.Retention, backupType); err != nil {
		return err
	}

	if err := b.validateRetentionFields(); err != nil {
		return err
	}

	switch backupType {
	case postgresql_physical.BackupTypeFullOnly:
		return b.validateFullOnly()

	case postgresql_physical.BackupTypeFullAndIncremental:
		return b.validateFullAndIncremental()

	case postgresql_physical.BackupTypeFullIncrementalAndWalStream:
		return b.validateFullIncrementalAndWalStream()

	default:
		return fmt.Errorf("unsupported backup type: %s", backupType)
	}
}

// The WAL stream supervisor runs a streamer exactly for these configs, and the
// WAL stream statistics call everything else "not streaming".
func (b *PhysicalBackupConfig) IsWalStreamingExpected() bool {
	if !b.IsBackupsEnabled || b.StorageID == nil {
		return false
	}

	return b.PostgresqlPhysical != nil && b.PostgresqlPhysical.BackupType.IsWalStreaming()
}

func (b *PhysicalBackupConfig) Copy(newDatabaseID uuid.UUID) *PhysicalBackupConfig {
	return &PhysicalBackupConfig{
		DatabaseID:                newDatabaseID,
		IsBackupsEnabled:          b.IsBackupsEnabled,
		FullBackupInterval:        b.FullBackupInterval.Copy(),
		IncrementalBackupInterval: b.IncrementalBackupInterval.Copy(),
		Retention:                 b.Retention,
		ChainsRetention:           b.ChainsRetention,
		FullBackupsRetention:      b.FullBackupsRetention,
		WalLagThresholdBytes:      b.WalLagThresholdBytes,
		StorageID:                 b.StorageID,
		Encryption:                b.Encryption,
		SendNotificationsOn:       b.SendNotificationsOn,
	}
}

// The database owns BackupType, so a switch there leaves the config in a shape
// Validate rejects, and an unsaveable config cannot be edited back into shape.
func (b *PhysicalBackupConfig) CoerceFieldsForBackupType(backupType postgresql_physical.BackupType) {
	switch backupType {
	case postgresql_physical.BackupTypeFullOnly:
		b.coerceFieldsForFullOnly()

	case postgresql_physical.BackupTypeFullAndIncremental:
		b.coerceFieldsForIncremental()
		b.WalLagThresholdBytes = 0

	case postgresql_physical.BackupTypeFullIncrementalAndWalStream:
		b.coerceFieldsForIncremental()

		if b.WalLagThresholdBytes <= 0 {
			b.WalLagThresholdBytes = defaultWalLagThresholdBytes
		}
	}
}

func (b *PhysicalBackupConfig) coerceFieldsForFullOnly() {
	b.IncrementalBackupInterval = intervals.Interval{}
	b.WalLagThresholdBytes = 0
	b.Retention = RetentionFullBackups
	b.ChainsRetention = ChainsRetention{}

	if b.FullBackupsRetention.Validate() != nil {
		b.FullBackupsRetention = FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  defaultFullBackupsRetentionCount,
		}
	}
}

func (b *PhysicalBackupConfig) coerceFieldsForIncremental() {
	if b.Retention != RetentionChains && b.Retention != RetentionChainsAndFullBackups {
		if b.FullBackupsRetention.Validate() == nil {
			b.Retention = RetentionChainsAndFullBackups
		} else {
			b.Retention = RetentionChains
		}
	}

	if b.Retention == RetentionChains {
		b.FullBackupsRetention = FullBackupsRetention{}
	}

	if b.ChainsRetention.Count <= 0 {
		b.ChainsRetention.Count = defaultChainsRetentionCount
	}

	if b.IncrementalBackupInterval.Validate() == nil &&
		isIncrementalStrictlyMoreFrequent(b.IncrementalBackupInterval, b.FullBackupInterval) {
		return
	}

	b.IncrementalBackupInterval = buildIntervalFasterThan(b.FullBackupInterval)
}

func buildIntervalFasterThan(fullBackupInterval intervals.Interval) intervals.Interval {
	switch fullBackupInterval.Type {
	case intervals.IntervalDaily:
		return intervals.Interval{Type: intervals.IntervalHourly}

	case intervals.IntervalWeekly, intervals.IntervalMonthly, intervals.IntervalCron:
		timeOfDay := defaultBackupTimeOfDay
		if fullBackupInterval.TimeOfDay != nil {
			timeOfDay = *fullBackupInterval.TimeOfDay
		}

		return intervals.Interval{Type: intervals.IntervalDaily, TimeOfDay: &timeOfDay}

	default:
		// No fixed interval is more frequent than HOURLY; the owner has to slow
		// the full cadence down first.
		return intervals.Interval{}
	}
}

func (b *PhysicalBackupConfig) validateRetentionFields() error {
	switch b.Retention {
	case RetentionChains:
		if b.ChainsRetention.Count <= 0 {
			return errors.New("CHAINS retention requires chains_retention.count > 0")
		}

		if !b.FullBackupsRetention.IsZero() {
			return errors.New("CHAINS retention must not set full_backups_retention.*")
		}

	case RetentionFullBackups:
		if !b.ChainsRetention.IsZero() {
			return errors.New("FULL_BACKUPS retention must not set chains_retention.*")
		}

		if err := b.FullBackupsRetention.Validate(); err != nil {
			return err
		}

	case RetentionChainsAndFullBackups:
		if b.ChainsRetention.Count <= 0 {
			return errors.New(
				"CHAINS_AND_FULL_BACKUPS retention requires chains_retention.count > 0",
			)
		}

		if err := b.FullBackupsRetention.Validate(); err != nil {
			return err
		}

	default:
		return fmt.Errorf("invalid retention: %q", b.Retention)
	}

	return nil
}

func (b *PhysicalBackupConfig) validateFullOnly() error {
	if !isIntervalZero(b.IncrementalBackupInterval) {
		return errors.New("incremental cadence cannot be set for FULL-only backups")
	}

	if b.WalLagThresholdBytes != 0 {
		return errors.New("wal lag threshold cannot be set when WAL streaming is disabled")
	}

	return nil
}

func (b *PhysicalBackupConfig) validateFullAndIncremental() error {
	if err := b.IncrementalBackupInterval.Validate(); err != nil {
		return fmt.Errorf("incremental backup interval: %w", err)
	}

	if !isIncrementalStrictlyMoreFrequent(b.IncrementalBackupInterval, b.FullBackupInterval) {
		return errors.New(
			"incremental cadence must be strictly more frequent than full cadence",
		)
	}

	if b.WalLagThresholdBytes != 0 {
		return errors.New("wal lag threshold cannot be set when WAL streaming is disabled")
	}

	return nil
}

func (b *PhysicalBackupConfig) validateFullIncrementalAndWalStream() error {
	if err := b.IncrementalBackupInterval.Validate(); err != nil {
		return fmt.Errorf("incremental backup interval: %w", err)
	}

	if !isIncrementalStrictlyMoreFrequent(b.IncrementalBackupInterval, b.FullBackupInterval) {
		return errors.New(
			"incremental cadence must be strictly more frequent than full cadence",
		)
	}

	if b.WalLagThresholdBytes <= 0 {
		return errors.New(
			"wal lag threshold must be greater than 0 for WAL streaming (defines slot-rebuild trigger)",
		)
	}

	return nil
}

func validateRetentionAllowedForBackupType(
	retention Retention,
	backupType postgresql_physical.BackupType,
) error {
	switch backupType {
	case postgresql_physical.BackupTypeFullOnly:
		if retention != RetentionFullBackups {
			return errors.New(
				"FULL_ONLY backups can only use FULL_BACKUPS retention (no chains exist)",
			)
		}

	case postgresql_physical.BackupTypeFullAndIncremental,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream:
		if retention != RetentionChains && retention != RetentionChainsAndFullBackups {
			return errors.New(
				"incremental backups must use CHAINS or CHAINS_AND_FULL_BACKUPS retention",
			)
		}
	}

	return nil
}

func isIntervalZero(i intervals.Interval) bool {
	return i.Type == "" &&
		i.TimeOfDay == nil &&
		i.Weekday == nil &&
		i.DayOfMonth == nil &&
		i.CronExpression == nil
}

func getFixedIntervalApproximatePeriod(t intervals.IntervalType) (time.Duration, bool) {
	switch t {
	case intervals.IntervalHourly:
		return time.Hour, true
	case intervals.IntervalDaily:
		return 24 * time.Hour, true
	case intervals.IntervalWeekly:
		return 7 * 24 * time.Hour, true
	case intervals.IntervalMonthly:
		return 30 * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func isIncrementalStrictlyMoreFrequent(incremental, full intervals.Interval) bool {
	if incremental.Type == intervals.IntervalCron || full.Type == intervals.IntervalCron {
		return true
	}

	incPeriod, incOk := getFixedIntervalApproximatePeriod(incremental.Type)
	fullPeriod, fullOk := getFixedIntervalApproximatePeriod(full.Type)

	if !incOk || !fullOk {
		return false
	}

	return incPeriod < fullPeriod
}
