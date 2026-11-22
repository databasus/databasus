package testreport

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"
)

// Recorder records failures and log lines instead of failing, so a harness test can check what a
// failing run reports and what its cleanup leaves behind. Log lines also reach the test's own log.
type Recorder struct {
	t        *testing.T
	mutex    sync.Mutex
	cleanups []func()
	failures []string
	logLines []string
}

func NewRecorder(t *testing.T) *Recorder {
	t.Helper()

	return &Recorder{t: t, mutex: sync.Mutex{}, cleanups: nil, failures: nil, logLines: nil}
}

func (r *Recorder) Helper() {}

func (r *Recorder) Context() context.Context {
	return r.t.Context()
}

func (r *Recorder) Cleanup(cleanup func()) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cleanups = append(r.cleanups, cleanup)
}

func (r *Recorder) Logf(format string, args ...any) {
	r.mutex.Lock()
	r.logLines = append(r.logLines, fmt.Sprintf(format, args...))
	r.mutex.Unlock()

	r.t.Logf(format, args...)
}

func (r *Recorder) Errorf(format string, args ...any) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *Recorder) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtime.Goexit()
}

// RunWithCleanups runs the body the way a test runs, then its cleanups in reverse order, each in a
// goroutine of its own because Fatalf ends the goroutine it is called from.
func (r *Recorder) RunWithCleanups(body func(reporter Reporter)) {
	runToCompletion(func() { body(r) })

	r.mutex.Lock()
	cleanups := slices.Clone(r.cleanups)
	r.mutex.Unlock()

	for _, cleanup := range slices.Backward(cleanups) {
		runToCompletion(cleanup)
	}
}

func (r *Recorder) GetFailures() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return slices.Clone(r.failures)
}

func (r *Recorder) GetLogLines() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return slices.Clone(r.logLines)
}

func runToCompletion(work func()) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		work()
	}()

	<-done
}
