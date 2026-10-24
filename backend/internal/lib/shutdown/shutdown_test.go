package shutdown

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// The test sends real signals to its own process, so it does not run in parallel with others.
func TestListenForStopSignal(t *testing.T) {
	t.Run("ends the context on the first signal and ignores the second", func(t *testing.T) {
		stopSignalCtx, stopListening := ListenForStopSignal(t.Context())
		defer stopListening()

		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

		select {
		case <-stopSignalCtx.Done():
		case <-time.After(3 * time.Second):
			t.Fatal("the context did not end after the stop signal")
		}

		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

		assert.ErrorIs(t, stopSignalCtx.Err(), context.Canceled)
	})
}

func newTestDrainSpec(flushCount *atomic.Int32, drainTimeout time.Duration, run func(context.Context) error) DrainSpec {
	return DrainSpec{
		Steps:        []Step{{Name: "http server", Run: run}},
		DrainTimeout: drainTimeout,
		FlushLogs: func(context.Context) error {
			flushCount.Add(1)

			return nil
		},
		FlushTimeout: testFlushTimeout,
		Logger:       slog.New(slog.DiscardHandler),
	}
}
