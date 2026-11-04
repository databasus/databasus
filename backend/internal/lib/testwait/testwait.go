// Package testwait lets a test wait for a condition instead of sleeping for a fixed time.
package testwait

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	// DefaultLimit bounds one wait, so a hung condition fails its test in minutes instead of taking
	// the whole run's timeout. The longest ordinary wait takes about 90 seconds on a loaded machine.
	DefaultLimit = 2 * time.Minute

	firstInterval    = 10 * time.Millisecond
	longestInterval  = time.Second
	longestReportGap = 30 * time.Second
)

// deadlineTB is implemented by *testing.T, which reports the deadline of the whole run.
type deadlineTB interface {
	Deadline() (time.Time, bool)
}

// Until polls condition with a growing interval until it returns nil, DefaultLimit passes, the
// test's deadline nears or the test's context ends. On timeout it fails the test with description
// and the error the last call returned. It stops a little before the deadline, so the test reports
// what it waited for instead of the run panicking on its timeout.
func Until(tb testing.TB, description string, condition func(ctx context.Context) error) {
	tb.Helper()

	Within(DefaultLimit).Until(tb, description, condition)
}

// Waiter waits with its own limit, for the rare test that observes a long period of real time and for
// a start-up wait that may legitimately take longer than DefaultLimit.
type Waiter struct {
	limit time.Duration
}

func Within(limit time.Duration) Waiter {
	return Waiter{limit: limit}
}

// Until waits like the package's Until, up to the Waiter's limit.
func (w Waiter) Until(tb testing.TB, description string, condition func(ctx context.Context) error) {
	tb.Helper()

	err := w.Poll(tb, description, condition)
	if err != nil {
		tb.Fatal(err.Error())
	}
}

// Poll waits like Until but returns ErrTimedOut or ErrTestEnded, with description and the last
// state, instead of failing the test. A caller that has to explain the timeout, such as with the
// log of the container it waited for, adds that before failing.
func Poll(tb testing.TB, description string, condition func(ctx context.Context) error) error {
	tb.Helper()

	return Within(DefaultLimit).Poll(tb, description, condition)
}

// Poll waits like the package's Poll, up to the Waiter's limit.
func (w Waiter) Poll(tb testing.TB, description string, condition func(ctx context.Context) error) error {
	tb.Helper()

	ctx := tb.Context()

	waitCtx, cancelWait := withWaitDeadline(ctx, tb, w.limit)
	defer cancelWait()

	interval := firstInterval

	for {
		lastErr := condition(waitCtx)
		if lastErr == nil {
			return nil
		}

		timer := time.NewTimer(interval)

		select {
		case <-timer.C:
		case <-waitCtx.Done():
			timer.Stop()

			if ctx.Err() != nil {
				return fmt.Errorf("%w for %s, last state: %w", ErrTestEnded, description, lastErr)
			}

			return fmt.Errorf("%w for %s, last state: %w", ErrTimedOut, description, lastErr)
		}

		interval = min(interval*2, longestInterval)
	}
}

func withWaitDeadline(ctx context.Context, tb testing.TB, limit time.Duration) (context.Context, context.CancelFunc) {
	tb.Helper()

	waitDeadline := time.Now().Add(limit)

	testWithDeadline, canReportDeadline := tb.(deadlineTB)
	if !canReportDeadline {
		return context.WithDeadline(ctx, waitDeadline)
	}

	testDeadline, hasDeadline := testWithDeadline.Deadline()
	if !hasDeadline {
		return context.WithDeadline(ctx, waitDeadline)
	}

	reportGap := min(time.Until(testDeadline)/10, longestReportGap)
	reportDeadline := testDeadline.Add(-reportGap)

	if reportDeadline.Before(waitDeadline) {
		return context.WithDeadline(ctx, reportDeadline)
	}

	return context.WithDeadline(ctx, waitDeadline)
}
