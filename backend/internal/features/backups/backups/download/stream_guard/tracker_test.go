package stream_guard

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

func (s failingStore) Clear(context.Context) error {
	return s.err
}

func Test_Tracker_AcquireDownloadLock_ConcurrentCallersAcquireOneLock(t *testing.T) {
	tracker := NewTracker(cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: clock.NewSystem()}))
	userID := uuid.New()

	const callerCount = 32
	var acquiredCount atomic.Int64
	var waitGroup sync.WaitGroup
	for range callerCount {
		waitGroup.Go(func() {
			err := tracker.AcquireDownloadLock(t.Context(), userID)
			if err == nil {
				acquiredCount.Add(1)
				return
			}

			assert.ErrorIs(t, err, ErrDownloadAlreadyInProgress)
		})
	}
	waitGroup.Wait()

	assert.Equal(t, int64(1), acquiredCount.Load())
}

func Test_Tracker_WhenStoreFails_FailsClosed(t *testing.T) {
	storeError := errors.New("store failed")
	tracker := NewTracker(failingStore{err: storeError})

	err := tracker.AcquireDownloadLock(t.Context(), uuid.New())
	require.ErrorIs(t, err, storeError)
}

func Test_Tracker_ReleasedLockCanBeAcquiredAgain(t *testing.T) {
	memoryStore := cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 4096, Clock: clock.NewSystem()})
	tracker := NewTracker(memoryStore)
	userID := uuid.New()

	require.NoError(t, tracker.AcquireDownloadLock(t.Context(), userID))
	require.NoError(t, tracker.ReleaseDownloadLock(t.Context(), userID))
	require.NoError(t, tracker.AcquireDownloadLock(t.Context(), userID))
}
