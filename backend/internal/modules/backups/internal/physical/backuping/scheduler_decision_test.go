package backuping

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
)

func TestDecidePhysicalBackup(t *testing.T) {
	cases := []struct {
		name                     string
		change                   func(*schedulingState, *config.PhysicalBackupConfig, time.Time)
		expectedKind             catalog.BackupType
		shouldClearRequest       bool
		shouldHaveParent         bool
		shouldReplaceImmediately bool
	}{
		{
			name: "schedules nothing when neither cadence is due",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.lastFull.CreatedAt = now
			},
		},
		{
			name: "schedules nothing when a failed full is not due",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.lastFull.Status = catalog.BackupStatusError
				state.extendableRootID = nil
				state.recentFullBackups = nil
			},
		},
		{
			name: "retries a failed full when its cadence is due",
			change: func(state *schedulingState, config *config.PhysicalBackupConfig, _ time.Time) {
				state.lastFull.Status = catalog.BackupStatusError
				config.FullBackupSchedule.Type = schedule.TypeHourly
			},
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name: "retries an incremental when the failed attempt leaves its chain extendable",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &catalog.Backup{
					Status:    catalog.BackupStatusError,
					CreatedAt: now.Add(-2 * time.Hour),
				}
			},
			expectedKind: catalog.BackupTypeIncremental,
		},
		{
			name: "schedules a full when both cadences are due",
			change: func(_ *schedulingState, config *config.PhysicalBackupConfig, _ time.Time) {
				config.FullBackupSchedule.Type = schedule.TypeHourly
			},
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name: "schedules an incremental when a daily incremental schedule passed its slot",
			change: func(state *schedulingState, config *config.PhysicalBackupConfig, now time.Time) {
				state.lastFull.CreatedAt = now.Add(-24 * time.Hour)
				config.IncrementalBackupSchedule = &schedule.Schedule{
					Type:      schedule.TypeDaily,
					TimeOfDay: new("04:00"),
				}
			},
			expectedKind: catalog.BackupTypeIncremental,
		},
		{
			name: "schedules nothing without an incremental schedule when the full cadence is not due",
			change: func(_ *schedulingState, config *config.PhysicalBackupConfig, _ time.Time) {
				config.IncrementalBackupSchedule = nil
			},
		},
		{
			name: "schedules no first full when the stored full cron has no date to run on",
			change: func(state *schedulingState, config *config.PhysicalBackupConfig, _ time.Time) {
				state.lastFull = nil
				state.extendableRootID = nil
				state.recentFullBackups = nil
				config.IncrementalBackupSchedule = nil
				config.FullBackupSchedule = schedule.Schedule{
					Type:           schedule.TypeCron,
					CronExpression: new("0 0 31 2 *"),
				}
			},
		},
		{
			name: "reanchors a broken chain when only the incremental cadence is due",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
			},
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name: "schedules nothing when recent full attempts all failed",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = []*catalog.Backup{
					{Status: catalog.BackupStatusError},
					{Status: catalog.BackupStatusError},
					{Status: catalog.BackupStatusError},
				}
			},
		},
		{
			name: "schedules nothing when recent full attempts were canceled",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = []*catalog.Backup{
					{Status: catalog.BackupStatusCanceled},
					{Status: catalog.BackupStatusCanceled},
					{Status: catalog.BackupStatusCanceled},
				}
			},
		},
		{
			name: "reanchors when one failed full follows a completed full",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.extendableRootID = nil
				state.recentFullBackups = append(
					[]*catalog.Backup{{Status: catalog.BackupStatusError}},
					state.recentFullBackups...,
				)
			},
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name: "waits for the full cadence when summaries are disabled",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.extendableRootID = nil
				state.lastIncremental = &catalog.Backup{
					Status:      catalog.BackupStatusChainBroken,
					ErrorReason: new(catalog.BackupErrorSummarizerOff),
					CreatedAt:   now.Add(-2 * time.Hour),
				}
			},
		},
		{
			name: "schedules nothing after a recent incremental cancellation",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &catalog.Backup{
					Status:    catalog.BackupStatusCanceled,
					CreatedAt: now,
				}
			},
		},
		{
			name:         "schedules a full when no full exists",
			change:       func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) { state.lastFull = nil },
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name:         "uses the full parent when a chain has no completed incremental",
			change:       func(_ *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {},
			expectedKind: catalog.BackupTypeIncremental,
		},
		{
			name: "uses the latest incremental parent when the chain has one",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.incrementalParentID = new(uuid.New())
			},
			expectedKind:     catalog.BackupTypeIncremental,
			shouldHaveParent: true,
		},
		{
			name: "schedules no incremental when only full backups are enabled",
			change: func(_ *schedulingState, config *config.PhysicalBackupConfig, _ time.Time) {
				config.BackupType = databases.PhysicalBackupTypeFull
			},
		},
		{
			name: "replaces a failed full immediately when failover interrupted it",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.lastFull.CreatedAt = now
				state.lastFull.Status = catalog.BackupStatusError
				state.lastFull.ErrorReason = new(catalog.BackupErrorFailoverDuringBackup)
			},
			expectedKind:             catalog.BackupTypeFull,
			shouldReplaceImmediately: true,
		},
		{
			name: "replaces a chain immediately when an incremental detects a timeline switch",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, now time.Time) {
				state.lastIncremental = &catalog.Backup{
					ID:          uuid.New(),
					Status:      catalog.BackupStatusChainBroken,
					ErrorReason: new(catalog.BackupErrorTimelineSwitchDetected),
					CreatedAt:   now,
					CompletedAt: new(now),
				}
			},
			expectedKind:             catalog.BackupTypeFull,
			shouldReplaceImmediately: true,
		},
		{
			name: "honors a forced full when the source identifier mismatches",
			change: func(state *schedulingState, config *config.PhysicalBackupConfig, now time.Time) {
				state.hasUnresolvedMismatch = true
				config.ForceFullRequestedAt = &now
			},
			expectedKind: catalog.BackupTypeFull,
		},
		{
			name: "schedules nothing automatically when the source identifier mismatches",
			change: func(state *schedulingState, _ *config.PhysicalBackupConfig, _ time.Time) {
				state.hasUnresolvedMismatch = true
			},
		},
		{
			name: "clears a forced incremental request when its chain disappeared",
			change: func(state *schedulingState, config *config.PhysicalBackupConfig, now time.Time) {
				state.extendableRootID = nil
				config.ForceIncrementalRequestedAt = &now
			},
			expectedKind:       catalog.BackupTypeFull,
			shouldClearRequest: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			now := time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
			full := &catalog.Backup{
				ID:        uuid.New(),
				Status:    catalog.BackupStatusCompleted,
				CreatedAt: now.Add(-3 * time.Hour),
			}
			state := schedulingState{
				lastFull:          full,
				extendableRootID:  &full.ID,
				recentFullBackups: []*catalog.Backup{full},
			}
			config := config.PhysicalBackupConfig{
				BackupType:                databases.PhysicalBackupTypeFullAndIncremental,
				FullBackupSchedule:        schedule.Schedule{Type: schedule.TypeWeekly},
				IncrementalBackupSchedule: &schedule.Schedule{Type: schedule.TypeHourly},
			}
			testCase.change(&state, &config, now)

			choice := decidePhysicalBackup(state, config, now)

			assert.Equal(t, testCase.expectedKind != "", choice.canSchedule)
			assert.Equal(t, testCase.expectedKind, choice.decision.kind)
			assert.Equal(t, testCase.shouldClearRequest, choice.shouldClearIncrementalRequest)
			if testCase.shouldReplaceImmediately {
				assert.Contains(
					t,
					[]string{"replacing FULL interrupted by failover", "re-anchoring after timeline switch"},
					choice.decision.reason,
				)
			}

			newestChoice := decidePhysicalBackupFromNewest(state, config, now)
			if newestChoice.needsChain {
				assert.Equal(t, choice, decidePhysicalBackupWithChain(state, config, now))
			} else {
				assert.Equal(t, choice, newestChoice)
			}

			if testCase.shouldHaveParent {
				require.NotNil(t, choice.decision.incrParentIncrID)
				assert.Equal(t, state.incrementalParentID, choice.decision.incrParentIncrID)
			}
			if choice.decision.kind == catalog.BackupTypeIncremental {
				assert.Equal(t, full.ID, choice.decision.incrRootFullBackupID)
			}
		})
	}
}

func TestDecidePhysicalBackupAfterReload(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name        string
		hasNewFull  bool
		hasMismatch bool
	}{
		{name: "schedules nothing when a full completes before the lock is acquired", hasNewFull: true},
		{name: "schedules nothing when a cluster mismatch is recorded before the lock is acquired", hasMismatch: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
			backupConfig := config.PhysicalBackupConfig{
				BackupType:         databases.PhysicalBackupTypeFull,
				FullBackupSchedule: schedule.Schedule{Type: schedule.TypeHourly},
			}
			state := schedulingState{
				lastFull: &catalog.Backup{CreatedAt: now.Add(-2 * time.Hour), Status: catalog.BackupStatusCompleted},
			}

			beforeLock := decidePhysicalBackupFromNewest(state, backupConfig, now)
			require.True(t, beforeLock.canSchedule)
			require.Equal(t, catalog.BackupTypeFull, beforeLock.decision.kind)

			if scenario.hasNewFull {
				state.lastFull = &catalog.Backup{CreatedAt: now, Status: catalog.BackupStatusCompleted}
			}
			state.hasUnresolvedMismatch = scenario.hasMismatch
			underLock := decidePhysicalBackupFromNewest(state, backupConfig, now)

			assert.False(t, underLock.canSchedule)
			assert.False(t, underLock.needsChain)
		})
	}
}
