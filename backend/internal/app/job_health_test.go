package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/storages"
)

//nolint:tparallel // newTestApp sets process environment variables, which a parallel test cannot do.
func TestJobHealth(t *testing.T) {
	t.Run("assigns a policy when every application job is assembled", func(t *testing.T) {
		application := newTestApp(t)
		for _, snapshot := range application.runner.GetSnapshots() {
			assert.Positive(t, application.jobHealthPolicies[snapshot.Name].MaxPassDuration)
		}
	})
	t.Run("uses explicit budgets when jobs are assembled", func(t *testing.T) {
		t.Parallel()
		policies := newJobHealthPolicies(
			[]background.Job{
				{Name: "ordinary"},
				{Name: "storage_file_deletion"},
				{Name: "healthcheck_probe"},
				{Name: "telemetry_ping"},
			},
			storages.ProductionFileTimings(),
		)

		assert.Len(t, policies, 4)
		assert.Equal(t, 5*time.Minute, policies["ordinary"].MaxPassDuration)
		assert.Equal(t, 101*time.Minute, policies["storage_file_deletion"].MaxPassDuration)
		assert.Equal(t, 30*time.Minute, policies["healthcheck_probe"].MaxPassDuration)
		assert.Equal(t, time.Minute, policies["telemetry_ping"].MaxPassDuration)
	})
	t.Run("expires at exact deadlines and restores freshness after failed completion", func(t *testing.T) {
		t.Parallel()
		activatedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		policy := JobHealthPolicy{MaxPassDuration: 5 * time.Minute}
		snapshot := background.JobSnapshot{ActivatedAt: activatedAt, Interval: time.Minute}
		assert.False(t, isJobStale(snapshot, policy, activatedAt.Add(5*time.Minute-time.Nanosecond)))
		assert.True(t, isJobStale(snapshot, policy, activatedAt.Add(5*time.Minute)))

		completedAt := activatedAt.Add(8 * time.Minute)
		snapshot.LastCompletedAt = &completedAt
		snapshot.LastOutcome = background.PassFailed
		assert.False(t, isJobStale(snapshot, policy, completedAt))
		assert.False(t, isJobStale(snapshot, policy, completedAt.Add(6*time.Minute-time.Nanosecond)))
		assert.True(t, isJobStale(snapshot, policy, completedAt.Add(6*time.Minute)))

		snapshot.IsRunning = true
		snapshot.LastStartedAt = &completedAt
		assert.False(t, isJobStale(snapshot, policy, completedAt.Add(4*time.Minute)))
		assert.True(t, isJobStale(snapshot, policy, completedAt.Add(5*time.Minute)))
	})
}
