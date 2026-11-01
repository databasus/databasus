// Package schedule decides when a recurring run is due: hourly, daily, weekly or monthly at a time of
// day, or by a cron expression.
package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Type string

const (
	TypeHourly  Type = "HOURLY"
	TypeDaily   Type = "DAILY"
	TypeWeekly  Type = "WEEKLY"
	TypeMonthly Type = "MONTHLY"
	TypeCron    Type = "CRON"
)

// Schedule names when a recurring run is due. TimeOfDay is "15:04" in UTC, Weekday is 0 to 7 with both 0
// and 7 meaning Sunday, and DayOfMonth is 1 to 31. In a month without that day, a monthly schedule runs on
// the month's last day. A cron expression must match a calendar date, so February 31 is refused.
type Schedule struct {
	Type           Type
	TimeOfDay      *string
	Weekday        *int
	DayOfMonth     *int
	CronExpression *string
}

const timeOfDayLayout = "15:04"

func (s Schedule) Validate() error {
	switch s.Type {
	case TypeHourly, TypeDaily, TypeWeekly, TypeMonthly, TypeCron:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidType, s.Type)
	}

	if s.Type == TypeDaily || s.Type == TypeWeekly || s.Type == TypeMonthly {
		if s.TimeOfDay == nil {
			return ErrTimeOfDayRequired
		}

		if _, err := time.Parse(timeOfDayLayout, *s.TimeOfDay); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidTimeOfDay, err)
		}
	}

	if s.Type == TypeWeekly {
		if s.Weekday == nil {
			return ErrWeekdayRequired
		}

		if *s.Weekday < 0 || *s.Weekday > 7 {
			return ErrWeekdayOutOfRange
		}
	}

	if s.Type == TypeMonthly {
		if s.DayOfMonth == nil {
			return ErrDayOfMonthRequired
		}

		if *s.DayOfMonth < 1 || *s.DayOfMonth > 31 {
			return ErrDayOfMonthOutOfRange
		}
	}

	if s.Type == TypeCron {
		if s.CronExpression == nil || *s.CronExpression == "" {
			return ErrCronExpressionMissing
		}

		if _, err := parseCron(*s.CronExpression); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCronExpression, err)
		}

		if s.isCronWithoutRunDate() {
			return fmt.Errorf("%w: it has no date to run on", ErrInvalidCronExpression)
		}
	}

	return nil
}

// IsDue reports whether a run is due at now after the last run at lastRun. A schedule that never ran is
// due at once, unless its cron expression has no date to run on, which is never due. A slot that passed
// without a run is caught up, but a weekly or monthly slot only within its own week or month.
func (s Schedule) IsDue(now time.Time, lastRun *time.Time) bool {
	if s.isCronWithoutRunDate() {
		return false
	}

	if lastRun == nil {
		return true
	}

	switch s.Type {
	case TypeHourly:
		return now.Sub(*lastRun) >= time.Hour
	case TypeDaily:
		return s.isDailyDue(now, *lastRun)
	case TypeWeekly:
		return s.isWeeklyDue(now, *lastRun)
	case TypeMonthly:
		return s.isMonthlyDue(now, *lastRun)
	case TypeCron:
		return s.isCronDue(now, *lastRun)
	default:
		return false
	}
}

// GetApproxPeriod is a coarse cadence for sizing a wait as a fraction of it, where the exact next run
// does not matter. A cron or unknown schedule returns 0, so the caller falls back to its own cap instead
// of guessing a period from an arbitrary expression.
func (s Schedule) GetApproxPeriod() time.Duration {
	switch s.Type {
	case TypeHourly:
		return time.Hour
	case TypeDaily:
		return 24 * time.Hour
	case TypeWeekly:
		return 7 * 24 * time.Hour
	case TypeMonthly:
		return 30 * 24 * time.Hour
	case TypeCron:
		return 0
	default:
		return 0
	}
}

func (s Schedule) isDailyDue(now, lastRun time.Time) bool {
	if s.TimeOfDay == nil {
		return !isSameDay(lastRun, now)
	}

	timeOfDay, err := time.Parse(timeOfDayLayout, *s.TimeOfDay)
	if err != nil {
		return false
	}

	todaySlot := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		timeOfDay.Hour(),
		timeOfDay.Minute(),
		0,
		0,
		now.Location(),
	)

	lastSlot := todaySlot
	if now.Before(todaySlot) {
		lastSlot = todaySlot.AddDate(0, 0, -1)
	}

	return !now.Before(lastSlot) && lastRun.Before(lastSlot)
}

func (s Schedule) isWeeklyDue(now, lastRun time.Time) bool {
	if s.Weekday == nil {
		return now.Sub(lastRun) >= 7*24*time.Hour
	}

	daysFromMonday := int(time.Weekday(*s.Weekday)) - 1
	if time.Weekday(*s.Weekday%7) == time.Sunday {
		daysFromMonday = 6
	}

	thisWeekSlot := s.atTimeOfDay(getStartOfWeek(now).AddDate(0, 0, daysFromMonday))
	if now.Before(thisWeekSlot) {
		return false
	}

	return lastRun.Before(thisWeekSlot)
}

func (s Schedule) isMonthlyDue(now, lastRun time.Time) bool {
	if s.DayOfMonth == nil {
		return lastRun.Before(getStartOfMonth(now))
	}

	slotDay := min(*s.DayOfMonth, getDaysInMonth(now))

	thisMonthSlot := s.atTimeOfDay(time.Date(now.Year(), now.Month(), slotDay, 0, 0, 0, 0, now.Location()))
	if now.Before(thisMonthSlot) {
		return false
	}

	return lastRun.Before(thisMonthSlot)
}

func (s Schedule) isCronDue(now, lastRun time.Time) bool {
	if s.CronExpression == nil || *s.CronExpression == "" {
		return false
	}

	cronSchedule, err := parseCron(*s.CronExpression)
	if err != nil {
		return false
	}

	return !now.Before(cronSchedule.Next(lastRun))
}

// isCronWithoutRunDate looks from a fixed leap year, because whether a day exists in a month does not depend
// on the moment and February 29 falls inside the parser's five-year search from there.
func (s Schedule) isCronWithoutRunDate() bool {
	if s.Type != TypeCron || s.CronExpression == nil {
		return false
	}

	cronSchedule, err := parseCron(*s.CronExpression)
	if err != nil {
		return false
	}

	searchStart := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	return cronSchedule.Next(searchStart).IsZero()
}

func (s Schedule) atTimeOfDay(day time.Time) time.Time {
	if s.TimeOfDay == nil {
		return day
	}

	timeOfDay, err := time.Parse(timeOfDayLayout, *s.TimeOfDay)
	if err != nil {
		return day
	}

	return time.Date(day.Year(), day.Month(), day.Day(), timeOfDay.Hour(), timeOfDay.Minute(), 0, 0, day.Location())
}

func parseCron(expression string) (cron.Schedule, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	return parser.Parse(expression)
}

func isSameDay(first, second time.Time) bool {
	firstYear, firstMonth, firstDay := first.Date()
	secondYear, secondMonth, secondDay := second.Date()

	return firstYear == secondYear && firstMonth == secondMonth && firstDay == secondDay
}

func getStartOfWeek(moment time.Time) time.Time {
	daysSinceSunday := int(moment.Weekday())
	if daysSinceSunday == 0 {
		daysSinceSunday = 7
	}

	return time.Date(moment.Year(), moment.Month(), moment.Day()-daysSinceSunday+1, 0, 0, 0, 0, moment.Location())
}

func getStartOfMonth(moment time.Time) time.Time {
	return time.Date(moment.Year(), moment.Month(), 1, 0, 0, 0, 0, moment.Location())
}

func getDaysInMonth(moment time.Time) int {
	return getStartOfMonth(moment).AddDate(0, 1, -1).Day()
}
