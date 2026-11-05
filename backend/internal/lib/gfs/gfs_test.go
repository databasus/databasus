package gfs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// itemAt builds an Item with a deterministic ID derived from the offset so a
// test can refer to the same instant twice and still get a stable identity.
func itemAt(reference time.Time, offset time.Duration) Item {
	at := reference.Add(offset)

	return Item{
		ID:        uuid.NewSHA1(uuid.Nil, []byte(at.Format(time.RFC3339Nano))),
		CreatedAt: at,
	}
}

func TestGetItemsToRetain(t *testing.T) {
	t.Parallel()

	t.Run("keeps nothing without items", func(t *testing.T) {
		t.Parallel()
		keep := GetItemsToRetain(nil, 1, 1, 1, 1, 1)

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

		keep := GetItemsToRetain(items, 0, 0, 0, 0, 0)

		assert.Empty(t, keep)
	})

	t.Run("keeps an item once with every level that claims it", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		newest := itemAt(reference, 0)
		older := itemAt(reference, -90*24*time.Hour)

		keep := GetItemsToRetain([]Item{newest, older}, 1, 1, 1, 1, 1)

		// The single newest item simultaneously fills the hourly, daily, weekly,
		// monthly, and yearly slot, so each level is satisfied by it alone.
		assert.NotEmpty(t, keep[newest.ID])
		assert.Len(t, keep, 1)
	})

	t.Run("sorts the items before applying the levels when they are unsorted", func(t *testing.T) {
		t.Parallel()
		reference := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
		newest := itemAt(reference, 0)
		middle := itemAt(reference, -1*time.Hour)
		oldest := itemAt(reference, -2*time.Hour)

		// Deliberately shuffled: the function must sort newest-first itself.
		unsorted := []Item{oldest, newest, middle}

		keep := GetItemsToRetain(unsorted, 2, 0, 0, 0, 0)

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

		GetItemsToRetain(items, 5, 0, 0, 0, 0)

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
			[]Item{hourMiddle, hourOldest, hourNewest, previousHour}, 2, 0, 0, 0, 0,
		)

		// All three of the first three share the same hour bucket, so only the
		// newest of them is kept; the previous hour fills the second hourly slot.
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

		keep := GetItemsToRetain(items, 2, 0, 0, 0, 0)

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
			[]Item{today, todayEarlier, yesterday, twoDaysAgo}, 0, 3, 0, 0, 0,
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

		// Two items in the same hour bucket: exactly one survives, regardless of
		// the input order, and the choice is stable across repeated runs.
		keepAsGiven := GetItemsToRetain([]Item{first, second}, 1, 0, 0, 0, 0)
		keepReversed := GetItemsToRetain([]Item{second, first}, 1, 0, 0, 0, 0)

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

		keep := GetItemsToRetain(items, 2, 2, 2, 2, 2)

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

		keep := GetItemsToRetain([]Item{recent, ancient}, 1, 1, 1, 1, 1)

		assert.NotEmpty(t, keep[recent.ID])
		assert.Empty(t, keep[ancient.ID])
		assert.Len(t, keep, 1)
	})
}
