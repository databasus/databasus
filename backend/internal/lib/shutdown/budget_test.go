package shutdown

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
)

func TestBudget(t *testing.T) {
	t.Parallel()

	t.Run("preserves values and its own timeout after ordinary cancellation", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			type contextKey struct{}
			budget := New(Deps{Clock: clock.NewSystem()})
			runCtx, cancelRun := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "restore"))
			cancelRun()
			startedAt := time.Now()
			cleanupCtx, cancelCleanup := budget.NewContext(
				runCtx,
				ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
			defer cancelCleanup()

			require.NoError(t, cleanupCtx.Err())
			assert.Equal(t, "restore", cleanupCtx.Value(contextKey{}))
			<-cleanupCtx.Done()
			assert.Equal(t, 30*time.Second, time.Since(startedAt))
		})
	})

	t.Run("shortens an active cleanup when shutdown begins", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			budget := New(Deps{Clock: clock.NewSystem()})
			cleanupCtx, cancelCleanup := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
			defer cancelCleanup()
			startedAt := time.Now()
			drainDeadline := budget.Begin(6 * time.Second)
			assert.Equal(t, drainDeadline, budget.Begin(time.Hour))
			<-cleanupCtx.Done()
			assert.Equal(t, 5*time.Second, time.Since(startedAt))
			assert.Equal(t, time.Second, time.Until(drainDeadline))
		})
	})

	t.Run("skips cleanup after its cutoff while leaving outcome time", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			budget := New(Deps{Clock: clock.NewSystem()})
			drainDeadline := budget.Begin(6 * time.Second)
			firstCleanup, cancelFirst := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
			defer cancelFirst()
			<-firstCleanup.Done()

			lateCleanup, cancelLate := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
			defer cancelLate()
			require.Error(t, lateCleanup.Err())

			outcomeCtx, cancelOutcome := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: 30 * time.Second, Reserve: 0},
			)
			defer cancelOutcome()
			require.NoError(t, outcomeCtx.Err())
			<-outcomeCtx.Done()
			assert.Equal(t, drainDeadline, time.Now().UTC())
		})
	})

	t.Run("keeps an earlier ordinary timeout when shutdown has more time", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			budget := New(Deps{Clock: clock.NewSystem()})
			budget.Begin(6 * time.Second)
			startedAt := time.Now()
			cleanupCtx, cancelCleanup := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: time.Second, Reserve: time.Second},
			)
			defer cancelCleanup()
			<-cleanupCtx.Done()
			assert.Equal(t, time.Second, time.Since(startedAt))
		})
	})

	t.Run("releases registrations and timers when cleanup finishes", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			budget := New(Deps{Clock: clock.NewSystem()})
			budget.Begin(6 * time.Second)
			cleanupCtx, cancelCleanup := budget.NewContext(
				t.Context(),
				ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
			cancelCleanup()
			cancelCleanup()
			require.Error(t, cleanupCtx.Err())
			assert.Empty(t, budget.contexts)
		})
	})
}
