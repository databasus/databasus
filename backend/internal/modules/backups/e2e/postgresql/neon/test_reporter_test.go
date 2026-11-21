package neon

import "context"

// testReporter is the part of testing.TB the run's setup, cleanup and sweep use, so a harness test
// can drive them with a reporter that records a failure instead of failing itself.
type testReporter interface {
	Helper()
	Context() context.Context
	Cleanup(cleanup func())
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}
