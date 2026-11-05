package httpapi

import "databasus-backend/internal/lib/schedule"

func toSchedule(body Schedule) schedule.Schedule {
	return schedule.Schedule{
		Type:           schedule.Type(body.Type),
		TimeOfDay:      body.TimeOfDay,
		Weekday:        body.Weekday,
		DayOfMonth:     body.DayOfMonth,
		CronExpression: body.CronExpression,
	}
}

func fromSchedule(savedSchedule schedule.Schedule) Schedule {
	return Schedule{
		Type:           ScheduleType(savedSchedule.Type),
		TimeOfDay:      savedSchedule.TimeOfDay,
		Weekday:        savedSchedule.Weekday,
		DayOfMonth:     savedSchedule.DayOfMonth,
		CronExpression: savedSchedule.CronExpression,
	}
}
