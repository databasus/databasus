package gfs

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func itemAt(reference time.Time, offset time.Duration) Item {
	at := reference.Add(offset)

	return Item{
		ID:        uuid.NewSHA1(uuid.Nil, []byte(at.Format(time.RFC3339Nano))),
		CreatedAt: at,
	}
}

func TestGetItemsToRetain(t *testing.T) {
	t.Parallel()

	t.Run("keeps completed weekly and monthly points when daily cleanup repeats", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
		var survivingItems []Item

		for day := range 25 {
			survivingItems = append(survivingItems, itemAt(start, time.Duration(day)*24*time.Hour))

			retainedItems := GetItemsToRetain(
				Selection{Items: survivingItems, Hours: 0, Days: 2, Weeks: 1, Months: 1, Years: 0},
			)

			var nextItems []Item
			for _, survivingItem := range survivingItems {
				if len(retainedItems[survivingItem.ID]) > 0 {
					nextItems = append(nextItems, survivingItem)
				}
			}

			survivingItems = nextItems
		}

		assert.ElementsMatch(t, []Item{
			itemAt(start, 19*24*time.Hour),
			itemAt(start, 20*24*time.Hour),
			itemAt(start, 23*24*time.Hour),
			itemAt(start, 24*24*time.Hour),
		}, survivingItems)
	})

	t.Run("keeps nothing without items", func(t *testing.T) {
		t.Parallel()
		keep := GetItemsToRetain(Selection{Items: nil, Hours: 1, Days: 1, Weeks: 1, Months: 1, Years: 1})

		assert.Empty(t, keep)
	})

	t.Run("keeps nothing when every level is zero", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		items := []Item{
			itemAt(reference, 0),
			itemAt(reference, -1*time.Hour),
			itemAt(reference, -24*time.Hour),
		}

		keep := GetItemsToRetain(Selection{Items: items, Hours: 0, Days: 0, Weeks: 0, Months: 0, Years: 0})

		assert.Empty(t, keep)
	})

	t.Run("keeps an item once with every level that claims it", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		newest := itemAt(reference, 0)
		older := itemAt(reference, -10*365*24*time.Hour)

		keep := GetItemsToRetain(
			Selection{Items: []Item{newest, older}, Hours: 1, Days: 1, Weeks: 1, Months: 1, Years: 1},
		)

		assert.Equal(t, []Tier{TierHourly, TierDaily, TierWeekly, TierMonthly, TierYearly}, keep[newest.ID])
		assert.Len(t, keep, 1)
	})

	t.Run("sorts the items before applying the levels when they are unsorted", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		newest := itemAt(reference, 0)
		middle := itemAt(reference, -1*time.Hour)
		oldest := itemAt(reference, -2*time.Hour)

		unsorted := []Item{oldest, newest, middle}

		keep := GetItemsToRetain(Selection{Items: unsorted, Hours: 2, Days: 0, Weeks: 0, Months: 0, Years: 0})

		assert.NotEmpty(t, keep[newest.ID])
		assert.NotEmpty(t, keep[middle.ID])
		assert.Empty(t, keep[oldest.ID])
		assert.Len(t, keep, 2)
	})

	t.Run("leaves the caller slice unchanged", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		oldest := itemAt(reference, -2*time.Hour)
		newest := itemAt(reference, 0)
		items := []Item{oldest, newest}

		GetItemsToRetain(Selection{Items: items, Hours: 5, Days: 0, Weeks: 0, Months: 0, Years: 0})

		assert.Equal(t, oldest.ID, items[0].ID)
		assert.Equal(t, newest.ID, items[1].ID)
	})

	t.Run("keeps only the newest item of an hour when several share it", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 30, 0, 0, time.UTC)
		hourNewest := itemAt(reference, 0)
		hourMiddle := itemAt(reference, -10*time.Minute)
		hourOldest := itemAt(reference, -20*time.Minute)
		previousHour := itemAt(reference, -1*time.Hour)

		keep := GetItemsToRetain(
			Selection{
				Items:  []Item{hourMiddle, hourOldest, hourNewest, previousHour},
				Hours:  2,
				Days:   0,
				Weeks:  0,
				Months: 0,
				Years:  0,
			},
		)

		assert.NotEmpty(t, keep[hourNewest.ID])
		assert.NotEmpty(t, keep[previousHour.ID])
		assert.Empty(t, keep[hourMiddle.ID])
		assert.Empty(t, keep[hourOldest.ID])
		assert.Len(t, keep, 2)
	})

	t.Run("stops keeping hourly items after the hourly count", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		items := []Item{
			itemAt(reference, 0),
			itemAt(reference, -1*time.Hour),
			itemAt(reference, -2*time.Hour),
			itemAt(reference, -3*time.Hour),
		}

		keep := GetItemsToRetain(Selection{Items: items, Hours: 2, Days: 0, Weeks: 0, Months: 0, Years: 0})

		assert.NotEmpty(t, keep[items[0].ID])
		assert.NotEmpty(t, keep[items[1].ID])
		assert.Empty(t, keep[items[2].ID])
		assert.Empty(t, keep[items[3].ID])
		assert.Len(t, keep, 2)
	})

	t.Run("keeps one item per day with daily retention", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		today := itemAt(reference, 0)
		todayEarlier := itemAt(reference, -3*time.Hour)
		yesterday := itemAt(reference, -24*time.Hour)
		twoDaysAgo := itemAt(reference, -48*time.Hour)

		keep := GetItemsToRetain(
			Selection{
				Items:  []Item{today, todayEarlier, yesterday, twoDaysAgo},
				Hours:  0,
				Days:   3,
				Weeks:  0,
				Months: 0,
				Years:  0,
			},
		)

		assert.NotEmpty(t, keep[today.ID])
		assert.NotEmpty(t, keep[yesterday.ID])
		assert.NotEmpty(t, keep[twoDaysAgo.ID])
		assert.Empty(t, keep[todayEarlier.ID])
		assert.Len(t, keep, 3)
	})

	t.Run("picks the same item when items share a timestamp", func(t *testing.T) {
		t.Parallel()
		at := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		first := Item{ID: uuid.New(), CreatedAt: at}
		second := Item{ID: uuid.New(), CreatedAt: at}

		keepAsGiven := GetItemsToRetain(
			Selection{Items: []Item{first, second}, Hours: 1, Days: 0, Weeks: 0, Months: 0, Years: 0},
		)
		keepReversed := GetItemsToRetain(
			Selection{Items: []Item{second, first}, Hours: 1, Days: 0, Weeks: 0, Months: 0, Years: 0},
		)

		assert.Len(t, keepAsGiven, 1)
		assert.Equal(t, keepAsGiven, keepReversed)
	})

	t.Run("keeps the items each level claims with mixed levels", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		now := itemAt(reference, 0)
		oneHourAgo := itemAt(reference, -1*time.Hour)
		yesterday := itemAt(reference, -24*time.Hour)
		lastWeek := itemAt(reference, -8*24*time.Hour)
		lastMonth := itemAt(reference, -40*24*time.Hour)
		lastYear := itemAt(reference, -300*24*time.Hour)

		items := []Item{now, oneHourAgo, yesterday, lastWeek, lastMonth, lastYear}

		keep := GetItemsToRetain(Selection{Items: items, Hours: 2, Days: 2, Weeks: 2, Months: 2, Years: 2})

		assert.NotEmpty(t, keep[now.ID])
		assert.NotEmpty(t, keep[oneHourAgo.ID])
		assert.NotEmpty(t, keep[yesterday.ID])
		assert.NotEmpty(t, keep[lastWeek.ID])
		assert.NotEmpty(t, keep[lastMonth.ID])
		assert.NotEmpty(t, keep[lastYear.ID])
	})

	t.Run("drops items older than every window", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		recent := itemAt(reference, 0)
		ancient := itemAt(reference, -10*365*24*time.Hour)

		keep := GetItemsToRetain(
			Selection{Items: []Item{recent, ancient}, Hours: 1, Days: 1, Weeks: 1, Months: 1, Years: 1},
		)

		assert.NotEmpty(t, keep[recent.ID])
		assert.Empty(t, keep[ancient.ID])
		assert.Len(t, keep, 1)
	})
}

func TestGfsCalendar(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name             string
		selection        Selection
		instants         []string
		retainedInstants []string
	}{
		{
			name:             "keeps current and previous calendar hours when the reference crosses midnight",
			selection:        Selection{Hours: 2},
			instants:         []string{"2026-11-05T00:05:00Z", "2026-11-04T23:00:00Z", "2026-11-04T22:59:59Z"},
			retainedInstants: []string{"2026-11-05T00:05:00Z", "2026-11-04T23:00:00Z"},
		},
		{
			name: "keeps the whole previous day when two daily periods are enabled", selection: Selection{Days: 2},
			instants:         []string{"2026-11-05T23:59:00Z", "2026-11-04T00:00:00Z", "2026-11-03T23:59:59Z"},
			retainedInstants: []string{"2026-11-05T23:59:00Z", "2026-11-04T00:00:00Z"},
		},
		{
			name:      "keeps completed weeks and a candidate when only weekly retention is enabled",
			selection: Selection{Weeks: 2},
			instants: []string{
				"2026-11-05T12:00:00Z",
				"2026-11-01T12:00:00Z",
				"2026-10-25T12:00:00Z",
				"2026-10-18T12:00:00Z",
			},
			retainedInstants: []string{"2026-11-05T12:00:00Z", "2026-11-01T12:00:00Z", "2026-10-25T12:00:00Z"},
		},
		{
			name: "leaves empty weeks empty without filling from older history", selection: Selection{Weeks: 1},
			instants:         []string{"2026-11-05T12:00:00Z", "2026-10-25T12:00:00Z"},
			retainedInstants: []string{"2026-11-05T12:00:00Z"},
		},
		{
			name:      "keeps completed months and a candidate without shorter levels",
			selection: Selection{Months: 2},
			instants: []string{
				"2026-11-05T12:00:00Z",
				"2026-10-31T12:00:00Z",
				"2026-09-30T12:00:00Z",
				"2026-08-31T12:00:00Z",
			},
			retainedInstants: []string{"2026-11-05T12:00:00Z", "2026-10-31T12:00:00Z", "2026-09-30T12:00:00Z"},
		},
		{
			name: "keeps February after a late March backup in a common year", selection: Selection{Months: 1},
			instants:         []string{"2026-03-31T12:00:00Z", "2026-02-28T12:00:00Z", "2026-01-31T12:00:00Z"},
			retainedInstants: []string{"2026-03-31T12:00:00Z", "2026-02-28T12:00:00Z"},
		},
		{
			name: "keeps leap February after a late March backup", selection: Selection{Months: 1},
			instants:         []string{"2028-03-31T12:00:00Z", "2028-02-29T12:00:00Z", "2028-02-28T12:00:00Z"},
			retainedInstants: []string{"2028-03-31T12:00:00Z", "2028-02-29T12:00:00Z"},
		},
		{
			name: "uses UTC months when timestamps have different offsets", selection: Selection{Months: 1},
			instants:         []string{"2026-11-01T01:30:00Z", "2026-11-01T00:30:00+03:00"},
			retainedInstants: []string{"2026-11-01T01:30:00Z", "2026-11-01T00:30:00+03:00"},
		},
		{
			name: "uses UTC days when timestamps have different offsets", selection: Selection{Days: 1},
			instants:         []string{"2026-11-01T01:30:00Z", "2026-11-01T00:30:00+03:00"},
			retainedInstants: []string{"2026-11-01T01:30:00Z"},
		},
		{
			name:      "keeps a Monday boundary when a new ISO week starts",
			selection: Selection{Weeks: 1},
			instants: []string{
				"2027-01-04T00:00:00Z",
				"2027-01-03T23:59:59Z",
				"2026-12-28T00:00:00Z",
				"2026-12-27T23:59:59Z",
			},
			retainedInstants: []string{"2027-01-04T00:00:00Z", "2027-01-03T23:59:59Z"},
		},
		{
			name:      "keeps completed years and a candidate without other levels",
			selection: Selection{Years: 2},
			instants: []string{
				"2027-01-01T00:00:00Z",
				"2026-12-31T23:59:59Z",
				"2025-12-31T23:59:59Z",
				"2024-12-31T23:59:59Z",
			},
			retainedInstants: []string{"2027-01-01T00:00:00Z", "2026-12-31T23:59:59Z", "2025-12-31T23:59:59Z"},
		},
		{
			name:             "keeps an older daily point when monthly retention is shorter",
			selection:        Selection{Days: 40, Months: 1},
			instants:         []string{"2026-11-05T12:00:00Z", "2026-09-27T00:00:00Z", "2026-09-26T23:59:59Z"},
			retainedInstants: []string{"2026-11-05T12:00:00Z", "2026-09-27T00:00:00Z"},
		},
		{
			name:      "keeps one shared point when weekly and monthly periods overlap",
			selection: Selection{Days: 2, Weeks: 1, Months: 1},
			instants: []string{
				"2026-06-04T12:00:00Z",
				"2026-06-03T12:00:00Z",
				"2026-05-31T12:00:00Z",
				"2026-05-30T12:00:00Z",
			},
			retainedInstants: []string{"2026-06-04T12:00:00Z", "2026-06-03T12:00:00Z", "2026-05-31T12:00:00Z"},
		},
		{
			name: "keeps no long-period candidate when long levels are disabled", selection: Selection{Hours: 1},
			instants:         []string{"2026-11-05T12:00:00Z", "2026-11-01T12:00:00Z", "2026-10-31T12:00:00Z"},
			retainedInstants: []string{"2026-11-05T12:00:00Z"},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			selection := scenario.selection
			for _, instant := range scenario.instants {
				selection.Items = append(selection.Items, parseItem(t, instant))
			}

			retainedTiers := GetItemsToRetain(selection)

			retainedIDs := make([]uuid.UUID, 0, len(scenario.retainedInstants))
			for _, instant := range scenario.retainedInstants {
				retainedIDs = append(retainedIDs, parseItem(t, instant).ID)
			}

			assert.ElementsMatch(t, retainedIDs, slices.Collect(maps.Keys(retainedTiers)))
		})
	}
}

func TestGfsRepeatedCleanup(t *testing.T) {
	t.Parallel()

	t.Run("expires previous monthly representatives after the next month arrives", func(t *testing.T) {
		t.Parallel()
		selection := Selection{Days: 2, Weeks: 1, Months: 1}
		start := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

		for day := range 53 {
			selection.Items = append(selection.Items, itemAt(start, time.Duration(day)*24*time.Hour))

			retainedTiers := GetItemsToRetain(selection)

			selection.Items = slices.DeleteFunc(
				selection.Items,
				func(item Item) bool { return len(retainedTiers[item.ID]) == 0 },
			)
		}

		retainedDates := make([]string, 0, len(selection.Items))
		for _, item := range selection.Items {
			retainedDates = append(retainedDates, item.CreatedAt.Format(time.DateOnly))
		}

		assert.ElementsMatch(t, []string{"2026-11-29", "2026-11-30", "2026-12-02", "2026-12-03"}, retainedDates)
	})

	t.Run("promotes and expires yearly candidates after cleanup across year boundaries", func(t *testing.T) {
		t.Parallel()
		selection := Selection{Years: 1}

		for _, instant := range []string{
			"2025-12-30T12:00:00Z", "2025-12-31T12:00:00Z", "2026-01-01T12:00:00Z",
			"2026-12-31T12:00:00Z", "2027-01-01T12:00:00Z", "2027-12-31T12:00:00Z", "2028-01-01T12:00:00Z",
		} {
			selection.Items = append(selection.Items, parseItem(t, instant))

			retainedTiers := GetItemsToRetain(selection)

			selection.Items = slices.DeleteFunc(
				selection.Items,
				func(item Item) bool { return len(retainedTiers[item.ID]) == 0 },
			)
		}

		assert.ElementsMatch(
			t,
			[]Item{parseItem(t, "2027-12-31T12:00:00Z"), parseItem(t, "2028-01-01T12:00:00Z")},
			selection.Items,
		)
	})
}

func parseItem(t *testing.T, instant string) Item {
	t.Helper()
	at, err := time.Parse(time.RFC3339, instant)
	require.NoError(t, err)

	return itemAt(at, 0)
}
