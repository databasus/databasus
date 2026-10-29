package httpapi

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/healthcheck/internal/configs"
)

func fromConfig(config configs.Config) HealthcheckConfig {
	return HealthcheckConfig{
		DatabaseId:                      config.DatabaseID,
		IsHealthcheckEnabled:            config.IsHealthcheckEnabled,
		IsNotificationEnabled:           config.IsNotificationEnabled,
		IntervalMinutes:                 config.IntervalMinutes,
		FailedAttemptsBeforeUnavailable: config.FailedAttemptsBeforeUnavailable,
		AttemptRetentionDays:            config.AttemptRetentionDays,
	}
}

func toConfig(id uuid.UUID, body HealthcheckConfigInput) configs.Config {
	return configs.Config{
		DatabaseID:                      id,
		IsHealthcheckEnabled:            body.IsHealthcheckEnabled,
		IsNotificationEnabled:           body.IsNotificationEnabled,
		IntervalMinutes:                 body.IntervalMinutes,
		FailedAttemptsBeforeUnavailable: body.FailedAttemptsBeforeUnavailable,
		AttemptRetentionDays:            body.AttemptRetentionDays,
	}
}
