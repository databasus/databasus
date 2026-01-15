package intervals

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Interval struct {
	ID       uuid.UUID    `json:"id"       gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Interval IntervalType `json:"interval" gorm:"type:text;not null"`

	TimeOfDay *string `json:"timeOfDay" gorm:"type:text;"`
	// only for WEEKLY
	Weekday *int `json:"weekday,omitempty" gorm:"type:int"`
	// only for MONTHLY
	DayOfMonth *int `json:"dayOfMonth,omitempty" gorm:"type:int"`
	// only for CRON
	CronExpression *string `json:"cronExpression,omitempty" gorm:"type:text"`
}

func (i *Interval) BeforeSave(tx *gorm.DB) error {
	return i.Validate()
}

func (i *Interval) Validate() error {
	// for daily, weekly and monthly intervals time of day is required
	if (i.Interval == IntervalDaily || i.Interval == IntervalWeekly || i.Interval == IntervalMonthly) &&
		i.TimeOfDay == nil {
		return errors.New("time of day is required for daily, weekly and monthly intervals")
	}

	// for weekly interval weekday is required
	if i.Interval == IntervalWeekly && i.Weekday == nil {
		return errors.New("weekday is required for weekly intervals")
	}

	// for monthly interval day of month is required
	if i.Interval == IntervalMonthly && i.DayOfMonth == nil {
		return errors.New("day of month is required for monthly intervals")
	}

	// for cron interval cron expression is required and must be valid
	if i.Interval == IntervalCron {
		if i.CronExpression == nil || *i.CronExpression == "" {
			return errors.New("cron expression is required for cron intervals")
		}
		if err := i.validateCronExpression(*i.CronExpression); err != nil {
			return err
		}
	}

	return nil
}

// ShouldTriggerBackup checks if a backup should be triggered based on the interval and last backup time
func (i *Interval) ShouldTriggerBackup(now time.Time, lastBackupTime *time.Time) bool {
	// If no backup has been made yet, trigger immediately
	if lastBackupTime == nil {
		return true
	}

	switch i.Interval {
	case IntervalHourly:
		return now.Sub(*lastBackupTime) >= time.Hour
	case IntervalDaily:
		return i.shouldTriggerDaily(now, *lastBackupTime)
	case IntervalWeekly:
		return i.shouldTriggerWeekly(now, *lastBackupTime)
	case IntervalMonthly:
		return i.shouldTriggerMonthly(now, *lastBackupTime)
	case IntervalCron:
		return i.shouldTriggerCron(now, *lastBackupTime)
	default:
		return false
	}
}

func (i *Interval) Copy() *Interval {
	return &Interval{
		ID:             uuid.Nil,
		Interval:       i.Interval,
		TimeOfDay:      i.TimeOfDay,
		Weekday:        i.Weekday,
		DayOfMonth:     i.DayOfMonth,
		CronExpression: i.CronExpression,
	}
}

// daily trigger: honour the TimeOfDay slot and catch up the previous one
func (i *Interval) shouldTriggerDaily(now, lastBackup time.Time) bool {
	if i.TimeOfDay == nil {
		return !isSameDay(lastBackup, now)
	}

	t, err := time.Parse("15:04", *i.TimeOfDay)
	if err != nil {
		return false // malformed ⇒ play safe
	}

	// Today's scheduled slot (todayTgt)
	todayTgt := time.Date(
		now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), 0, 0, now.Location(),
	)

	// The last scheduled slot that should already have happened
	var lastScheduled time.Time
	if now.Before(todayTgt) {
		lastScheduled = todayTgt.AddDate(0, 0, -1)
	} else {
		lastScheduled = todayTgt
	}

	// Fire when we are past that slot AND no backup has been taken since it
	return (now.After(lastScheduled) || now.Equal(lastScheduled)) &&
		lastBackup.Before(lastScheduled)
}

// weekly trigger: on specified weekday/calendar week, otherwise ≥7 days
func (i *Interval) shouldTriggerWeekly(now, lastBackup time.Time) bool {
	if i.Weekday != nil {
		targetWd := time.Weekday(*i.Weekday)

		// Calculate the target datetime for this week
		startOfWeek := getStartOfWeek(now)

		// Convert Go weekday to days from Monday: Sunday=6, Monday=0, Tuesday=1, ..., Saturday=5
		var daysFromMonday int
		if targetWd == time.Sunday {
			daysFromMonday = 6
		} else {
			daysFromMonday = int(targetWd) - 1
		}

		targetThisWeek := startOfWeek.AddDate(0, 0, daysFromMonday)

		if i.TimeOfDay != nil {
			t, err := time.Parse("15:04", *i.TimeOfDay)
			if err == nil {
				targetThisWeek = time.Date(
					targetThisWeek.Year(),
					targetThisWeek.Month(),
					targetThisWeek.Day(),
					t.Hour(),
					t.Minute(),
					0,
					0,
					targetThisWeek.Location(),
				)
			}
		}

		// If current time is at or after the target time this week
		// and no backup has been made at or after the target time, trigger
		if now.After(targetThisWeek) || now.Equal(targetThisWeek) {
			return lastBackup.Before(targetThisWeek)
		}

		return false
	}

	// no Weekday: generic 7-day interval
	return now.Sub(lastBackup) >= 7*24*time.Hour
}

// monthly trigger: on specified day/calendar month, otherwise next calendar month
func (i *Interval) shouldTriggerMonthly(now, lastBackup time.Time) bool {
	if i.DayOfMonth != nil {
		day := *i.DayOfMonth

		// Calculate the target datetime for this month
		targetThisMonth := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, now.Location())

		if i.TimeOfDay != nil {
			t, err := time.Parse("15:04", *i.TimeOfDay)
			if err == nil {
				targetThisMonth = time.Date(
					targetThisMonth.Year(),
					targetThisMonth.Month(),
					targetThisMonth.Day(),
					t.Hour(),
					t.Minute(),
					0,
					0,
					targetThisMonth.Location(),
				)
			}
		}

		// If current time is at or after the target time this month
		// and no backup has been made at or after the target time, trigger
		if now.After(targetThisMonth) || now.Equal(targetThisMonth) {
			return lastBackup.Before(targetThisMonth)
		}

		return false
	}
	// no DayOfMonth: if we're in a new calendar month
	return lastBackup.Before(getStartOfMonth(now))
}

func isSameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func getStartOfWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, t.Location())
}

func getStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// cron trigger: check if we've passed a scheduled cron time since last backup
func (i *Interval) shouldTriggerCron(now, lastBackup time.Time) bool {
	if i.CronExpression == nil || *i.CronExpression == "" {
		return false
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*i.CronExpression)
	if err != nil {
		return false
	}

	// Find the next scheduled time after the last backup
	nextAfterLastBackup := schedule.Next(lastBackup)

	// If we're at or past that next scheduled time, trigger
	return now.After(nextAfterLastBackup) || now.Equal(nextAfterLastBackup)
}

func (i *Interval) validateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	if err != nil {
		return errors.New("invalid cron expression: " + err.Error())
	}
	return nil
}
