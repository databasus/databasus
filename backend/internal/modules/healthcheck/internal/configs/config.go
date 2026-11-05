package configs

import "github.com/google/uuid"

type Config struct {
	DatabaseID                      uuid.UUID
	IsHealthcheckEnabled            bool
	IsNotificationEnabled           bool
	IntervalMinutes                 int
	FailedAttemptsBeforeUnavailable int
	AttemptRetentionDays            int
}

func (Config) TableName() string { return "healthcheck_configs" }

func GetDefaultConfig(databaseID uuid.UUID) Config {
	return Config{
		DatabaseID:                      databaseID,
		IsHealthcheckEnabled:            true,
		IsNotificationEnabled:           true,
		IntervalMinutes:                 1,
		FailedAttemptsBeforeUnavailable: 3,
		AttemptRetentionDays:            7,
	}
}
