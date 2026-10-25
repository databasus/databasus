package background

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/testwait"
)

func TestRunner(t *testing.T) {
	t.Parallel()

	t.Run("adds job_name and a job_id of its own to the records of every pass", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		logger := logging.NewWithSinks(capturer).GetLogger()
		runner := NewRunner(newRunnerDeps(logger), Job{
			Name:     "audit_events_cleanup",
			Interval: time.Hour,
			Run: func(ctx context.Context, _ time.Time) error {
				logger.InfoContext(ctx, "expired events deleted")

				return nil
			},
		})

		require.NoError(t, runner.RunPass(t.Context(), "audit_events_cleanup"))
		require.NoError(t, runner.RunPass(t.Context(), "audit_events_cleanup"))

		passRecords := capturer.getRecordsWithMessage("expired events deleted")
		require.Len(t, passRecords, 2)
		assert.Equal(t, "audit_events_cleanup", passRecords[0]["job_name"])
		assert.NotEmpty(t, passRecords[0]["job_id"])
		assert.NotEqual(t, passRecords[0]["job_id"], passRecords[1]["job_id"])
	})

	t.Run("logs the error and runs the next pass when a pass fails", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		var passCount atomic.Int64
		runner := NewRunner(newRunnerDeps(logging.NewWithSinks(capturer).GetLogger()), Job{
			Name:     "failing",
			Interval: time.Millisecond,
			Run: func(context.Context, time.Time) error {
				passCount.Add(1)

				return errors.New("disk is full")
			},
		})

		startUntilTestEnds(t, runner)

		waitForPassCount(t, &passCount, 2)
		failureRecords := capturer.getRecordsWithMessage("background job pass failed")
		require.NotEmpty(t, failureRecords)
		assert.Equal(t, "disk is full", failureRecords[0]["error"])
		assert.Equal(t, "failing", failureRecords[0]["job_name"])
	})

	t.Run("logs the panic with its stack and runs the next pass when a pass panics", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		var passCount atomic.Int64
		runner := NewRunner(newRunnerDeps(logging.NewWithSinks(capturer).GetLogger()), Job{
			Name:     "exploding",
			Interval: time.Millisecond,
			Run: func(context.Context, time.Time) error {
				passCount.Add(1)
				panic("boom")
			},
		})

		startUntilTestEnds(t, runner)

		waitForPassCount(t, &passCount, 2)
		panicRecords := capturer.getRecordsWithMessage("background job pass panicked")
		require.NotEmpty(t, panicRecords)
		assert.Contains(t, panicRecords[0]["stack"], "runner_test.go")
	})

	t.Run("logs at info when the context ends during a pass", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		runner := NewRunner(newRunnerDeps(logging.NewWithSinks(capturer).GetLogger()), Job{
			Name:     "interrupted",
			Interval: time.Hour,
			Run: func(ctx context.Context, _ time.Time) error {
				<-ctx.Done()

				return fmt.Errorf("upload part: %w", ctx.Err())
			},
		})
		passContext, cancelPass := context.WithCancel(t.Context())
		cancelPass()

		err := runner.RunPass(passContext, "interrupted")

		require.ErrorIs(t, err, context.Canceled)
		assert.Len(t, capturer.getRecordsWithMessage("background job pass interrupted"), 1)
		assert.Empty(t, capturer.getRecordsWithMessage("background job pass failed"))
	})

	t.Run("never overlaps two passes of one job when they are run concurrently", func(t *testing.T) {
		t.Parallel()
		var runningCount, overlapCount, passCount atomic.Int64
		runner := NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), Job{
			Name:     "exclusive",
			Interval: time.Millisecond,
			Run: func(context.Context, time.Time) error {
				if runningCount.Add(1) > 1 {
					overlapCount.Add(1)
				}

				runtime.Gosched()
				runningCount.Add(-1)
				passCount.Add(1)

				return nil
			},
		})

		startUntilTestEnds(t, runner)

		var waitGroup sync.WaitGroup
		for range 20 {
			waitGroup.Go(func() {
				assert.NoError(t, runner.RunPass(t.Context(), "exclusive"))
			})
		}
		waitGroup.Wait()

		waitForPassCount(t, &passCount, 22)
		assert.Zero(t, overlapCount.Load())
	})

	t.Run("panics when started a second time", func(t *testing.T) {
		t.Parallel()
		runner := NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), newIdleJob("idle"))

		startUntilTestEnds(t, runner)

		assert.Panics(t, func() { runner.Start(t.Context()) })
	})

	t.Run("panics when two jobs share a name", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), newIdleJob("idle"), newIdleJob("idle"))
		})
	})

	t.Run("returns the pass error and passes the clock's time when a pass is run", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
		manualClock.Advance(366 * 24 * time.Hour)
		var passTime time.Time
		passError := errors.New("database is unavailable")
		runner := NewRunner(
			RunnerDeps{Logger: slog.New(slog.DiscardHandler), Clock: manualClock},
			Job{
				Name:     "audit_events_cleanup",
				Interval: time.Hour,
				Run: func(_ context.Context, now time.Time) error {
					passTime = now

					return passError
				},
			},
		)

		err := runner.RunPass(t.Context(), "audit_events_cleanup")

		require.ErrorIs(t, err, passError)
		assert.Equal(t, manualClock.Now(), passTime)
	})

	t.Run("returns ErrUnknownJob when no job has the name", func(t *testing.T) {
		t.Parallel()
		runner := NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), newIdleJob("idle"))

		err := runner.RunPass(t.Context(), "missing")

		require.ErrorIs(t, err, ErrUnknownJob)
	})

	t.Run("returns from wait after the context ends", func(t *testing.T) {
		t.Parallel()
		var passCount atomic.Int64
		runner := NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), Job{
			Name:     "hourly",
			Interval: time.Hour,
			Run: func(context.Context, time.Time) error {
				passCount.Add(1)

				return nil
			},
		})
		loopContext, stopLoops := context.WithCancel(t.Context())
		runner.Start(loopContext)
		waitForPassCount(t, &passCount, 1)

		stopLoops()

		require.NoError(t, runner.Wait(t.Context()))
		assert.Equal(t, int64(1), passCount.Load())
	})
}

func newRunnerDeps(logger *slog.Logger) RunnerDeps {
	return RunnerDeps{Logger: logger, Clock: clock.NewSystem()}
}

func newIdleJob(name string) Job {
	return Job{
		Name:     name,
		Interval: time.Hour,
		Run:      func(context.Context, time.Time) error { return nil },
	}
}

func startUntilTestEnds(t *testing.T, runner *Runner) {
	t.Helper()

	loopContext, stopLoops := context.WithCancel(t.Context())
	runner.Start(loopContext)

	t.Cleanup(func() {
		stopLoops()
		assert.NoError(t, runner.Wait(context.WithoutCancel(t.Context())))
	})
}

func waitForPassCount(t *testing.T, passCount *atomic.Int64, minimumCount int64) {
	t.Helper()

	testwait.Until(t, fmt.Sprintf("%d passes", minimumCount), func(context.Context) error {
		if passCount.Load() < minimumCount {
			return fmt.Errorf("%d passes ran", passCount.Load())
		}

		return nil
	})
}

type recordCapturer struct {
	mutex   sync.Mutex
	records []slog.Record
}

func (c *recordCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (c *recordCapturer) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *recordCapturer) WithGroup(string) slog.Handler { return c }

func (c *recordCapturer) Handle(_ context.Context, record slog.Record) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.records = append(c.records, record.Clone())

	return nil
}

func (c *recordCapturer) getRecordsWithMessage(message string) []map[string]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var matchingRecords []map[string]string

	for _, record := range c.records {
		if record.Message != message {
			continue
		}

		attributes := map[string]string{}
		record.Attrs(func(attr slog.Attr) bool {
			attributes[attr.Key] = attr.Value.String()

			return true
		})

		matchingRecords = append(matchingRecords, attributes)
	}

	return matchingRecords
}
