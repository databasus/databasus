package verifications

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/backups"
)

func backupWithSize(archiveMb, restoredMb float64) backups.LogicalBackup {
	return backups.LogicalBackup{
		BackupSizeMb:      archiveMb,
		BackupRawDbSizeMb: restoredMb,
	}
}

func Test_EvaluateDiskAdmission_AcrossScenarios_RespectsAgentBudget(t *testing.T) {
	cases := []struct {
		name              string
		capacity          AgentCapacity
		runningBackups    []backups.LogicalBackup
		candidate         backups.LogicalBackup
		isFitWithinBudget bool
	}{
		{
			name:              "empty agent fits small candidate",
			capacity:          AgentCapacity{MaxDiskGb: 10},
			runningBackups:    nil,
			candidate:         backupWithSize(50, 100),
			isFitWithinBudget: true,
		},
		{
			name:           "empty agent fits candidate at exact budget boundary",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			// archive 4096 + restored 4096 + gap 2048 = 10240 (the full 10 GB budget)
			candidate:         backupWithSize(4096, 4096),
			isFitWithinBudget: true,
		},
		{
			name:              "empty agent rejects candidate one MB over budget",
			capacity:          AgentCapacity{MaxDiskGb: 10},
			runningBackups:    nil,
			candidate:         backupWithSize(4097, 4096),
			isFitWithinBudget: false,
		},
		{
			name:     "one running, room for another small",
			capacity: AgentCapacity{MaxDiskGb: 12},
			runningBackups: []backups.LogicalBackup{
				backupWithSize(50, 200),
			},
			candidate:         backupWithSize(50, 200),
			isFitWithinBudget: true,
		},
		{
			name:     "one running, restored size of candidate blows budget",
			capacity: AgentCapacity{MaxDiskGb: 10},
			runningBackups: []backups.LogicalBackup{
				backupWithSize(500, 4000),
			},
			candidate:         backupWithSize(500, 4000),
			isFitWithinBudget: false,
		},
		{
			name:     "three concurrent small jobs fit in 25 GB",
			capacity: AgentCapacity{MaxDiskGb: 25},
			runningBackups: []backups.LogicalBackup{
				backupWithSize(100, 500),
				backupWithSize(100, 500),
				backupWithSize(100, 500),
			},
			candidate:         backupWithSize(100, 500),
			isFitWithinBudget: true,
		},
		{
			name:     "per-job gap saturation: three RUNNING already over a 5 GB agent",
			capacity: AgentCapacity{MaxDiskGb: 5},
			runningBackups: []backups.LogicalBackup{
				backupWithSize(100, 400),
				backupWithSize(100, 400),
				backupWithSize(100, 400),
			},
			candidate:         backupWithSize(10, 10),
			isFitWithinBudget: false,
		},
		{
			name:              "tiny database fits a 5 GB agent",
			capacity:          AgentCapacity{MaxDiskGb: 5},
			runningBackups:    nil,
			candidate:         backupWithSize(0, 0.01),
			isFitWithinBudget: true,
		},
		{
			name:              "unknown raw DB size keeps the full gap and fits a roomy agent",
			capacity:          AgentCapacity{MaxDiskGb: 10},
			runningBackups:    nil,
			candidate:         backupWithSize(200, 0),
			isFitWithinBudget: true,
		},
		{
			name:              "unknown raw DB size keeps the full gap and does not fit a 5 GB agent",
			capacity:          AgentCapacity{MaxDiskGb: 5},
			runningBackups:    nil,
			candidate:         backupWithSize(200, 0),
			isFitWithinBudget: false,
		},
		{
			name:              "zero-size candidate still consumes the per-job gap and fits",
			capacity:          AgentCapacity{MaxDiskGb: 10},
			runningBackups:    nil,
			candidate:         backupWithSize(0, 0),
			isFitWithinBudget: true,
		},
		{
			name:              "negative sizes clamp to zero and still fit",
			capacity:          AgentCapacity{MaxDiskGb: 10},
			runningBackups:    nil,
			candidate:         backupWithSize(-50, -100),
			isFitWithinBudget: true,
		},
		{
			name:              "zero-capacity agent rejects everything",
			capacity:          AgentCapacity{MaxDiskGb: 0},
			runningBackups:    nil,
			candidate:         backupWithSize(1, 1),
			isFitWithinBudget: false,
		},
		{
			name:              "raw DB size dominates archive size in the cost",
			capacity:          AgentCapacity{MaxDiskGb: 2},
			runningBackups:    nil,
			candidate:         backupWithSize(10, 2000),
			isFitWithinBudget: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			admission := EvaluateDiskAdmission(
				testCase.capacity,
				testCase.runningBackups,
				testCase.candidate,
			)

			assert.Equal(t, testCase.isFitWithinBudget, admission.IsFit)
		})
	}
}

func Test_EstimateRequiredForRestoreDiskMb_AcrossRestoredSizes_ClampsGapBetweenFloorAndCap(t *testing.T) {
	cases := []struct {
		name           string
		backup         backups.LogicalBackup
		requiredDiskMb int64
	}{
		{
			name:           "unknown raw DB size falls back to the gap cap",
			backup:         backupWithSize(200, 0),
			requiredDiskMb: 200 + maxDiskGapMb,
		},
		{
			name:           "tiny restored size uses the gap floor",
			backup:         backupWithSize(0, 0.01),
			requiredDiskMb: 1 + minDiskGapMb,
		},
		{
			name:           "restored size below the floor crossover uses the gap floor",
			backup:         backupWithSize(100, 2000),
			requiredDiskMb: 2100 + minDiskGapMb,
		},
		{
			name:           "restored size between floor and cap scales the gap",
			backup:         backupWithSize(500, 6000),
			requiredDiskMb: 6500 + 3000,
		},
		{
			name:           "restored size above 10 GB pins the gap at the cap",
			backup:         backupWithSize(4000, 40000),
			requiredDiskMb: 44000 + maxDiskGapMb,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.requiredDiskMb, EstimateRequiredForRestoreDiskMb(testCase.backup))
		})
	}
}
