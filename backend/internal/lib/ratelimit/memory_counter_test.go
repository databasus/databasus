package ratelimit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/lib/ratelimit/ratelimittest"
)

func TestMemoryCounter(t *testing.T) {
	t.Parallel()

	t.Run("counts refused attempts in the sliding window", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC))
		counter := newMemoryCounter(cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: manualClock}),
			manualClock)
		attempt := ratelimit.Attempt{Scope: "signin", Identifier: "user", Limit: 2, Window: time.Minute}

		isAllowedByStep := make([]bool, 0, 6)
		for _, pause := range []time.Duration{
			0, 10 * time.Second, 10 * time.Second, 10 * time.Second,
			40 * time.Second, 20 * time.Second,
		} {
			manualClock.Advance(pause)

			isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
			require.NoError(t, err)

			isAllowedByStep = append(isAllowedByStep, isAllowed)
		}

		assert.Equal(t, []bool{true, true, false, false, false, true}, isAllowedByStep)
	})

	t.Run("records nothing when the settings are not positive", func(t *testing.T) {
		t.Parallel()
		systemClock := clock.NewSystem()
		counter := newMemoryCounter(cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: systemClock}),
			systemClock)
		attempt := ratelimit.Attempt{Scope: "signin", Identifier: "user", Limit: 0, Window: time.Minute}

		_, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.ErrorIs(t, err, ratelimit.ErrInvalidSettings)

		attempt.Limit = 1
		isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, isAllowed)
	})

	t.Run("refuses the attempt and returns the store error when the store fails", func(t *testing.T) {
		t.Parallel()
		storeError := errors.New("store failed")
		counter := newMemoryCounter(failingStore{err: storeError}, clock.NewSystem())

		isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), ratelimit.Attempt{
			Scope: "signin", Identifier: "user", Limit: 1, Window: time.Minute,
		})

		assert.False(t, isAllowed)
		require.ErrorIs(t, err, storeError)
	})
}

func TestMemoryCounterContract(t *testing.T) {
	t.Parallel()

	ratelimittest.RunCounterContract(t, func(*testing.T) ratelimit.Counter {
		systemClock := clock.NewSystem()

		return newMemoryCounter(cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 64 * 1024, Clock: systemClock}),
			systemClock)
	})
}

func newMemoryCounter(store cache.Store, clockSource clock.Source) *ratelimit.MemoryCounter {
	return ratelimit.NewMemoryCounter(ratelimit.Deps{Store: store, Clock: clockSource})
}

// failingStore stands in for a store that cannot answer, which the memory store never does.
type failingStore struct {
	err error
}

func (s failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Set(context.Context, cache.Entry) error {
	return s.err
}

func (s failingStore) CreateIfAbsent(context.Context, cache.Entry) (bool, error) {
	return false, s.err
}

func (s failingStore) ReadAndDelete(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Delete(context.Context, string) error {
	return s.err
}
