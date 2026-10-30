package shutdown

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testFlushTimeout = 2 * time.Second

func TestDrain(t *testing.T) {
	t.Parallel()

	t.Run("finishes at once when every step returns", func(t *testing.T) {
		t.Parallel()
		var flushCount atomic.Int32

		drainStartedAt := time.Now()
		Drain(t.Context(), newTestDrainSpec(&flushCount, 6*time.Second, func(context.Context) error { return nil }))

		assert.Less(t, time.Since(drainStartedAt), time.Second)
		assert.Equal(t, int32(1), flushCount.Load())
	})

	t.Run("finishes at the drain limit when a step ignores its context", func(t *testing.T) {
		t.Parallel()
		const drainTimeout = 300 * time.Millisecond
		var flushCount atomic.Int32
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })

		drainStartedAt := time.Now()
		Drain(t.Context(), newTestDrainSpec(&flushCount, drainTimeout, func(context.Context) error {
			<-release

			return nil
		}))

		drainDuration := time.Since(drainStartedAt)
		assert.GreaterOrEqual(t, drainDuration, drainTimeout)
		assert.Less(t, drainDuration, drainTimeout+time.Second)
		assert.Equal(t, int32(1), flushCount.Load())
	})

	t.Run("drains after its context is cancelled by the stop signal", func(t *testing.T) {
		t.Parallel()
		var flushCount atomic.Int32
		var isStepContextAlive atomic.Bool
		stoppedCtx, stop := context.WithCancel(t.Context())
		stop()

		Drain(stoppedCtx, newTestDrainSpec(&flushCount, time.Second, func(stepCtx context.Context) error {
			isStepContextAlive.Store(stepCtx.Err() == nil)

			return errors.New("server forced to shut down")
		}))

		assert.True(t, isStepContextAlive.Load())
		assert.Equal(t, int32(1), flushCount.Load())
	})
}

func newTestDrainSpec(flushCount *atomic.Int32, drainTimeout time.Duration, run func(context.Context) error) DrainSpec {
	return DrainSpec{
		Steps:    []Step{{Name: "http server", Run: run}},
		Deadline: time.Now().Add(drainTimeout),
		FlushLogs: func(context.Context) error {
			flushCount.Add(1)

			return nil
		},
		FlushTimeout: testFlushTimeout,
		Logger:       slog.New(slog.DiscardHandler),
	}
}
