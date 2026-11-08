package gfs

import (
	"cmp"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func GetItemsToRetain(selection Selection) map[uuid.UUID][]Tier {
	retainedTiers := make(map[uuid.UUID][]Tier)

	if len(selection.Items) == 0 {
		return retainedTiers
	}

	itemsNewestFirst := sortNewestFirst(selection.Items)
	levels := []struct {
		tier  Tier
		count int
	}{
		{tier: TierHourly, count: selection.Hours},
		{tier: TierDaily, count: selection.Days},
		{tier: TierWeekly, count: selection.Weeks},
		{tier: TierMonthly, count: selection.Months},
		{tier: TierYearly, count: selection.Years},
	}

	for _, level := range levels {
		if level.count <= 0 {
			continue
		}

		referencePeriod := getPeriodStart(itemsNewestFirst[0].CreatedAt, level.tier)
		referencePeriodNumber := getPeriodNumber(referencePeriod, level.tier)
		oldestPeriodAge := level.count

		if level.tier == TierHourly || level.tier == TierDaily {
			oldestPeriodAge--
		}

		seenPeriods := make(map[time.Time]bool)
		for _, item := range itemsNewestFirst {
			periodStart := getPeriodStart(item.CreatedAt, level.tier)

			if referencePeriodNumber-getPeriodNumber(periodStart, level.tier) > int64(oldestPeriodAge) {
				break
			}

			if seenPeriods[periodStart] {
				continue
			}

			seenPeriods[periodStart] = true
			retainedTiers[item.ID] = append(retainedTiers[item.ID], level.tier)
		}
	}

	return retainedTiers
}

func getPeriodStart(createdAt time.Time, tier Tier) time.Time {
	createdAt = createdAt.UTC()
	dayStart := time.Date(createdAt.Year(), createdAt.Month(), createdAt.Day(), 0, 0, 0, 0, time.UTC)

	switch tier {
	case TierHourly:
		return createdAt.Truncate(time.Hour)
	case TierDaily:
		return dayStart
	case TierWeekly:
		daysSinceMonday := (int(createdAt.Weekday()) + 6) % 7
		return dayStart.AddDate(0, 0, -daysSinceMonday)
	case TierMonthly:
		return time.Date(createdAt.Year(), createdAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	case TierYearly:
		return time.Date(createdAt.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	return time.Time{}
}

func getPeriodNumber(periodStart time.Time, tier Tier) int64 {
	switch tier {
	case TierHourly:
		return periodStart.Unix() / 3600
	case TierDaily:
		return periodStart.Unix() / 86400
	case TierWeekly:
		return (periodStart.Unix() + 3*86400) / (7 * 86400)
	case TierMonthly:
		return int64(periodStart.Year())*12 + int64(periodStart.Month())
	case TierYearly:
		return int64(periodStart.Year())
	}

	return 0
}

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
