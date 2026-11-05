package configs

import (
	"github.com/google/uuid"

	"databasus-backend/internal/lib/schedule"
)

type Config struct {
	DatabaseID                     uuid.UUID
	IsAutomaticVerificationEnabled bool
	ScheduleType                   VerificationScheduleType
	Schedule                       schedule.Schedule
	NotificationTriggers           []VerificationNotificationType
}

func GetDefault(databaseID uuid.UUID) Config {
	return Config{
		DatabaseID:                     databaseID,
		IsAutomaticVerificationEnabled: false,
		ScheduleType:                   VerificationScheduleAfterBackup,
		Schedule: schedule.Schedule{
			Type:           schedule.TypeWeekly,
			TimeOfDay:      new("04:00"),
			Weekday:        new(0),
			DayOfMonth:     nil,
			CronExpression: nil,
		},
		NotificationTriggers: []VerificationNotificationType{NotificationVerificationFailed},
	}
}

type VerificationNotificationType string

const (
	NotificationVerificationSuccess VerificationNotificationType = "VERIFICATION_SUCCESS"
	NotificationVerificationFailed  VerificationNotificationType = "VERIFICATION_FAILED"
)

type VerificationScheduleType string

const (
	VerificationScheduleInterval    VerificationScheduleType = "INTERVAL"
	VerificationScheduleAfterBackup VerificationScheduleType = "AFTER_BACKUP"
)
