package app

import (
	"time"

	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/storages"
)

type JobHealthPolicy struct {
	MaxPassDuration time.Duration
}

func newJobHealthPolicies(jobs []background.Job, fileTimings storages.FileTimings) map[string]JobHealthPolicy {
	policies := make(map[string]JobHealthPolicy, len(jobs))
	for _, job := range jobs {
		duration := 5 * time.Minute
		switch job.Name {
		case "telemetry_ping":
			duration = time.Minute
		case "storage_file_deletion":
			duration = max(duration, time.Duration(fileTimings.ClaimBatchSize)*fileTimings.AttemptLease+time.Minute)
		case "healthcheck_probe":
			duration = 30 * time.Minute
		}
		policies[job.Name] = JobHealthPolicy{MaxPassDuration: duration}
	}
	return policies
}

func isJobStale(snapshot background.JobSnapshot, policy JobHealthPolicy, now time.Time) bool {
	deadline := snapshot.ActivatedAt.Add(policy.MaxPassDuration)
	if snapshot.LastCompletedAt != nil {
		deadline = snapshot.LastCompletedAt.Add(snapshot.Interval + policy.MaxPassDuration)
	}
	if !now.Before(deadline) {
		return true
	}
	return snapshot.IsRunning && snapshot.LastStartedAt != nil &&
		!now.Before(snapshot.LastStartedAt.Add(policy.MaxPassDuration))
}

func (a *App) getJobHealth(now time.Time) []systemhttpapi.JobHealth {
	snapshots := a.runner.GetSnapshots()
	jobs := make([]systemhttpapi.JobHealth, 0, len(snapshots))
	for _, snapshot := range snapshots {
		policy := a.jobHealthPolicies[snapshot.Name]
		var outcome *systemhttpapi.JobHealthLastOutcome
		if snapshot.LastCompletedAt != nil {
			outcome = new(systemhttpapi.JobHealthLastOutcome(snapshot.LastOutcome))
		}
		jobs = append(jobs, systemhttpapi.JobHealth{
			Name:                   snapshot.Name,
			IntervalSeconds:        snapshot.Interval.Seconds(),
			MaxPassDurationSeconds: policy.MaxPassDuration.Seconds(),
			IsRunning:              snapshot.IsRunning,
			IsStale:                isJobStale(snapshot, policy, now),
			ActivatedAt:            snapshot.ActivatedAt,
			LastStartedAt:          snapshot.LastStartedAt,
			LastCompletedAt:        snapshot.LastCompletedAt,
			LastOutcome:            outcome,
		})
	}
	return jobs
}
