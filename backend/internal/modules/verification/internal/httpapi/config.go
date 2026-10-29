package httpapi

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/verification/internal/configs"
)

func fromConfig(config configs.Config) VerificationConfig {
	triggers := make([]VerificationNotificationTrigger, len(config.NotificationTriggers))
	for index, trigger := range config.NotificationTriggers {
		triggers[index] = VerificationNotificationTrigger(trigger)
	}
	return VerificationConfig{
		DatabaseId:                     config.DatabaseID,
		IsAutomaticVerificationEnabled: config.IsAutomaticVerificationEnabled,
		ScheduleType:                   VerificationScheduleType(config.ScheduleType),
		Schedule:                       fromSchedule(config.Schedule),
		NotificationTriggers:           triggers,
	}
}

func toConfig(id uuid.UUID, body VerificationConfigInput) configs.Config {
	triggers := make([]configs.VerificationNotificationType, len(body.NotificationTriggers))
	for index, trigger := range body.NotificationTriggers {
		triggers[index] = configs.VerificationNotificationType(trigger)
	}
	return configs.Config{
		DatabaseID:                     id,
		IsAutomaticVerificationEnabled: body.IsAutomaticVerificationEnabled,
		ScheduleType:                   configs.VerificationScheduleType(body.ScheduleType),
		Schedule:                       toSchedule(body.Schedule),
		NotificationTriggers:           triggers,
	}
}
