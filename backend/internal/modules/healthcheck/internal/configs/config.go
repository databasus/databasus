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

// GetDefaultConfig switches the checks off on Neon: its compute sleeps while idle and is billed only
// while awake, and a check every minute would keep it awake around the clock.
func GetDefaultConfig(databaseID uuid.UUID, isNeonHost bool) Config {
	return Config{
		DatabaseID:                      databaseID,
		IsHealthcheckEnabled:            !isNeonHost,
		IsNotificationEnabled:           true,
		IntervalMinutes:                 1,
		FailedAttemptsBeforeUnavailable: 3,
		AttemptRetentionDays:            7,
	}
}

// ConfigView is a config as its editor reads it: the stored values and whether switching the checks
// on asks the person for a confirmation, which follows the database's current host and is not stored.
type ConfigView struct {
	Config                Config
	ShouldConfirmEnabling bool
}
