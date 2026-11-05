package schedule

import "errors"

var (
	ErrInvalidType           = errors.New("invalid schedule type")
	ErrTimeOfDayRequired     = errors.New("time of day is required for daily, weekly and monthly schedules")
	ErrInvalidTimeOfDay      = errors.New("invalid time of day")
	ErrWeekdayRequired       = errors.New("weekday is required for weekly schedules")
	ErrWeekdayOutOfRange     = errors.New("weekday must be between 0 and 7")
	ErrDayOfMonthRequired    = errors.New("day of month is required for monthly schedules")
	ErrDayOfMonthOutOfRange  = errors.New("day of month must be between 1 and 31")
	ErrCronExpressionMissing = errors.New("cron expression is required for cron schedules")
	ErrInvalidCronExpression = errors.New("invalid cron expression")
)
