package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolSaturated(t *testing.T) {
	t.Run("reflects slot occupancy as jobs start and finish", func(t *testing.T) {
		group := newTestGroup(t)
		pool := NewPool(2)
		assert.False(t, pool.IsSaturated())

		release := make(chan struct{})
		var started sync.WaitGroup
		started.Add(2)

		for range 2 {
			pool.Go(t.Context(), group, func(context.Context) error {
				started.Done()
				<-release
				return nil
			})
		}

		started.Wait()
		assert.True(t, pool.IsSaturated(), "pool must report saturated at MaxConcurrentJobs")

		close(release)
		pool.Wait()
		assert.False(t, pool.IsSaturated(), "slots must be released when jobs finish")
	})
}

func TestPool(t *testing.T) {
	t.Run("never exceeds cap and wait drains all when many jobs are submitted", func(t *testing.T) {
		const capLimit = 4
		group := newTestGroup(t)
		pool := NewPool(capLimit)

		var concurrent, maxConcurrent atomic.Int32

		for range 100 {
			pool.Go(t.Context(), group, func(context.Context) error {
				cur := concurrent.Add(1)
				for {
					observed := maxConcurrent.Load()
					if cur <= observed || maxConcurrent.CompareAndSwap(observed, cur) {
						break
					}
				}

				concurrent.Add(-1)
				return nil
			})
		}

		pool.Wait()

		assert.LessOrEqual(t, int(maxConcurrent.Load()), capLimit,
			"concurrency must never exceed the slot count")
		assert.Equal(t, int32(0), concurrent.Load(), "Wait must drain every job")
	})
}
