// Package gfs selects the items a grandfather-father-son retention keeps: the newest item of each hour,
// day, week, month and year, for as many periods as each level names.
package gfs

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func GetItemsToRetain(
	items []Item,
	hours, days, weeks, months, years int,
) map[uuid.UUID][]Tier {
	keep := make(map[uuid.UUID][]Tier)

	if len(items) == 0 {
		return keep
	}

	itemsNewestFirst := sortNewestFirst(items)

	hoursSeen := make(map[string]bool)
	daysSeen := make(map[string]bool)
	weeksSeen := make(map[string]bool)
	monthsSeen := make(map[string]bool)
	yearsSeen := make(map[string]bool)

	hoursKept, daysKept, weeksKept, monthsKept, yearsKept := 0, 0, 0, 0, 0

	// Compute per-level time-window cutoffs so higher-frequency slots
	// cannot absorb items that belong to lower-frequency levels.
	ref := itemsNewestFirst[0].CreatedAt

	rawHourlyCutoff := ref.Add(-time.Duration(hours) * time.Hour)
	rawDailyCutoff := ref.Add(-time.Duration(days) * 24 * time.Hour)
	rawWeeklyCutoff := ref.Add(-time.Duration(weeks) * 7 * 24 * time.Hour)
	rawMonthlyCutoff := ref.AddDate(0, -months, 0)
	rawYearlyCutoff := ref.AddDate(-years, 0, 0)

	// Hierarchical capping: each level's window cannot extend further back
	// than the nearest active lower-frequency level's window.
	yearlyCutoff := rawYearlyCutoff

	monthlyCutoff := rawMonthlyCutoff
	if years > 0 {
		monthlyCutoff = laterOf(monthlyCutoff, yearlyCutoff)
	}

	weeklyCutoff := rawWeeklyCutoff
	if months > 0 {
		weeklyCutoff = laterOf(weeklyCutoff, monthlyCutoff)
	} else if years > 0 {
		weeklyCutoff = laterOf(weeklyCutoff, yearlyCutoff)
	}

	dailyCutoff := rawDailyCutoff
	switch {
	case weeks > 0:
		dailyCutoff = laterOf(dailyCutoff, weeklyCutoff)
	case months > 0:
		dailyCutoff = laterOf(dailyCutoff, monthlyCutoff)
	case years > 0:
		dailyCutoff = laterOf(dailyCutoff, yearlyCutoff)
	}

	hourlyCutoff := rawHourlyCutoff
	switch {
	case days > 0:
		hourlyCutoff = laterOf(hourlyCutoff, dailyCutoff)
	case weeks > 0:
		hourlyCutoff = laterOf(hourlyCutoff, weeklyCutoff)
	case months > 0:
		hourlyCutoff = laterOf(hourlyCutoff, monthlyCutoff)
	case years > 0:
		hourlyCutoff = laterOf(hourlyCutoff, yearlyCutoff)
	}

	for _, item := range itemsNewestFirst {
		t := item.CreatedAt

		hourKey := t.Format("2006-01-02-15")
		dayKey := t.Format("2006-01-02")
		weekYear, week := t.ISOWeek()
		weekKey := fmt.Sprintf("%d-%02d", weekYear, week)
		monthKey := t.Format("2006-01")
		yearKey := t.Format("2006")

		if hours > 0 && hoursKept < hours && !hoursSeen[hourKey] && t.After(hourlyCutoff) {
			keep[item.ID] = append(keep[item.ID], TierHourly)
			hoursSeen[hourKey] = true
			hoursKept++
		}

		if days > 0 && daysKept < days && !daysSeen[dayKey] && t.After(dailyCutoff) {
			keep[item.ID] = append(keep[item.ID], TierDaily)
			daysSeen[dayKey] = true
			daysKept++
		}

		if weeks > 0 && weeksKept < weeks && !weeksSeen[weekKey] && t.After(weeklyCutoff) {
			keep[item.ID] = append(keep[item.ID], TierWeekly)
			weeksSeen[weekKey] = true
			weeksKept++
		}

		if months > 0 && monthsKept < months && !monthsSeen[monthKey] && t.After(monthlyCutoff) {
			keep[item.ID] = append(keep[item.ID], TierMonthly)
			monthsSeen[monthKey] = true
			monthsKept++
		}

		if years > 0 && yearsKept < years && !yearsSeen[yearKey] && t.After(yearlyCutoff) {
			keep[item.ID] = append(keep[item.ID], TierYearly)
			yearsSeen[yearKey] = true
			yearsKept++
		}
	}

	return keep
}

func laterOf(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}

// sortNewestFirst returns a copy of items ordered from newest to oldest,
// leaving the caller's slice untouched. Ties are broken by ID so the result
// is deterministic when several items share the exact same timestamp.
func sortNewestFirst(items []Item) []Item {
	sorted := slices.Clone(items)

	slices.SortFunc(sorted, func(a, b Item) int {
		if byTime := b.CreatedAt.Compare(a.CreatedAt); byTime != 0 {
			return byTime
		}

		return cmp.Compare(a.ID.String(), b.ID.String())
	})

	return sorted
}
