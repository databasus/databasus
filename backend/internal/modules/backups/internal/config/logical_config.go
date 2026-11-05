package config

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/schedule"
)

type LogicalRetentionPolicy string

const (
	LogicalRetentionPolicyTimePeriod LogicalRetentionPolicy = "TIME_PERIOD"
	LogicalRetentionPolicyCount      LogicalRetentionPolicy = "COUNT"
	LogicalRetentionPolicyGfs        LogicalRetentionPolicy = "GFS"
)

// RetentionPeriod is how long the TIME_PERIOD retention keeps a backup.
type RetentionPeriod string

const (
	RetentionPeriodDay     RetentionPeriod = "DAY"
	RetentionPeriodWeek    RetentionPeriod = "WEEK"
	RetentionPeriodMonth   RetentionPeriod = "MONTH"
	RetentionPeriod3Months RetentionPeriod = "3_MONTH"
	RetentionPeriod6Months RetentionPeriod = "6_MONTH"
	RetentionPeriodYear    RetentionPeriod = "YEAR"
	RetentionPeriod2Years  RetentionPeriod = "2_YEARS"
	RetentionPeriod3Years  RetentionPeriod = "3_YEARS"
	RetentionPeriod4Years  RetentionPeriod = "4_YEARS"
	RetentionPeriod5Years  RetentionPeriod = "5_YEARS"
	RetentionPeriodForever RetentionPeriod = "FOREVER"
)

const (
	daysInMonth = 30
	daysInYear  = 365
)

// GetDuration is 0 for FOREVER and for an unknown period, which keep every backup.
func (p RetentionPeriod) GetDuration() time.Duration {
	day := 24 * time.Hour

	switch p {
	case RetentionPeriodDay:
		return day
	case RetentionPeriodWeek:
		return 7 * day
	case RetentionPeriodMonth:
		return daysInMonth * day
	case RetentionPeriod3Months:
		return 3 * daysInMonth * day
	case RetentionPeriod6Months:
		return 6 * daysInMonth * day
	case RetentionPeriodYear:
		return daysInYear * day
	case RetentionPeriod2Years:
		return 2 * daysInYear * day
	case RetentionPeriod3Years:
		return 3 * daysInYear * day
	case RetentionPeriod4Years:
		return 4 * daysInYear * day
	case RetentionPeriod5Years:
		return 5 * daysInYear * day
	case RetentionPeriodForever:
		return 0
	default:
		return 0
	}
}

func (p RetentionPeriod) isValid() bool {
	switch p {
	case RetentionPeriodDay, RetentionPeriodWeek, RetentionPeriodMonth, RetentionPeriod3Months,
		RetentionPeriod6Months, RetentionPeriodYear, RetentionPeriod2Years, RetentionPeriod3Years,
		RetentionPeriod4Years, RetentionPeriod5Years, RetentionPeriodForever:
		return true
	default:
		return false
	}
}

// LogicalRetention keeps backups for Period, keeps the last Count backups, or keeps them by Gfs,
// whichever Policy names. The fields of the other policies are kept as they were saved.
type LogicalRetention struct {
	Policy LogicalRetentionPolicy
	Period RetentionPeriod
	Count  int
	Gfs    GfsRetention
}

type LogicalBackupConfig struct {
	BackupConfig
	Schedule        schedule.Schedule
	Retention       LogicalRetention
	IsRetryIfFailed bool
	// MaxFailedTriesCount bounds the retries of a failed backup while IsRetryIfFailed is on.
	MaxFailedTriesCount int
}

func (c LogicalBackupConfig) GetBackupConfig() BackupConfig {
	return c.BackupConfig
}

func (c LogicalBackupConfig) GetMethod() Method {
	return MethodLogical
}

func getLogicalNotificationTriggers() []NotificationTrigger {
	return []NotificationTrigger{
		NotificationTriggerBackupFailed,
		NotificationTriggerBackupSuccess,
	}
}

func (c LogicalBackupConfig) validate() error {
	err := validateSchedule("schedule", c.Schedule)
	if err != nil {
		return err
	}

	err = c.validateRetention()
	if err != nil {
		return err
	}

	if c.IsRetryIfFailed && c.MaxFailedTriesCount <= 0 {
		return newInvalidSettingError("maxFailedTriesCount", "must be greater than 0 when failed backups are retried")
	}

	return validateCommonFields(c.BackupConfig, getLogicalNotificationTriggers())
}

func (c LogicalBackupConfig) validateRetention() error {
	if !c.Retention.Period.isValid() {
		return newInvalidSettingError("retention.period", "is not a retention period")
	}

	switch c.Retention.Policy {
	case LogicalRetentionPolicyTimePeriod:
		return nil
	case LogicalRetentionPolicyCount:
		if c.Retention.Count <= 0 {
			return newInvalidSettingError("retention.count", "must be greater than 0")
		}

		return nil
	case LogicalRetentionPolicyGfs:
		if !c.Retention.Gfs.HasAnyPeriod() {
			return newInvalidSettingError("retention", "must keep at least one GFS period")
		}

		return nil
	default:
		return newInvalidSettingError("retention.policy", "is not a retention policy")
	}
}

// copyFor gives a copied database the same config. The copy is in the original's workspace, so the
// storage stays valid for it.
func (c LogicalBackupConfig) copyFor(copiedDatabaseID uuid.UUID) LogicalBackupConfig {
	copiedConfig := c
	copiedConfig.DatabaseID = copiedDatabaseID
	copiedConfig.NotificationTriggers = slices.Clone(c.NotificationTriggers)

	return copiedConfig
}

func validateSchedule(field string, validatedSchedule schedule.Schedule) error {
	err := validatedSchedule.Validate()

	switch {
	case err == nil:
		return nil
	case errors.Is(err, schedule.ErrTimeOfDayRequired), errors.Is(err, schedule.ErrInvalidTimeOfDay):
		return newInvalidSettingError(field+".timeOfDay", err.Error())
	case errors.Is(err, schedule.ErrWeekdayRequired), errors.Is(err, schedule.ErrWeekdayOutOfRange):
		return newInvalidSettingError(field+".weekday", err.Error())
	case errors.Is(err, schedule.ErrDayOfMonthRequired), errors.Is(err, schedule.ErrDayOfMonthOutOfRange):
		return newInvalidSettingError(field+".dayOfMonth", err.Error())
	case errors.Is(err, schedule.ErrCronExpressionMissing), errors.Is(err, schedule.ErrInvalidCronExpression):
		return newInvalidSettingError(field+".cronExpression", err.Error())
	default:
		return newInvalidSettingError(field+".type", err.Error())
	}
}
