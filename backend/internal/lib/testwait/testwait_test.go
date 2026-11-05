package testwait

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUntil(t *testing.T) {
	t.Parallel()

	t.Run("returns once the condition becomes true", func(t *testing.T) {
		t.Parallel()
		var callCount atomic.Int32

		Until(t, "the third call", func(context.Context) error {
			if callCount.Add(1) < 3 {
				return errors.New("not yet")
			}

			return nil
		})

		assert.Equal(t, int32(3), callCount.Load())
	})

	t.Run("fails with the description and the last state when the deadline nears", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{
			TB:       t,
			ctx:      t.Context(),
			deadline: time.Now().Add(500 * time.Millisecond),
		}
		var callCount atomic.Int32

		runUntil(recordingTest, "the backup to complete", func(context.Context) error {
			return fmt.Errorf("status is IN_PROGRESS after call %d", callCount.Add(1))
		})

		require.True(t, recordingTest.isFailed)
		assert.Greater(t, callCount.Load(), int32(1))
		assert.Contains(t, recordingTest.failure, "timed out waiting for the backup to complete")
		assert.Contains(t, recordingTest.failure, fmt.Sprintf("after call %d", callCount.Load()))
	})

	t.Run("fails at once when the test context is cancelled", func(t *testing.T) {
		t.Parallel()
		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()
		recordingTest := &recordingTB{TB: t, ctx: cancelledCtx, deadline: time.Now().Add(time.Hour)}
		startedAt := time.Now()

		runUntil(recordingTest, "the health endpoint", func(context.Context) error {
			return errors.New("connection refused")
		})

		require.True(t, recordingTest.isFailed)
		assert.Less(t, time.Since(startedAt), time.Second)
		assert.Contains(t, recordingTest.failure, "because the test ended")
		assert.Contains(t, recordingTest.failure, "the health endpoint")
		assert.Contains(t, recordingTest.failure, "connection refused")
	})
}

func TestWithin(t *testing.T) {
	t.Parallel()

	t.Run(
		"fails with the description and the last state when its limit passes before the deadline",
		func(t *testing.T) {
			t.Parallel()
			recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}
			startedAt := time.Now()
			finished := make(chan struct{})

			go func() {
				defer close(finished)

				Within(
					200*time.Millisecond,
				).Until(recordingTest, "the restore container to go", func(context.Context) error {
					return errors.New("1 container remains")
				})
			}()
			<-finished

			require.True(t, recordingTest.isFailed)
			assert.Less(t, time.Since(startedAt), 10*time.Second)
			assert.Contains(t, recordingTest.failure, "timed out waiting for the restore container to go")
			assert.Contains(t, recordingTest.failure, "1 container remains")
		},
	)
}

func TestWithinPoll(t *testing.T) {
	t.Parallel()

	t.Run("waits up to its limit when the limit is longer than the default", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}
		longLimit := 2 * DefaultLimit
		var waitDeadline time.Time

		err := Within(longLimit).Poll(recordingTest, "the deadline to be read", func(ctx context.Context) error {
			waitDeadline, _ = ctx.Deadline()

			return nil
		})

		require.NoError(t, err)
		assert.Greater(t, time.Until(waitDeadline), DefaultLimit)
		assert.LessOrEqual(t, time.Until(waitDeadline), longLimit)
	})

	t.Run("returns ErrTimedOut without failing the test when its limit passes", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}

		err := Within(200*time.Millisecond).Poll(recordingTest, "the instance to answer", func(context.Context) error {
			return errors.New("connection refused")
		})

		require.ErrorIs(t, err, ErrTimedOut)
		assert.Contains(t, err.Error(), "the instance to answer")
		assert.Contains(t, err.Error(), "connection refused")
		assert.False(t, recordingTest.isFailed)
	})
}

func TestDefaultLimit(t *testing.T) {
	t.Parallel()

	t.Run("bounds Until when the run's deadline is later", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}
		var (
			waitDeadline time.Time
			hasDeadline  bool
		)

		runUntil(recordingTest, "the deadline to be read", func(ctx context.Context) error {
			waitDeadline, hasDeadline = ctx.Deadline()

			return nil
		})

		require.True(t, hasDeadline)
		assertEndsAtDefaultLimit(t, waitDeadline)
		assert.Equal(t, 2*time.Minute, DefaultLimit)
	})

	t.Run("bounds Poll when the run's deadline is later", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}
		var (
			waitDeadline time.Time
			hasDeadline  bool
		)

		err := Poll(recordingTest, "the deadline to be read", func(ctx context.Context) error {
			waitDeadline, hasDeadline = ctx.Deadline()

			return nil
		})

		require.NoError(t, err)
		require.True(t, hasDeadline)
		assertEndsAtDefaultLimit(t, waitDeadline)
	})

	t.Run("bounds a wait when the test has no deadline", func(t *testing.T) {
		t.Parallel()
		testWithoutDeadline := struct{ testing.TB }{TB: t}
		var (
			waitDeadline time.Time
			hasDeadline  bool
		)

		err := Poll(testWithoutDeadline, "the deadline to be read", func(ctx context.Context) error {
			waitDeadline, hasDeadline = ctx.Deadline()

			return nil
		})

		require.NoError(t, err)
		require.True(t, hasDeadline)
		assertEndsAtDefaultLimit(t, waitDeadline)
	})
}

func assertEndsAtDefaultLimit(t *testing.T, waitDeadline time.Time) {
	t.Helper()

	assert.LessOrEqual(t, time.Until(waitDeadline), DefaultLimit)
	assert.Greater(t, time.Until(waitDeadline), DefaultLimit-time.Minute)
}

func TestWaitDeadline(t *testing.T) {
	t.Parallel()

	t.Run("ends at the default limit when the run's deadline is later", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: time.Now().Add(time.Hour)}

		waitCtx, cancelWait := withWaitDeadline(t.Context(), recordingTest, DefaultLimit)
		defer cancelWait()

		waitDeadline, hasDeadline := waitCtx.Deadline()

		require.True(t, hasDeadline)
		assert.LessOrEqual(t, time.Until(waitDeadline), 2*time.Minute)
	})

	t.Run("ends before the run's deadline when that deadline is nearer than the limit", func(t *testing.T) {
		t.Parallel()
		runDeadline := time.Now().Add(time.Minute)
		recordingTest := &recordingTB{TB: t, ctx: t.Context(), deadline: runDeadline}

		waitCtx, cancelWait := withWaitDeadline(t.Context(), recordingTest, DefaultLimit)
		defer cancelWait()

		waitDeadline, hasDeadline := waitCtx.Deadline()

		require.True(t, hasDeadline)
		assert.True(t, waitDeadline.Before(runDeadline))
	})
}

func TestPoll(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrTimedOut with the description and the last state when the deadline nears", func(t *testing.T) {
		t.Parallel()
		recordingTest := &recordingTB{
			TB:       t,
			ctx:      t.Context(),
			deadline: time.Now().Add(500 * time.Millisecond),
		}

		err := Poll(recordingTest, "the instance to answer", func(context.Context) error {
			return errors.New("connection refused")
		})

		require.ErrorIs(t, err, ErrTimedOut)
		assert.Contains(t, err.Error(), "the instance to answer")
		assert.Contains(t, err.Error(), "connection refused")
		assert.False(t, recordingTest.isFailed)
	})

	t.Run("returns ErrTestEnded when the test context is cancelled", func(t *testing.T) {
		t.Parallel()
		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()
		recordingTest := &recordingTB{TB: t, ctx: cancelledCtx, deadline: time.Now().Add(time.Hour)}

		err := Poll(recordingTest, "the instance to answer", func(context.Context) error {
			return errors.New("connection refused")
		})

		require.ErrorIs(t, err, ErrTestEnded)
		assert.False(t, recordingTest.isFailed)
	})
}

// runUntil calls Until in its own goroutine, because the recording Fatalf ends the goroutine
// like testing.T does.
func runUntil(recordingTest *recordingTB, description string, condition func(ctx context.Context) error) {
	isFinished := make(chan struct{})

	go func() {
		defer close(isFinished)

		Until(recordingTest, description, condition)
	}()

	<-isFinished
}

type recordingTB struct {
	testing.TB

	ctx      context.Context // stands in for the context testing.T holds
	deadline time.Time
	isFailed bool
	failure  string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Context() context.Context {
	return r.ctx
}

func (r *recordingTB) Deadline() (time.Time, bool) {
	return r.deadline, true
}

func (r *recordingTB) Fatal(args ...any) {
	r.isFailed = true
	r.failure = fmt.Sprint(args...)
	runtime.Goexit()
}
