package download

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
)

func TestUserStreams(t *testing.T) {
	t.Parallel()

	t.Run("gives the stream to exactly one of many concurrent callers", func(t *testing.T) {
		t.Parallel()
		userStreams := NewUserStreams(newMemoryStore())
		userID := uuid.New()

		const callerCount = 32

		var acquiredCount atomic.Int64

		var waitGroup sync.WaitGroup
		for range callerCount {
			waitGroup.Go(func() {
				_, err := userStreams.Acquire(t.Context(), userID)
				if err == nil {
					acquiredCount.Add(1)

					return
				}

				assert.ErrorIs(t, err, ErrUserStreamHeld)
			})
		}
		waitGroup.Wait()

		assert.Equal(t, int64(1), acquiredCount.Load())
	})

	t.Run("gives a released stream to the next caller", func(t *testing.T) {
		t.Parallel()
		userStreams := NewUserStreams(newMemoryStore())
		userID := uuid.New()

		userStream, err := userStreams.Acquire(t.Context(), userID)
		require.NoError(t, err)
		require.NoError(t, userStream.Release(t.Context()))

		_, err = userStreams.Acquire(t.Context(), userID)
		require.NoError(t, err)
	})

	t.Run("reports a held stream while it runs and a free one after its release", func(t *testing.T) {
		t.Parallel()
		userStreams := NewUserStreams(newMemoryStore())
		userID := uuid.New()

		userStream, err := userStreams.Acquire(t.Context(), userID)
		require.NoError(t, err)

		isHeldWhileRunning, err := userStreams.IsHeld(t.Context(), userID)
		require.NoError(t, err)
		require.NoError(t, userStream.Release(t.Context()))
		isHeldAfterRelease, err := userStreams.IsHeld(t.Context(), userID)
		require.NoError(t, err)

		assert.True(t, isHeldWhileRunning)
		assert.False(t, isHeldAfterRelease)
	})

	t.Run("refuses the stream when the cache fails", func(t *testing.T) {
		t.Parallel()
		errStoreFailed := errors.New("store failed")
		userStreams := NewUserStreams(failingStore{err: errStoreFailed})

		_, acquireErr := userStreams.Acquire(t.Context(), uuid.New())
		isHeld, heldErr := userStreams.IsHeld(t.Context(), uuid.New())

		require.ErrorIs(t, acquireErr, errStoreFailed)
		require.ErrorIs(t, heldErr, errStoreFailed)
		assert.True(t, isHeld)
	})
}

func newMemoryStore() *cache.MemoryStore {
	return cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: clock.NewSystem()})
}

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
