package background

import (
	"cmp"
	"slices"
	"time"
)

type PassOutcome string

const (
	PassSucceeded PassOutcome = "SUCCEEDED"
	PassFailed    PassOutcome = "FAILED"
	PassPanicked  PassOutcome = "PANICKED"
	PassCanceled  PassOutcome = "CANCELED"
)

type JobSnapshot struct {
	Name            string
	Interval        time.Duration
	ActivatedAt     time.Time
	IsRunning       bool
	LastStartedAt   *time.Time
	LastCompletedAt *time.Time
	LastOutcome     PassOutcome
	LastError       error
}

func (r *Runner) GetSnapshots() []JobSnapshot {
	snapshots := make([]JobSnapshot, 0, len(r.jobs))
	for _, job := range r.jobs {
		job.snapshotMutex.RLock()
		snapshot := job.snapshot
		if snapshot.LastStartedAt != nil {
			startedAt := *snapshot.LastStartedAt
			snapshot.LastStartedAt = &startedAt
		}
		if snapshot.LastCompletedAt != nil {
			completedAt := *snapshot.LastCompletedAt
			snapshot.LastCompletedAt = &completedAt
		}
		job.snapshotMutex.RUnlock()

		snapshots = append(snapshots, snapshot)
	}
	slices.SortFunc(snapshots, func(first, second JobSnapshot) int {
		return cmp.Compare(first.Name, second.Name)
	})

	return snapshots
}
