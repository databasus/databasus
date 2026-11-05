package ratelimit

import (
	"context"
	"strconv"
	"sync"
	"time"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
)

const cacheNamespace = "rate_limit_attempts"

type Deps struct {
	Store cache.Store
	Clock clock.Source
}

// MemoryCounter keeps a sliding window of attempt times per scope and identifier. A refused
// attempt is counted too, so a caller that keeps trying stays refused until it pauses for a window.
type MemoryCounter struct {
	mutex    sync.Mutex
	attempts *cache.JSONStore[[]time.Time]
	clock    clock.Source
}

func NewMemoryCounter(deps Deps) *MemoryCounter {
	return &MemoryCounter{
		mutex:    sync.Mutex{},
		attempts: cache.NewJSONStore[[]time.Time](deps.Store, cacheNamespace),
		clock:    deps.Clock,
	}
}

func (c *MemoryCounter) RecordAttemptAndCheckIsAllowed(ctx context.Context, attempt Attempt) (bool, error) {
	if attempt.Limit <= 0 || attempt.Window <= 0 {
		return false, ErrInvalidSettings
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := c.clock.Now()

	storedAttempts, err := c.attempts.Get(ctx, getCounterKey(attempt))
	if err != nil {
		return false, err
	}

	windowStart := now.Add(-attempt.Window)
	recentAttempts := make([]time.Time, 0, attempt.Limit+1)

	if storedAttempts != nil {
		for _, recordedAt := range *storedAttempts {
			if recordedAt.After(windowStart) {
				recentAttempts = append(recentAttempts, recordedAt)
			}
		}
	}

	recentAttempts = append(recentAttempts, now)
	if len(recentAttempts) > attempt.Limit+1 {
		recentAttempts = recentAttempts[len(recentAttempts)-(attempt.Limit+1):]
	}

	err = c.attempts.SetWithLifetime(ctx, cache.ExpiringValue[[]time.Time]{
		Key:      getCounterKey(attempt),
		Value:    recentAttempts,
		Lifetime: attempt.Window,
	})
	if err != nil {
		return false, err
	}

	return len(recentAttempts) <= attempt.Limit, nil
}

func getCounterKey(attempt Attempt) string {
	return strconv.Itoa(len(attempt.Scope)) + ":" + attempt.Scope + ":" + attempt.Identifier
}
