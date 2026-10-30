package apptest

import "testing"

func (a *App) PausePhysicalJobs(t *testing.T) {
	t.Helper()
	if err := a.stopPhysicalWork(t.Context()); err != nil {
		t.Fatalf("pause physical jobs: %v", err)
	}
}

func (a *App) ResumePhysicalJobs(t *testing.T) {
	t.Helper()
	a.startPhysicalLoops(t)
}
