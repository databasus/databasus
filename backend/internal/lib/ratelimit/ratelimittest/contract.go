// Package ratelimittest checks that a counter keeps the promises of ratelimit.Counter.
package ratelimittest

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/ratelimit"
)

type CounterFactory func(t *testing.T) ratelimit.Counter

func RunCounterContract(t *testing.T, createCounter CounterFactory) {
	t.Helper()

	t.Run("refuses every attempt after the limit when the window has not passed", func(t *testing.T) {
		t.Parallel()
		counter := createCounter(t)
		attempt := ratelimit.Attempt{
			Scope: "signin", Identifier: "user", Limit: 2, Window: time.Minute,
		}

		firstIsAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, firstIsAllowed)
		secondIsAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, secondIsAllowed)
		thirdIsAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.False(t, thirdIsAllowed)
		fourthIsAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.False(t, fourthIsAllowed)
	})

	t.Run("counts each scope and identifier apart", func(t *testing.T) {
		t.Parallel()
		counter := createCounter(t)
		attempt := ratelimit.Attempt{
			Scope: "signin", Identifier: "first", Limit: 1, Window: time.Minute,
		}
		isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, isAllowed)
		isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.False(t, isAllowed)

		attempt.Identifier = "second"
		isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, isAllowed)
		attempt.Scope = "password_reset"
		attempt.Identifier = "first"
		isAllowed, err = counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
		require.NoError(t, err)
		assert.True(t, isAllowed)
	})

	t.Run("allows one attempt when callers race for the last allowed one", func(t *testing.T) {
		t.Parallel()
		counter := createCounter(t)
		attempt := ratelimit.Attempt{
			Scope: "signin", Identifier: "user", Limit: 5, Window: time.Minute,
		}
		for range 4 {
			isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
			require.NoError(t, err)
			require.True(t, isAllowed)
		}

		const callerCount = 32
		var allowedCount atomic.Int64
		operationErrors := make(chan error, callerCount)
		var waitGroup sync.WaitGroup

		for range callerCount {
			waitGroup.Go(func() {
				isAllowed, err := counter.RecordAttemptAndCheckIsAllowed(t.Context(), attempt)
				operationErrors <- err
				if isAllowed {
					allowedCount.Add(1)
				}
			})
		}

		waitGroup.Wait()
		close(operationErrors)

		for operationError := range operationErrors {
			require.NoError(t, operationError)
		}

		assert.Equal(t, int64(1), allowedCount.Load())
	})
}
