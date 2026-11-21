package neon

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"
)

// recordingReporter records failures instead of failing, so a harness test can check what a
// failing run reports and what its cleanup leaves behind.
type recordingReporter struct {
	t        *testing.T
	mu       sync.Mutex
	cleanups []func()
	failures []string
}

func newRecordingReporter(t *testing.T) *recordingReporter {
	t.Helper()

	return &recordingReporter{t: t, mu: sync.Mutex{}, cleanups: nil, failures: nil}
}

func (r *recordingReporter) Helper() {}

func (r *recordingReporter) Context() context.Context {
	return r.t.Context()
}

func (r *recordingReporter) Cleanup(cleanup func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanups = append(r.cleanups, cleanup)
}

func (r *recordingReporter) Logf(format string, args ...any) {
	r.t.Logf(format, args...)
}

func (r *recordingReporter) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recordingReporter) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtime.Goexit()
}

// runWithCleanups runs the body the way a test runs, then its cleanups in reverse order, each in a
// goroutine of its own because Fatalf ends the goroutine it is called from.
func (r *recordingReporter) runWithCleanups(body func(reporter testReporter)) {
	runToCompletion(func() { body(r) })

	r.mu.Lock()
	cleanups := slices.Clone(r.cleanups)
	r.mu.Unlock()

	for _, cleanup := range slices.Backward(cleanups) {
		runToCompletion(cleanup)
	}
}

func (r *recordingReporter) getFailures() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.failures)
}

func runToCompletion(work func()) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		work()
	}()

	<-done
}
