package backuping_physical

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

func TestDecidePhysicalBackup(t *testing.T) {
	cases := []struct {
		name                     string
		change                   func(*schedulingState, *backups.PhysicalBackupConfig, time.Time)
		expectedKind             physical_enums.PhysicalBackupType
		shouldClearRequest       bool
		shouldHaveParent         bool
		shouldReplaceImmediately bool
	}{
		{
			name: "schedules nothing when neither cadence is due",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.lastFull.CreatedAt = now
			},
		},
		{
			name: "schedules nothing when a failed full is not due",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.lastFull.Status = physical_enums.PhysicalBackupStatusError
				state.extendableRootID = nil
				state.recentFullBackups = nil
			},
		},
		{
			name: "retries a failed full when its cadence is due",
			change: func(state *schedulingState, config *backups.PhysicalBackupConfig, _ time.Time) {
				state.lastFull.Status = physical_enums.PhysicalBackupStatusError
				config.FullBackupSchedule.Type = schedule.TypeHourly
			},
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name: "retries an incremental when the failed attempt leaves its chain extendable",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &physical_models.PhysicalIncrementalBackup{
					Status:    physical_enums.PhysicalBackupStatusError,
					CreatedAt: now.Add(-2 * time.Hour),
				}
			},
			expectedKind: physical_enums.PhysicalBackupTypeIncremental,
		},
		{
			name: "schedules a full when both cadences are due",
			change: func(_ *schedulingState, config *backups.PhysicalBackupConfig, _ time.Time) {
				config.FullBackupSchedule.Type = schedule.TypeHourly
			},
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name: "reanchors a broken chain when only the incremental cadence is due",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
			},
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name: "schedules nothing when recent full attempts all failed",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = []*physical_models.PhysicalFullBackup{
					{Status: physical_enums.PhysicalBackupStatusError},
					{Status: physical_enums.PhysicalBackupStatusError},
					{Status: physical_enums.PhysicalBackupStatusError},
				}
			},
		},
		{
			name: "schedules nothing when recent full attempts were canceled",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = []*physical_models.PhysicalFullBackup{
					{Status: physical_enums.PhysicalBackupStatusCanceled},
					{Status: physical_enums.PhysicalBackupStatusCanceled},
					{Status: physical_enums.PhysicalBackupStatusCanceled},
				}
			},
		},
		{
			name: "reanchors when one failed full follows a completed full",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = append(
					[]*physical_models.PhysicalFullBackup{{Status: physical_enums.PhysicalBackupStatusError}},
					state.recentFullBackups...)
			},
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name: "waits for the full cadence when summaries are disabled",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.extendableRootID = nil
				state.lastIncremental = &physical_models.PhysicalIncrementalBackup{
					Status:      physical_enums.PhysicalBackupStatusChainBroken,
					ErrorReason: new(physical_enums.PhysicalBackupErrorSummarizerOff),
					CreatedAt:   now.Add(-2 * time.Hour),
				}
			},
		},
		{
			name: "schedules nothing after a recent incremental cancellation",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &physical_models.PhysicalIncrementalBackup{
					Status:    physical_enums.PhysicalBackupStatusCanceled,
					CreatedAt: now,
				}
			},
		},
		{
			name:         "schedules a full when no full exists",
			change:       func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) { state.lastFull = nil },
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name:         "uses the full parent when a chain has no completed incremental",
			change:       func(_ *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {},
			expectedKind: physical_enums.PhysicalBackupTypeIncremental,
		},
		{
			name: "uses the latest incremental parent when the chain has one",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.incrementalParentID = new(uuid.New())
			},
			expectedKind:     physical_enums.PhysicalBackupTypeIncremental,
			shouldHaveParent: true,
		},
		{
			name: "schedules no incremental when only full backups are enabled",
			change: func(_ *schedulingState, config *backups.PhysicalBackupConfig, _ time.Time) {
				config.BackupType = databases.PhysicalBackupTypeFull
			},
		},
		{
			name: "replaces a failed full immediately when failover interrupted it",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.lastFull.CreatedAt = now
				state.lastFull.Status = physical_enums.PhysicalBackupStatusError
				state.lastFull.ErrorReason = new(physical_enums.PhysicalBackupErrorFailoverDuringBackup)
			},
			expectedKind:             physical_enums.PhysicalBackupTypeFull,
			shouldReplaceImmediately: true,
		},
		{
			name: "replaces a chain immediately when an incremental detects a timeline switch",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &physical_models.PhysicalIncrementalBackup{
					ID:          uuid.New(),
					Status:      physical_enums.PhysicalBackupStatusChainBroken,
					ErrorReason: new(physical_enums.PhysicalBackupErrorTimelineSwitchDetected),
					CreatedAt:   now,
					CompletedAt: new(now),
				}
			},
			expectedKind:             physical_enums.PhysicalBackupTypeFull,
			shouldReplaceImmediately: true,
		},
		{
			name: "honors a forced full when the source identifier mismatches",
			change: func(state *schedulingState, config *backups.PhysicalBackupConfig, now time.Time) {
				state.hasUnresolvedMismatch = true
				config.ForceFullRequestedAt = &now
			},
			expectedKind: physical_enums.PhysicalBackupTypeFull,
		},
		{
			name: "schedules nothing automatically when the source identifier mismatches",
			change: func(state *schedulingState, _ *backups.PhysicalBackupConfig, _ time.Time) {
				state.hasUnresolvedMismatch = true
			},
		},
		{
			name: "clears a forced incremental request when its chain disappeared",
			change: func(state *schedulingState, config *backups.PhysicalBackupConfig, now time.Time) {
				state.extendableRootID = nil
				config.ForceIncrementalRequestedAt = &now
			},
			expectedKind:       physical_enums.PhysicalBackupTypeFull,
			shouldClearRequest: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			now := time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
			full := &physical_models.PhysicalFullBackup{
				ID:        uuid.New(),
				Status:    physical_enums.PhysicalBackupStatusCompleted,
				CreatedAt: now.Add(-3 * time.Hour),
			}
			state := schedulingState{
				lastFull:          full,
				extendableRootID:  &full.ID,
				recentFullBackups: []*physical_models.PhysicalFullBackup{full},
			}
			config := backups.PhysicalBackupConfig{
				BackupType:                databases.PhysicalBackupTypeFullAndIncremental,
				FullBackupSchedule:        schedule.Schedule{Type: schedule.TypeWeekly},
				IncrementalBackupSchedule: &schedule.Schedule{Type: schedule.TypeHourly},
			}
			testCase.change(&state, &config, now)

			choice := decidePhysicalBackup(state, config, now)

			assert.Equal(t, testCase.expectedKind != "", choice.canSchedule)
			assert.Equal(t, testCase.expectedKind, choice.decision.kind)
			assert.Equal(t, testCase.shouldClearRequest, choice.shouldClearIncrementalRequest)
			assert.Equal(t, testCase.shouldReplaceImmediately, choice.decision.immediateTrigger != nil)
			if testCase.shouldReplaceImmediately {
				if state.lastIncremental != nil {
					assert.Equal(t, state.lastIncremental.ID, choice.decision.immediateTrigger.backupID)
				} else {
					assert.Equal(t, full.ID, choice.decision.immediateTrigger.backupID)
				}
			}
			if testCase.shouldHaveParent {
				require.NotNil(t, choice.decision.incrParentIncrID)
				assert.Equal(t, state.incrementalParentID, choice.decision.incrParentIncrID)
			}
			if choice.decision.kind == physical_enums.PhysicalBackupTypeIncremental {
				assert.Equal(t, full.ID, choice.decision.incrRootFullBackupID)
			}
		})
	}
}
