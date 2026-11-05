package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
)

func TestMemoryStore(t *testing.T) {
	t.Parallel()

	t.Run("returns a copy of the payload when the caller changes the read or written bytes", func(t *testing.T) {
		t.Parallel()
		store := newTestMemoryStore(128, clock.NewManual(getTestStart()))

		writtenPayload := []byte("value")
		require.NoError(t, store.Set(t.Context(), Entry{Key: "alpha", Payload: writtenPayload, Lifetime: time.Minute}))
		writtenPayload[0] = 'X'

		readPayload, isFound, err := store.Get(t.Context(), "alpha")
		require.NoError(t, err)
		require.True(t, isFound)
		readPayload[0] = 'Y'

		storedPayload, isFound, err := store.Get(t.Context(), "alpha")
		require.NoError(t, err)
		require.True(t, isFound)
		assert.Equal(t, []byte("value"), storedPayload)
	})

	t.Run("forgets an entry when its lifetime has passed", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(getTestStart())
		store := newTestMemoryStore(128, manualClock)
		require.NoError(t, store.Set(t.Context(), Entry{Key: "alpha", Payload: []byte("value"), Lifetime: time.Minute}))

		manualClock.Advance(time.Minute)

		_, isFound, err := store.Get(t.Context(), "alpha")
		require.NoError(t, err)
		assert.False(t, isFound)
	})

	t.Run("changes no entry when the context is canceled", func(t *testing.T) {
		t.Parallel()
		store := newTestMemoryStore(128, clock.NewManual(getTestStart()))
		require.NoError(
			t,
			store.Set(t.Context(), Entry{Key: "existing", Payload: []byte("value"), Lifetime: time.Minute}),
		)
		canceledContext, cancel := context.WithCancel(t.Context())
		cancel()

		setError := store.Set(canceledContext, Entry{Key: "new", Payload: []byte("value"), Lifetime: time.Minute})
		deleteError := store.Delete(canceledContext, "existing")

		require.ErrorIs(t, setError, context.Canceled)
		require.ErrorIs(t, deleteError, context.Canceled)

		_, isNewFound, err := store.Get(t.Context(), "new")
		require.NoError(t, err)
		assert.False(t, isNewFound)

		_, isExistingFound, err := store.Get(t.Context(), "existing")
		require.NoError(t, err)
		assert.True(t, isExistingFound)
	})

	t.Run("refuses a write without evicting live entries when the budget is reached", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(getTestStart())
		store := newTestMemoryStore(10, manualClock)
		require.NoError(t, store.Set(t.Context(), Entry{Key: "a", Payload: []byte("1234"), Lifetime: time.Minute}))
		require.NoError(t, store.Set(t.Context(), Entry{Key: "b", Payload: []byte("1234"), Lifetime: time.Minute}))

		err := store.Set(t.Context(), Entry{Key: "c", Payload: []byte("1"), Lifetime: time.Minute})

		require.ErrorIs(t, err, ErrPayloadBudgetReached)

		_, isFirstFound, err := store.Get(t.Context(), "a")
		require.NoError(t, err)
		assert.True(t, isFirstFound)

		_, isSecondFound, err := store.Get(t.Context(), "b")
		require.NoError(t, err)
		assert.True(t, isSecondFound)
	})

	t.Run("reclaims expired entries when a write would exceed the budget", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(getTestStart())
		store := newTestMemoryStore(10, manualClock)
		require.NoError(t, store.Set(t.Context(), Entry{Key: "a", Payload: []byte("1234"), Lifetime: time.Minute}))
		require.NoError(t, store.Set(t.Context(), Entry{Key: "b", Payload: []byte("1234"), Lifetime: time.Minute}))
		manualClock.Advance(time.Minute)

		err := store.Set(t.Context(), Entry{Key: "r", Payload: []byte("x"), Lifetime: time.Minute})

		require.NoError(t, err)
	})

	t.Run("keeps unrelated expired entries when a write fits the budget", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(getTestStart())
		store := newTestMemoryStore(128, manualClock)
		require.NoError(
			t,
			store.Set(t.Context(), Entry{Key: "expired", Payload: []byte("value"), Lifetime: time.Minute}),
		)
		manualClock.Advance(time.Minute)

		require.NoError(
			t,
			store.Set(t.Context(), Entry{Key: "current", Payload: []byte("value"), Lifetime: time.Minute}),
		)

		assert.Contains(t, store.entries, "expired")
	})

	t.Run("returns an error when the lifetime is not positive", func(t *testing.T) {
		t.Parallel()
		store := newTestMemoryStore(128, clock.NewManual(getTestStart()))

		err := store.Set(t.Context(), Entry{Key: "invalid", Payload: []byte("value"), Lifetime: 0})

		require.ErrorIs(t, err, ErrInvalidLifetime)
	})
}

func getTestStart() time.Time { return time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC) }

func newTestMemoryStore(payloadBudgetBytes int64, clockSource clock.Source) *MemoryStore {
	return NewMemoryStore(Deps{PayloadBudgetBytes: payloadBudgetBytes, Clock: clockSource})
}
