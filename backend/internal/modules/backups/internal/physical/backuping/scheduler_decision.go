package backuping

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type schedulingChoice struct {
	decision                      backupDecision
	canSchedule                   bool
	shouldClearIncrementalRequest bool
}

func decidePhysicalBackup(state schedulingState, config config.PhysicalBackupConfig, now time.Time) schedulingChoice {
	choice := schedulingChoice{}
	if config.ForceFullRequestedAt != nil {
		choice.decision = backupDecision{
			kind:                 catalog.BackupTypeFull,
			reason:               "full requested out of cadence",
			forceFullRequestedAt: config.ForceFullRequestedAt,
		}
		choice.canSchedule = true
		return choice
	}
	if state.hasUnresolvedMismatch {
		return choice
	}
	if decision, canReplace := decideImmediateReplacementFull(state.lastFull, state.lastIncremental); canReplace {
		choice.decision, choice.canSchedule = decision, true
		return choice
	}
	if config.ForceIncrementalRequestedAt != nil {
		choice.shouldClearIncrementalRequest = !isIncrementalEnabled(config) ||
			(state.chainError == nil && state.extendableRootID == nil)
		if isIncrementalEnabled(config) && state.chainError == nil && state.parentError == nil &&
			state.extendableRootID != nil {
			choice.decision = backupDecision{
				kind:                        catalog.BackupTypeIncremental,
				incrRootFullBackupID:        *state.extendableRootID,
				incrParentIncrID:            state.incrementalParentID,
				forceIncrementalRequestedAt: config.ForceIncrementalRequestedAt,
			}
			choice.canSchedule = true
			return choice
		}
	}
	if config.FullBackupSchedule.IsDue(now, createdAtOrNil(state.lastFull)) {
		choice.decision = backupDecision{kind: catalog.BackupTypeFull, reason: "full cadence is due"}
		choice.canSchedule = true
		return choice
	}
	if !isIncrementalEnabled(config) ||
		!config.IncrementalBackupSchedule.IsDue(
			now,
			newestCreatedAt(state.lastFull, state.lastIncremental),
		) || state.chainError != nil {
		return choice
	}
	if state.extendableRootID == nil {
		if isChainBrokenBySummarizerOff(state.lastIncremental) || state.recentFullsError != nil ||
			hasNoRecentCompletedFull(state.recentFullBackups) {
			return choice
		}
		choice.decision = backupDecision{
			kind:   catalog.BackupTypeFull,
			reason: "re-anchoring: no extendable chain",
		}
	} else {
		if state.parentError != nil {
			return choice
		}
		choice.decision = backupDecision{
			kind:                 catalog.BackupTypeIncremental,
			reason:               "incremental cadence is due",
			incrRootFullBackupID: *state.extendableRootID,
			incrParentIncrID:     state.incrementalParentID,
		}
	}
	choice.canSchedule = true
	return choice
}
