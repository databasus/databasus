package verifications

import (
	"math"

	"databasus-backend/internal/modules/backups"
)

// A restore peaks above archive + final DB size: WAL, index sort spill and the
// cluster baseline all land inside the disk the agent watches. That overhead
// roughly tracks the data loaded, hence the ratio; the floor covers a near-empty
// cluster, and the ceiling keeps a huge database from demanding unbounded
// headroom. BackupRawDbSizeMb is 0 when the size probe failed, so 0 is
// "unknown", not "empty".
const (
	minDiskGapMb               = 1536
	maxDiskGapMb               = 5120
	diskGapToRestoredSizeRatio = 0.5
)

type DiskAdmission struct {
	IsFit               bool
	CandidateRequiredMb int64
	RunningUsedMb       int64
	TotalBudgetMb       int64
}

func EvaluateDiskAdmission(
	capacity AgentCapacity,
	runningBackups []backups.LogicalBackup,
	candidateBackup backups.LogicalBackup,
) DiskAdmission {
	admission := DiskAdmission{
		CandidateRequiredMb: EstimateRequiredForRestoreDiskMb(candidateBackup),
		RunningUsedMb:       sumEstimatedRequiredDiskMb(runningBackups),
		TotalBudgetMb:       int64(capacity.MaxDiskGb) * 1024,
	}

	if capacity.MaxDiskGb <= 0 {
		return admission
	}

	admission.IsFit = admission.RunningUsedMb+admission.CandidateRequiredMb <= admission.TotalBudgetMb

	return admission
}

func EstimateRequiredForRestoreDiskMb(backup backups.LogicalBackup) int64 {
	archiveSizeMb := max(backup.BackupSizeMb, 0)
	restoredSizeMb := max(backup.BackupRawDbSizeMb, 0)

	gapMb := float64(maxDiskGapMb)
	if restoredSizeMb > 0 {
		gapMb = min(maxDiskGapMb, max(minDiskGapMb, restoredSizeMb*diskGapToRestoredSizeRatio))
	}

	return int64(math.Ceil(archiveSizeMb + restoredSizeMb + gapMb))
}

func sumEstimatedRequiredDiskMb(runningBackups []backups.LogicalBackup) int64 {
	var total int64
	for _, backup := range runningBackups {
		total += EstimateRequiredForRestoreDiskMb(backup)
	}

	return total
}
