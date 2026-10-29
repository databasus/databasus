package verification_config

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"databasus-backend/internal/features/intervals"
)

type BackupVerificationConfig struct {
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey;not null"`

	IsScheduledVerificationEnabled bool `json:"isScheduledVerificationEnabled" gorm:"column:is_scheduled_verification_enabled;type:boolean;not null;default:false"`

	ScheduleType         VerificationScheduleType `json:"scheduleType"         gorm:"column:schedule_type;type:text;not null;default:'AFTER_BACKUP'"`
	VerificationInterval intervals.Interval       `json:"verificationInterval" gorm:"embedded"`

	SendNotificationsOn pq.StringArray `json:"sendNotificationsOn" gorm:"column:send_notifications_on;type:text[];not null;default:'{}'"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (BackupVerificationConfig) TableName() string {
	return "verification_configs"
}

func (c *BackupVerificationConfig) Validate() error {
	if c.ScheduleType != VerificationScheduleInterval && c.ScheduleType != VerificationScheduleAfterBackup {
		return fmt.Errorf("invalid verification schedule type: %q", c.ScheduleType)
	}

	if c.IsScheduledVerificationEnabled && c.ScheduleType == VerificationScheduleInterval {
		if err := c.VerificationInterval.Validate(); err != nil {
			return fmt.Errorf("verification interval: %w", err)
		}
	}

	for _, notification := range c.SendNotificationsOn {
		switch VerificationNotificationType(notification) {
		case NotificationVerificationSuccess,
			NotificationVerificationFailed:
		default:
			return fmt.Errorf("invalid verification notification type: %q", notification)
		}
	}

	return nil
}

func (c *BackupVerificationConfig) Copy(newDatabaseID uuid.UUID) *BackupVerificationConfig {
	return &BackupVerificationConfig{
		DatabaseID:                     newDatabaseID,
		IsScheduledVerificationEnabled: c.IsScheduledVerificationEnabled,
		ScheduleType:                   c.ScheduleType,
		VerificationInterval:           c.VerificationInterval.Copy(),
		SendNotificationsOn:            slices.Clone(c.SendNotificationsOn),
	}
}
