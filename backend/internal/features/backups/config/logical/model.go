package backups_config_logical

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/util/period"
)

type LogicalBackupConfig struct {
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey;not null"`

	IsBackupsEnabled bool `json:"isBackupsEnabled" gorm:"column:is_backups_enabled;type:boolean;not null"`

	RetentionPolicyType RetentionPolicyType `json:"retentionPolicyType" gorm:"column:retention_policy_type;type:text;not null;default:'TIME_PERIOD'"`
	RetentionTimePeriod period.TimePeriod   `json:"retentionTimePeriod" gorm:"column:retention_time_period;type:text;not null;default:''"`

	RetentionCount     int `json:"retentionCount"     gorm:"column:retention_count;type:int;not null;default:0"`
	RetentionGfsHours  int `json:"retentionGfsHours"  gorm:"column:retention_gfs_hours;type:int;not null;default:0"`
	RetentionGfsDays   int `json:"retentionGfsDays"   gorm:"column:retention_gfs_days;type:int;not null;default:0"`
	RetentionGfsWeeks  int `json:"retentionGfsWeeks"  gorm:"column:retention_gfs_weeks;type:int;not null;default:0"`
	RetentionGfsMonths int `json:"retentionGfsMonths" gorm:"column:retention_gfs_months;type:int;not null;default:0"`
	RetentionGfsYears  int `json:"retentionGfsYears"  gorm:"column:retention_gfs_years;type:int;not null;default:0"`

	BackupInterval intervals.Interval `json:"backupInterval" gorm:"embedded"`

	Storage   *BackupConfigStorage `json:"storage"   gorm:"-"`
	StorageID *uuid.UUID           `json:"storageId" gorm:"column:storage_id;type:uuid;"`

	SendNotificationsOn       []BackupNotificationType `json:"sendNotificationsOn" gorm:"-"`
	SendNotificationsOnString string                   `json:"-"                   gorm:"column:send_notifications_on;type:text;not null"`

	IsRetryIfFailed     bool `json:"isRetryIfFailed"     gorm:"column:is_retry_if_failed;type:boolean;not null"`
	MaxFailedTriesCount int  `json:"maxFailedTriesCount" gorm:"column:max_failed_tries_count;type:int;not null"`

	Encryption backups_core_enums.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
}

func (h *LogicalBackupConfig) TableName() string {
	return "logical_backup_configs"
}

func (b *LogicalBackupConfig) BeforeSave(tx *gorm.DB) error {
	// Convert SendNotificationsOn array to string
	if len(b.SendNotificationsOn) > 0 {
		notificationTypes := make([]string, len(b.SendNotificationsOn))

		for i, notificationType := range b.SendNotificationsOn {
			notificationTypes[i] = string(notificationType)
		}

		b.SendNotificationsOnString = strings.Join(notificationTypes, ",")
	} else {
		b.SendNotificationsOnString = ""
	}

	return nil
}

func (b *LogicalBackupConfig) AfterFind(tx *gorm.DB) error {
	// Convert SendNotificationsOnString to array
	if b.SendNotificationsOnString != "" {
		notificationTypes := strings.Split(b.SendNotificationsOnString, ",")
		b.SendNotificationsOn = make([]BackupNotificationType, len(notificationTypes))

		for i, notificationType := range notificationTypes {
			b.SendNotificationsOn[i] = BackupNotificationType(notificationType)
		}
	} else {
		b.SendNotificationsOn = []BackupNotificationType{}
	}

	return nil
}

func (b *LogicalBackupConfig) Validate() error {
	if err := b.BackupInterval.Validate(); err != nil {
		return fmt.Errorf("backup interval: %w", err)
	}

	if err := b.validateRetentionPolicy(); err != nil {
		return err
	}

	if b.IsRetryIfFailed && b.MaxFailedTriesCount <= 0 {
		return errors.New("max failed tries count must be greater than 0")
	}

	if b.Encryption != "" && b.Encryption != backups_core_enums.BackupEncryptionNone &&
		b.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return errors.New("encryption must be NONE or ENCRYPTED")
	}

	return nil
}

func (b *LogicalBackupConfig) Copy(newDatabaseID uuid.UUID) *LogicalBackupConfig {
	return &LogicalBackupConfig{
		DatabaseID:          newDatabaseID,
		IsBackupsEnabled:    b.IsBackupsEnabled,
		RetentionPolicyType: b.RetentionPolicyType,
		RetentionTimePeriod: b.RetentionTimePeriod,
		RetentionCount:      b.RetentionCount,
		RetentionGfsHours:   b.RetentionGfsHours,
		RetentionGfsDays:    b.RetentionGfsDays,
		RetentionGfsWeeks:   b.RetentionGfsWeeks,
		RetentionGfsMonths:  b.RetentionGfsMonths,
		RetentionGfsYears:   b.RetentionGfsYears,
		BackupInterval:      b.BackupInterval.Copy(),
		StorageID:           b.StorageID,
		SendNotificationsOn: b.SendNotificationsOn,
		IsRetryIfFailed:     b.IsRetryIfFailed,
		MaxFailedTriesCount: b.MaxFailedTriesCount,
		Encryption:          b.Encryption,
	}
}

func (b *LogicalBackupConfig) validateRetentionPolicy() error {
	switch b.RetentionPolicyType {
	case RetentionPolicyTypeTimePeriod, "":
		if b.RetentionTimePeriod == "" {
			return errors.New("retention time period is required")
		}

	case RetentionPolicyTypeCount:
		if b.RetentionCount <= 0 {
			return errors.New("retention count must be greater than 0")
		}

	case RetentionPolicyTypeGFS:
		if b.RetentionGfsHours <= 0 && b.RetentionGfsDays <= 0 && b.RetentionGfsWeeks <= 0 &&
			b.RetentionGfsMonths <= 0 && b.RetentionGfsYears <= 0 {
			return errors.New("at least one GFS retention field must be greater than 0")
		}

	default:
		return errors.New("invalid retention policy type")
	}

	return nil
}
