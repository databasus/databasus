package app

import (
	"context"
	"slices"
	"time"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/recoverypause"
	"databasus-backend/internal/modules/instancebackups"
)

// getLocalMaintenanceJobNames are the jobs that touch only the internal database server and the
// data volume, so they keep running while a recovered instance is paused; the recovery preparation
// is among them, so root admin can replace a paused snapshot with another. Every other job reaches a
// storage, a user database, a notifier or another service and skips its passes until root admin
// activates the instance.
func getLocalMaintenanceJobNames() []string {
	return []string{
		"audit_events_cleanup",
		"users_expired_codes_cleanup",
		"backups_download_tokens_cleanup",
		instancebackups.RecoveryPreparationJobName,
	}
}

// gateJobsOnRecoveryPause makes every job outside local maintenance skip its passes while the
// gate is closed. A skipped pass succeeds, so the job's health stays as it was.
func gateJobsOnRecoveryPause(jobs []background.Job, recoveryPause *recoverypause.Gate) []background.Job {
	gatedJobs := make([]background.Job, 0, len(jobs))

	for _, job := range jobs {
		if slices.Contains(getLocalMaintenanceJobNames(), job.Name) {
			gatedJobs = append(gatedJobs, job)

			continue
		}

		run := job.Run
		job.Run = func(ctx context.Context, now time.Time) error {
			if recoveryPause.IsPaused() {
				return nil
			}

			return run(ctx, now)
		}
		gatedJobs = append(gatedJobs, job)
	}

	return gatedJobs
}
