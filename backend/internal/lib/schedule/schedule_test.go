package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDueForHourlySchedule(t *testing.T) {
	t.Parallel()

	testedSchedule := &Schedule{
		Type: TypeHourly,
	}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("is due without a previous run", func(t *testing.T) {
		t.Parallel()

		isDue := testedSchedule.IsDue(baseTime, nil)
		assert.True(t, isDue)
	})

	t.Run("is not due when the last run was 59 minutes ago", func(t *testing.T) {
		t.Parallel()

		lastRun := baseTime.Add(-59 * time.Minute)
		isDue := testedSchedule.IsDue(baseTime, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is due when the last run was exactly 1 hour ago", func(t *testing.T) {
		t.Parallel()

		lastRun := baseTime.Add(-1 * time.Hour)
		isDue := testedSchedule.IsDue(baseTime, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is due when the last run was 2 hours ago", func(t *testing.T) {
		t.Parallel()

		lastRun := baseTime.Add(-2 * time.Hour)
		isDue := testedSchedule.IsDue(baseTime, &lastRun)
		assert.True(t, isDue)
	})
}

func TestIsDueForDailySchedule(t *testing.T) {
	t.Parallel()

	timeOfDay := "09:00"
	testedSchedule := &Schedule{
		Type:      TypeDaily,
		TimeOfDay: &timeOfDay,
	}

	// Base time: January 15, 2024
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("is due without a previous run", func(t *testing.T) {
		t.Parallel()

		now := baseDate.Add(10 * time.Hour) // 10:00 AM
		isDue := testedSchedule.IsDue(now, nil)
		assert.True(t, isDue)
	})

	t.Run("is not due at 08:59 when no run happened today", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 8, 59, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is due at exactly 09:00 when no run happened today", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is due at 09:01 when no run happened today", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 9, 1, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is not due when a run happened at 09:00 today", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)    // 3 PM
		lastRun := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Today at 9 AM
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run(
		"is due at 09:00 when the last run was yesterday at 09:00",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
			lastRun := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday at 9 AM
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due at 09:00 when the last run was yesterday at 15:00",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
			lastRun := time.Date(2024, 1, 14, 15, 0, 0, 0, time.UTC) // Yesterday at 15:00
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due at the slot after an earlier manual run",
		func(t *testing.T) {
			t.Parallel()

			timeOfDay := "21:00"
			testedSchedule := &Schedule{
				Type:      TypeDaily,
				TimeOfDay: &timeOfDay,
			}

			manual := time.Date(2025, 6, 6, 16, 17, 0, 0, time.UTC)   // manual earlier
			scheduled := time.Date(2025, 6, 6, 21, 0, 0, 0, time.UTC) // scheduled time

			isDue := testedSchedule.IsDue(scheduled, &manual)
			assert.True(t, isDue, "scheduled run should trigger even after earlier manual backup")
		},
	)

	t.Run("is due the next day when the previous slot was missed", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "21:00"
		testedSchedule := &Schedule{
			Type:      TypeDaily,
			TimeOfDay: &timeOfDay,
		}

		// It's June-07 15:00 UTC, yesterday's scheduled backup was missed
		now := time.Date(2025, 6, 7, 15, 0, 0, 0, time.UTC)
		lastRun := time.Date(2025, 6, 6, 16, 0, 0, 0, time.UTC) // before yesterday's 21:00

		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue, "should catch up missed 21:00 backup the next day at 15:00")
	})
}

func TestIsDueForWeeklySchedule(t *testing.T) {
	t.Parallel()

	timeOfDay := "15:00"
	weekday := 3 // Wednesday (0=Sunday, 1=Monday, ..., 3=Wednesday)
	testedSchedule := &Schedule{
		Type:      TypeWeekly,
		TimeOfDay: &timeOfDay,
		Weekday:   &weekday,
	}

	// Base time: Wednesday, January 17, 2024 (to ensure we're on Wednesday)
	wednesday := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	t.Run("is due without a previous run", func(t *testing.T) {
		t.Parallel()

		now := wednesday.Add(16 * time.Hour) // 4 PM Wednesday
		isDue := testedSchedule.IsDue(now, nil)
		assert.True(t, isDue)
	})

	t.Run(
		"is not due on Wednesday at 14:59 when no run happened this week",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 17, 14, 59, 0, 0, time.UTC)
			lastRun := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.False(t, isDue)
		},
	)

	t.Run(
		"is due on Wednesday at exactly 15:00 when no run happened this week",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 17, 15, 0, 0, 0, time.UTC)
			lastRun := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run("is due on Wednesday at 15:01 when no run happened this week", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 17, 15, 1, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run(
		"is not due when the run of Wednesday 15:00 already happened",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC) // Thursday

			// Wednesday this week at scheduled time
			lastRun := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.False(t, isDue)
		},
	)

	t.Run(
		"is due at the slot after an earlier manual run",
		func(t *testing.T) {
			t.Parallel()

			// Wednesday at scheduled time
			now := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			// Manual backup same day, before scheduled time
			lastRun := time.Date(
				2024,
				1,
				17,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is not due when a manual run happened after the slot",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC) // Thursday
			// Manual backup after scheduled time
			lastRun := time.Date(
				2024,
				1,
				17,
				16,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.False(t, isDue)
		},
	)

	t.Run(
		"is due after the slot when the run was missed",
		func(t *testing.T) {
			t.Parallel()

			// Thursday after missed Wednesday
			now := time.Date(
				2024,
				1,
				18,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			lastRun := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due at this week's slot when the last run was last week",
		func(t *testing.T) {
			t.Parallel()

			// Wednesday at scheduled time
			now := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			lastRun := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due on Friday at 00:00 after a manual run on Wednesday",
		func(t *testing.T) {
			t.Parallel()

			timeOfDay := "00:00"
			weekday := 5 // Friday (0=Sunday, 1=Monday, ..., 5=Friday)
			fridaySchedule := &Schedule{
				Type:      TypeWeekly,
				TimeOfDay: &timeOfDay,
				Weekday:   &weekday,
			}

			// Friday at 00:00 - scheduled backup time
			friday := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC) // Friday Jan 19, 2024
			// Manual backup was done on Wednesday
			wednesdayBackup := time.Date(
				2024,
				1,
				17,
				21,
				0,
				0,
				0,
				time.UTC,
			) // Wednesday Jan 17, 2024 at 21:00

			isDue := fridaySchedule.IsDue(friday, &wednesdayBackup)
			assert.True(
				t,
				isDue,
				"Friday scheduled backup should trigger despite Wednesday manual backup",
			)
		},
	)
}

func TestIsDueForMonthlySchedule(t *testing.T) {
	t.Parallel()

	timeOfDay := "08:00"
	dayOfMonth := 10
	testedSchedule := &Schedule{
		Type:       TypeMonthly,
		TimeOfDay:  &timeOfDay,
		DayOfMonth: &dayOfMonth,
	}

	t.Run("is due without a previous run", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		isDue := testedSchedule.IsDue(now, nil)
		assert.True(t, isDue)
	})

	t.Run(
		"is not due on the 10th at 07:59 when no run happened this month",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 10, 7, 59, 0, 0, time.UTC)
			lastRun := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.False(t, isDue)
		},
	)

	t.Run(
		"is due on the 10th at exactly 08:00 when no run happened this month",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
			lastRun := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due on the 10th after 08:00 when no run happened this month",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 10, 8, 1, 0, 0, time.UTC)
			lastRun := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is due on the 11th when the run of the 10th was missed",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 11, 10, 0, 0, 0, time.UTC)
			lastRun := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run("is not due when the run already happened at the slot", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC) // This month at scheduled time
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run(
		"is due at the slot after an earlier manual run",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC) // Scheduled time
			// Manual backup earlier this month
			lastRun := time.Date(
				2024,
				1,
				5,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	t.Run(
		"is not due when a manual run happened after the slot",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
			// Manual backup after scheduled time
			lastRun := time.Date(
				2024,
				1,
				10,
				9,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.False(t, isDue)
		},
	)

	t.Run(
		"is due at this month's slot when the last run was last month",
		func(t *testing.T) {
			t.Parallel()

			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
			// Previous month at scheduled time
			lastRun := time.Date(
				2023,
				12,
				10,
				8,
				0,
				0,
				0,
				time.UTC,
			)
			isDue := testedSchedule.IsDue(now, &lastRun)
			assert.True(t, isDue)
		},
	)

	octoberSlot := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	novemberSlot := time.Date(2026, 11, 30, 8, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		dayOfMonth    int
		now           time.Time
		lastRun       time.Time
		expectedIsDue bool
	}{
		{
			name:          "is not due on the last day before the time when day 31 is past the end of the month",
			dayOfMonth:    31,
			now:           time.Date(2026, 11, 30, 7, 59, 0, 0, time.UTC),
			lastRun:       octoberSlot,
			expectedIsDue: false,
		},
		{
			name:          "is due on the last day at the time when day 31 is past the end of the month",
			dayOfMonth:    31,
			now:           novemberSlot,
			lastRun:       octoberSlot,
			expectedIsDue: true,
		},
		{
			name:          "is due on a late pass on the last day when day 31 is past the end of the month",
			dayOfMonth:    31,
			now:           time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC),
			lastRun:       octoberSlot,
			expectedIsDue: true,
		},
		{
			name:          "is not due on December 1 after a run on the November slot",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 1, 8, 0, 0, 0, time.UTC),
			lastRun:       novemberSlot,
			expectedIsDue: false,
		},
		{
			name:          "is not due on December 30 after a run on the November slot",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 30, 8, 0, 0, 0, time.UTC),
			lastRun:       novemberSlot,
			expectedIsDue: false,
		},
		{
			name:          "is due on December 31 at the time after a run on the November slot",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 31, 8, 0, 0, 0, time.UTC),
			lastRun:       novemberSlot,
			expectedIsDue: true,
		},
		{
			name:          "is not due on December 1 when the November slot was missed",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 1, 8, 0, 0, 0, time.UTC),
			lastRun:       octoberSlot,
			expectedIsDue: false,
		},
		{
			name:          "is not due on December 30 when the November slot was missed",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 30, 8, 0, 0, 0, time.UTC),
			lastRun:       octoberSlot,
			expectedIsDue: false,
		},
		{
			name:          "is due on December 31 at the time when the November slot was missed",
			dayOfMonth:    31,
			now:           time.Date(2026, 12, 31, 8, 0, 0, 0, time.UTC),
			lastRun:       octoberSlot,
			expectedIsDue: true,
		},
		{
			name:          "is due at the November slot after a manual run on November 5",
			dayOfMonth:    31,
			now:           novemberSlot,
			lastRun:       time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC),
			expectedIsDue: true,
		},
		{
			name:          "is due on February 28 at the time with day 30 in a common year",
			dayOfMonth:    30,
			now:           time.Date(2026, 2, 28, 8, 0, 0, 0, time.UTC),
			lastRun:       time.Date(2026, 1, 30, 8, 0, 0, 0, time.UTC),
			expectedIsDue: true,
		},
		{
			name:          "is not due on February 28 with day 30 in a leap year",
			dayOfMonth:    30,
			now:           time.Date(2028, 2, 28, 8, 0, 0, 0, time.UTC),
			lastRun:       time.Date(2028, 1, 30, 8, 0, 0, 0, time.UTC),
			expectedIsDue: false,
		},
		{
			name:          "is due on February 29 at the time with day 30 in a leap year",
			dayOfMonth:    30,
			now:           time.Date(2028, 2, 29, 8, 0, 0, 0, time.UTC),
			lastRun:       time.Date(2028, 1, 30, 8, 0, 0, 0, time.UTC),
			expectedIsDue: true,
		},
		{
			name:          "is not due on October 30 with day 31 in a month that has it",
			dayOfMonth:    31,
			now:           time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC),
			lastRun:       time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC),
			expectedIsDue: false,
		},
		{
			name:          "is due on October 31 at the time with day 31 in a month that has it",
			dayOfMonth:    31,
			now:           octoberSlot,
			lastRun:       time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC),
			expectedIsDue: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			timeOfDay := "08:00"
			testedSchedule := Schedule{
				Type:       TypeMonthly,
				TimeOfDay:  &timeOfDay,
				DayOfMonth: &testCase.dayOfMonth,
			}

			isDue := testedSchedule.IsDue(testCase.now, &testCase.lastRun)

			assert.Equal(t, testCase.expectedIsDue, isDue)
		})
	}
}

func TestIsDueForCronSchedule(t *testing.T) {
	t.Parallel()

	cronExpr := "0 2 * * *" // Daily at 2:00 AM
	testedSchedule := &Schedule{
		Type:           TypeCron,
		CronExpression: &cronExpr,
	}

	t.Run("is due without a previous run", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		isDue := testedSchedule.IsDue(now, nil)
		assert.True(t, isDue)
	})

	t.Run("is not due before the cron time", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 1, 59, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is due exactly at the cron time", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is due after the cron time", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is not due when a run already happened after the cron time", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 15, 2, 5, 0, 0, time.UTC) // Today at 2:05 AM
		isDue := testedSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is due on Monday at 03:00 with a weekly cron expression", func(t *testing.T) {
		t.Parallel()

		weeklyCron := "0 3 * * 1" // Every Monday at 3 AM
		weeklySchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &weeklyCron,
		}

		// Monday Jan 15, 2024 at 3:00 AM
		monday := time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC)
		// Last backup was previous Monday
		lastRun := time.Date(2024, 1, 8, 3, 0, 0, 0, time.UTC)

		isDue := weeklySchedule.IsDue(monday, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is due on the 15th at 04:30 with a cron expression of two days a month", func(t *testing.T) {
		t.Parallel()

		complexCron := "30 4 1,15 * *" // 1st and 15th of each month at 4:30 AM
		complexSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &complexCron,
		}

		// Jan 15, 2024 at 4:30 AM
		now := time.Date(2024, 1, 15, 4, 30, 0, 0, time.UTC)
		// Last backup was Jan 1
		lastRun := time.Date(2024, 1, 1, 4, 30, 0, 0, time.UTC)

		isDue := complexSchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is due every 6 hours with a cron expression of every 6 hours", func(t *testing.T) {
		t.Parallel()

		sixHourlyCron := "0 */6 * * *" // Every 6 hours (0:00, 6:00, 12:00, 18:00)
		sixHourlySchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &sixHourlyCron,
		}

		// 12:00 - next trigger after 6:00
		now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		// Last backup was at 6:00
		lastRun := time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC)

		isDue := sixHourlySchedule.IsDue(now, &lastRun)
		assert.True(t, isDue)
	})

	t.Run("is not due with an invalid cron expression", func(t *testing.T) {
		t.Parallel()

		invalidCron := "invalid cron"
		invalidSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &invalidCron,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		isDue := invalidSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is not due with a last run when the cron expression has no date to run on", func(t *testing.T) {
		t.Parallel()

		februaryThirtyFirst := "0 0 31 2 *"
		undatedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &februaryThirtyFirst,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := now.Add(-time.Minute)

		assert.False(t, undatedSchedule.IsDue(now, &lastRun))
	})

	t.Run("is not due without a last run when the cron expression has no date to run on", func(t *testing.T) {
		t.Parallel()

		februaryThirtyFirst := "0 0 31 2 *"
		undatedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &februaryThirtyFirst,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		assert.False(t, undatedSchedule.IsDue(now, nil))
	})

	t.Run("is due at the cron time with a time zone prefix", func(t *testing.T) {
		t.Parallel()

		zonedCron := "TZ=Europe/Moscow 0 2 * * *"
		zonedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &zonedCron,
		}

		lastRun := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)
		beforeSlot := time.Date(2024, 1, 14, 22, 59, 0, 0, time.UTC)
		atSlot := time.Date(2024, 1, 14, 23, 0, 0, 0, time.UTC)

		assert.False(t, zonedSchedule.IsDue(beforeSlot, &lastRun))
		assert.True(t, zonedSchedule.IsDue(atSlot, &lastRun))
	})

	t.Run("is not due with an empty cron expression", func(t *testing.T) {
		t.Parallel()

		emptyCron := ""
		emptySchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &emptyCron,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		isDue := emptySchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})

	t.Run("is not due without a cron expression", func(t *testing.T) {
		t.Parallel()

		nilSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: nil,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastRun := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		isDue := nilSchedule.IsDue(now, &lastRun)
		assert.False(t, isDue)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("fails for a daily schedule without a time of day", func(t *testing.T) {
		t.Parallel()

		testedSchedule := &Schedule{
			Type: TypeDaily,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrTimeOfDayRequired)
	})

	t.Run("fails for a weekly schedule without a weekday", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		testedSchedule := &Schedule{
			Type:      TypeWeekly,
			TimeOfDay: &timeOfDay,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrWeekdayRequired)
	})

	t.Run("fails for a monthly schedule without a day of month", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		testedSchedule := &Schedule{
			Type:      TypeMonthly,
			TimeOfDay: &timeOfDay,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrDayOfMonthRequired)
	})

	t.Run("passes for an hourly schedule without other fields", func(t *testing.T) {
		t.Parallel()

		testedSchedule := &Schedule{
			Type: TypeHourly,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("passes for a weekly schedule with a weekday and a time of day", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		weekday := 1
		testedSchedule := &Schedule{
			Type:      TypeWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("fails for a daily schedule with an invalid time of day", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "25:00"
		testedSchedule := &Schedule{
			Type:      TypeDaily,
			TimeOfDay: &timeOfDay,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTimeOfDay)
	})

	t.Run("fails for a daily schedule with an empty time of day", func(t *testing.T) {
		t.Parallel()

		timeOfDay := ""
		testedSchedule := &Schedule{
			Type:      TypeDaily,
			TimeOfDay: &timeOfDay,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTimeOfDay)
	})

	t.Run("fails for a weekly schedule with a weekday above 7", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		weekday := 8
		testedSchedule := &Schedule{
			Type:      TypeWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWeekdayOutOfRange)
	})

	t.Run("fails for a weekly schedule with a negative weekday", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		weekday := -1
		testedSchedule := &Schedule{
			Type:      TypeWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWeekdayOutOfRange)
	})

	t.Run("passes for a weekly schedule with weekday 7 as Sunday", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		weekday := 7
		testedSchedule := &Schedule{
			Type:      TypeWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("passes for a monthly schedule with a day of month and a time of day", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		dayOfMonth := 15
		testedSchedule := &Schedule{
			Type:       TypeMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("fails for a monthly schedule with day of month 0", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		dayOfMonth := 0
		testedSchedule := &Schedule{
			Type:       TypeMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDayOfMonthOutOfRange)
	})

	t.Run("fails for a monthly schedule with a day of month above 31", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		dayOfMonth := 32
		testedSchedule := &Schedule{
			Type:       TypeMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := testedSchedule.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDayOfMonthOutOfRange)
	})

	t.Run("passes for a monthly schedule with day of month 31", func(t *testing.T) {
		t.Parallel()

		timeOfDay := "09:00"
		dayOfMonth := 31
		testedSchedule := &Schedule{
			Type:       TypeMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("fails for a cron schedule without an expression", func(t *testing.T) {
		t.Parallel()

		testedSchedule := &Schedule{
			Type: TypeCron,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrCronExpressionMissing)
	})

	t.Run("fails for a cron schedule with an empty expression", func(t *testing.T) {
		t.Parallel()

		emptyCron := ""
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &emptyCron,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrCronExpressionMissing)
	})

	t.Run("fails for a cron schedule with an invalid expression", func(t *testing.T) {
		t.Parallel()

		invalidCron := "invalid cron"
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &invalidCron,
		}
		err := testedSchedule.Validate()
		assert.ErrorIs(t, err, ErrInvalidCronExpression)
	})

	for _, undatedCron := range []string{"0 0 31 2 *", "0 0 31 2,4 *"} {
		t.Run("fails for a cron schedule with no date to run on with "+undatedCron, func(t *testing.T) {
			t.Parallel()

			testedSchedule := &Schedule{
				Type:           TypeCron,
				CronExpression: &undatedCron,
			}

			err := testedSchedule.Validate()
			assert.ErrorIs(t, err, ErrInvalidCronExpression)
		})
	}

	t.Run("passes for a cron schedule with a date only in leap years", func(t *testing.T) {
		t.Parallel()

		februaryTwentyNinth := "0 0 29 2 *"
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &februaryTwentyNinth,
		}

		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("passes for a cron schedule with a time zone prefix", func(t *testing.T) {
		t.Parallel()

		zonedCron := "TZ=Europe/Moscow 0 2 * * *"
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &zonedCron,
		}

		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("passes for a cron schedule with a daily expression", func(t *testing.T) {
		t.Parallel()

		cronExpr := "0 2 * * *" // Daily at 2 AM
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &cronExpr,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})

	t.Run("passes for a cron schedule with a complex expression", func(t *testing.T) {
		t.Parallel()

		cronExpr := "30 4 1,15 * *" // 1st and 15th of each month at 4:30 AM
		testedSchedule := &Schedule{
			Type:           TypeCron,
			CronExpression: &cronExpr,
		}
		err := testedSchedule.Validate()
		assert.NoError(t, err)
	})
}
