package clock

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystem(t *testing.T) {
	t.Parallel()

	t.Run("returns the time in utc", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, time.UTC, NewSystem().Now().Location())
	})
}

func TestManual(t *testing.T) {
	t.Parallel()

	t.Run("returns the start in utc when started in another zone", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))

		manualClock := NewManual(start)

		assert.Equal(t, time.UTC, manualClock.Now().Location())
		assert.True(t, start.Equal(manualClock.Now()))
	})

	t.Run("stands still when not moved", func(t *testing.T) {
		t.Parallel()
		manualClock := NewManual(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))

		firstReading := manualClock.Now()
		secondReading := manualClock.Now()

		assert.Equal(t, firstReading, secondReading)
	})

	t.Run("moves by the advanced duration", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		manualClock := NewManual(start)

		manualClock.Advance(90 * time.Minute)

		assert.Equal(t, start.Add(90*time.Minute), manualClock.Now())
	})

	t.Run("returns the set time in utc when set in another zone", func(t *testing.T) {
		t.Parallel()
		manualClock := NewManual(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
		setTime := time.Date(2027, 1, 1, 0, 0, 0, 0, time.FixedZone("UTC-5", -5*60*60))

		manualClock.Set(setTime)

		assert.Equal(t, time.UTC, manualClock.Now().Location())
		assert.True(t, setTime.Equal(manualClock.Now()))
	})

	t.Run("adds up every advance when advanced concurrently", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		manualClock := NewManual(start)

		var waitGroup sync.WaitGroup
		for range 100 {
			waitGroup.Go(func() {
				manualClock.Advance(time.Second)
			})
		}
		waitGroup.Wait()

		assert.Equal(t, start.Add(100*time.Second), manualClock.Now())
	})
}
