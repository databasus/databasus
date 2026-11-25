package verifications

import (
	"math"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/modules/backups"
)

// A restore peaks at the archive and the restored database plus the sorts of the indexes it builds
// at once: each build spills about the size of its index, and pg_restore -j N builds up to N at a
// time. The reserve covers the WAL, which max_wal_size=1GB of the restore container keeps near
// 1 GB, and the empty cluster.
const restorePeakReserveMb = 1536

// A backup without recorded index sizes, or without a known database size, cannot be planned from
// its indexes. It restores with one job and the gap it always had: half the database, clamped,
// and the cap when the size is unknown.
const (
	minUnplannedGapMb               = 1536
	maxUnplannedGapMb               = 5120
	unplannedGapToDatabaseSizeRatio = 0.5
)

// RestorePlan is how a verification restores on its agent: with how many parallel jobs, and within
// how much disk, its disk budget.
type RestorePlan struct {
	ParallelJobs int
	DiskBudgetMb int64
}

// PlanRestore picks the most parallel jobs whose restore peak fits the agent's free disk, which is
// its disk capacity minus the disk budgets of the verifications it runs. It reports false when not
// even one job fits.
func PlanRestore(
	capacity AgentCapacity,
	reservedDiskMb int64,
	backup backups.LogicalBackup,
) (RestorePlan, bool) {
	freeDiskMb := int64(capacity.MaxDiskGb)*1024 - reservedDiskMb

	if backup.BackupLargestIndexSizesMb == nil || backup.BackupRawDbSizeMb <= 0 {
		unplannedPlan := RestorePlan{ParallelJobs: 1, DiskBudgetMb: getUnplannedDiskBudgetMb(backup)}

		return unplannedPlan, unplannedPlan.DiskBudgetMb <= freeDiskMb
	}

	maxParallelJobs := pgrestore.GetParallelJobs(getPerJobCPUBudget(capacity), backup.TimescaledbVersion != "")
	for parallelJobs := maxParallelJobs; parallelJobs >= 1; parallelJobs-- {
		restorePeakMb := getRestorePeakMb(backup, parallelJobs)
		if restorePeakMb <= freeDiskMb {
			return RestorePlan{ParallelJobs: parallelJobs, DiskBudgetMb: restorePeakMb}, true
		}
	}

	return RestorePlan{ParallelJobs: 0, DiskBudgetMb: 0}, false
}

func getPerJobCPUBudget(capacity AgentCapacity) int {
	if capacity.MaxConcurrentJobs <= 0 {
		return 1
	}

	return max(1, capacity.MaxCPU/capacity.MaxConcurrentJobs)
}

func getRestorePeakMb(backup backups.LogicalBackup, parallelJobs int) int64 {
	restorePeakMb := max(backup.BackupSizeMb, 0) + backup.BackupRawDbSizeMb + restorePeakReserveMb

	largestIndexSizesMb := backup.BackupLargestIndexSizesMb
	for _, indexSizeMb := range largestIndexSizesMb[:min(parallelJobs, len(largestIndexSizesMb))] {
		restorePeakMb += indexSizeMb
	}

	return int64(math.Ceil(restorePeakMb))
}

func getUnplannedDiskBudgetMb(backup backups.LogicalBackup) int64 {
	archiveSizeMb := max(backup.BackupSizeMb, 0)
	databaseSizeMb := max(backup.BackupRawDbSizeMb, 0)

	gapMb := float64(maxUnplannedGapMb)
	if databaseSizeMb > 0 {
		gapMb = min(maxUnplannedGapMb, max(minUnplannedGapMb, databaseSizeMb*unplannedGapToDatabaseSizeRatio))
	}

	return int64(math.Ceil(archiveSizeMb + databaseSizeMb + gapMb))
}
