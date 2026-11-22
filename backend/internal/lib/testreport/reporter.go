// Package testreport lets a test drive its setup, cleanup and sweep helpers with a reporter that
// records failures instead of failing, so a harness test can check what a failing run reports.
package testreport

import "context"

// Reporter is the part of testing.TB that setup, cleanup and sweep helpers use.
type Reporter interface {
	Helper()
	Context() context.Context
	Cleanup(cleanup func())
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}
