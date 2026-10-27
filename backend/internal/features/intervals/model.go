package intervals

import (
	"time"

	"databasus-backend/internal/lib/schedule"
)

// Interval is the schedule of the old verification config, stored in its columns. Its rules live in
// lib/schedule until the verification config moves into its module.
type Interval struct {
	Type IntervalType `json:"type" gorm:"column:interval_type;type:text;not null"`

	TimeOfDay      *string `json:"timeOfDay"                gorm:"column:time_of_day;type:text"`
	Weekday        *int    `json:"weekday,omitempty"        gorm:"column:weekday;type:int"`
	DayOfMonth     *int    `json:"dayOfMonth,omitempty"     gorm:"column:day_of_month;type:int"`
	CronExpression *string `json:"cronExpression,omitempty" gorm:"column:cron_expression;type:text"`
}

func (i *Interval) ToSchedule() schedule.Schedule {
	return schedule.Schedule{
		Type:           schedule.Type(i.Type),
		TimeOfDay:      i.TimeOfDay,
		Weekday:        i.Weekday,
		DayOfMonth:     i.DayOfMonth,
		CronExpression: i.CronExpression,
	}
}

func (i *Interval) Validate() error {
	return i.ToSchedule().Validate()
}

func (i *Interval) ShouldTriggerBackup(now time.Time, lastBackupTime *time.Time) bool {
	return i.ToSchedule().IsDue(now, lastBackupTime)
}

func (i *Interval) ApproxPeriod() time.Duration {
	return i.ToSchedule().GetApproxPeriod()
}

func (i *Interval) Copy() Interval {
	return Interval{
		Type:           i.Type,
		TimeOfDay:      i.TimeOfDay,
		Weekday:        i.Weekday,
		DayOfMonth:     i.DayOfMonth,
		CronExpression: i.CronExpression,
	}
}
