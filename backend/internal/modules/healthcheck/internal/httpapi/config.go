package httpapi

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/healthcheck/internal/configs"
)

func fromConfigView(view configs.ConfigView) HealthcheckConfig {
	config := view.Config

	return HealthcheckConfig{
		DatabaseId:                      config.DatabaseID,
		IsHealthcheckEnabled:            config.IsHealthcheckEnabled,
		IsNotificationEnabled:           config.IsNotificationEnabled,
		IntervalMinutes:                 config.IntervalMinutes,
		FailedAttemptsBeforeUnavailable: config.FailedAttemptsBeforeUnavailable,
		AttemptRetentionDays:            config.AttemptRetentionDays,
		ShouldConfirmEnabling:           view.ShouldConfirmEnabling,
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
