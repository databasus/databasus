package verifications

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/backups"
)

func newPlannedBackup(archiveMb, databaseMb float64, largestIndexSizesMb pq.Float64Array) backups.LogicalBackup {
	return backups.LogicalBackup{
		BackupSizeMb:              archiveMb,
		BackupRawDbSizeMb:         databaseMb,
		BackupLargestIndexSizesMb: largestIndexSizesMb,
	}
}

func TestPlanRestore(t *testing.T) {
	t.Parallel()

	indexedBackup := newPlannedBackup(10000, 20000, pq.Float64Array{3000, 2000, 1000, 500, 100})
	timescaleBackup := indexedBackup
	timescaleBackup.TimescaledbVersion = "2.17.0"

	cases := []struct {
		name           string
		capacity       AgentCapacity
		reservedDiskMb int64
		backup         backups.LogicalBackup
		expectedPlan   RestorePlan
		isFit          bool
	}{
		{
			name:         "restores with every job of the per-job CPU budget when the disk holds them",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 4, DiskBudgetMb: 38036},
			isFit:        true,
		},
		{
			name:         "restores with fewer jobs when the disk holds fewer index builds",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 36},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 2, DiskBudgetMb: 36536},
			isFit:        true,
		},
		{
			name:     "does not fit when the disk holds not even one job",
			capacity: AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 33},
			backup:   indexedBackup,
			isFit:    false,
		},
		{
			name:         "restores with the share of CPU each concurrent job gets",
			capacity:     AgentCapacity{MaxCPU: 8, MaxConcurrentJobs: 2, MaxDiskGb: 100},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 4, DiskBudgetMb: 38036},
			isFit:        true,
		},
		{
			name:         "restores with one job when there are fewer CPUs than concurrent jobs",
			capacity:     AgentCapacity{MaxCPU: 1, MaxConcurrentJobs: 2, MaxDiskGb: 100},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 34536},
			isFit:        true,
		},
		{
			name:         "restores with one job when the agent reports zero concurrent jobs",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 0, MaxDiskGb: 100},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 34536},
			isFit:        true,
		},
		{
			name:         "restores with at most eight jobs and counts every index when fewer were recorded",
			capacity:     AgentCapacity{MaxCPU: 16, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       indexedBackup,
			expectedPlan: RestorePlan{ParallelJobs: 8, DiskBudgetMb: 38136},
			isFit:        true,
		},
		{
			name:         "restores a timescaledb backup with one job",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       timescaleBackup,
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 34536},
			isFit:        true,
		},
		{
			name:         "restores a backup without recorded index sizes with one job and the clamped gap",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       newPlannedBackup(10000, 20000, nil),
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 35120},
			isFit:        true,
		},
		{
			name:         "uses the gap floor for a small database without recorded index sizes",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       newPlannedBackup(100, 2000, nil),
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 3636},
			isFit:        true,
		},
		{
			name:         "restores a backup of an unknown database size with one job and the gap cap",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       newPlannedBackup(10000, 0, pq.Float64Array{3000, 2000}),
			expectedPlan: RestorePlan{ParallelJobs: 1, DiskBudgetMb: 15120},
			isFit:        true,
		},
		{
			name:         "restores a backup with no indexes with every job",
			capacity:     AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 100},
			backup:       newPlannedBackup(10000, 20000, pq.Float64Array{}),
			expectedPlan: RestorePlan{ParallelJobs: 4, DiskBudgetMb: 31536},
			isFit:        true,
		},
		{
			name:           "restores with the jobs the disk left by a running verification holds",
			capacity:       AgentCapacity{MaxCPU: 8, MaxConcurrentJobs: 2, MaxDiskGb: 100},
			reservedDiskMb: 65000,
			backup:         indexedBackup,
			expectedPlan:   RestorePlan{ParallelJobs: 2, DiskBudgetMb: 36536},
			isFit:          true,
		},
		{
			name:           "does not fit beside a running verification that holds most of the disk",
			capacity:       AgentCapacity{MaxCPU: 8, MaxConcurrentJobs: 2, MaxDiskGb: 100},
			reservedDiskMb: 70000,
			backup:         indexedBackup,
			isFit:          false,
		},
		{
			name:           "fits when the restore peak equals the free disk",
			capacity:       AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 38},
			reservedDiskMb: 38912 - 38036,
			backup:         indexedBackup,
			expectedPlan:   RestorePlan{ParallelJobs: 4, DiskBudgetMb: 38036},
			isFit:          true,
		},
		{
			name:     "does not fit an agent without disk",
			capacity: AgentCapacity{MaxCPU: 4, MaxConcurrentJobs: 1, MaxDiskGb: 0},
			backup:   newPlannedBackup(1, 1, pq.Float64Array{}),
			isFit:    false,
		},
	}

	for _, planCase := range cases {
		t.Run(planCase.name, func(t *testing.T) {
			t.Parallel()

			plan, isFit := PlanRestore(planCase.capacity, planCase.reservedDiskMb, planCase.backup)

			assert.Equal(t, planCase.isFit, isFit)
			if planCase.isFit {
				assert.Equal(t, planCase.expectedPlan, plan)
			}
		})
	}
}
