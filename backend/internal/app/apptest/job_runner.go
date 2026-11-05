package apptest

import (
	"context"
	"testing"

	"databasus-backend/internal/lib/background"
)

func (a *App) StartJobRunner(tb testing.TB, names []string) *background.Runner {
	tb.Helper()

	jobsCtx, cancelJobs := context.WithCancel(context.WithoutCancel(tb.Context()))
	runner := a.app.NewJobRunner(names)
	runner.Start(jobsCtx)

	tb.Cleanup(func() {
		cancelJobs()
		if err := runner.Wait(context.WithoutCancel(tb.Context())); err != nil {
			tb.Errorf("stop background jobs: %v", err)
		}
	})

	return runner
}
