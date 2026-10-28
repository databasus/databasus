package backuping_physical

import (
	"log/slog"
	"time"

	"github.com/google/uuid"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/storage"
)

type schedulingState struct {
	lastFull              *physical_models.PhysicalFullBackup
	lastIncremental       *physical_models.PhysicalIncrementalBackup
	hasUnresolvedMismatch bool
	extendableRootID      *uuid.UUID
	incrementalParentID   *uuid.UUID
	recentFullBackups     []*physical_models.PhysicalFullBackup
	chainError            error
	parentError           error
	recentFullsError      error
}

func (s *PhysicalBackupsScheduler) loadSchedulingState(
	logger *slog.Logger,
	now time.Time,
	config backups.PhysicalBackupConfig,
) (schedulingState, error) {
	state := schedulingState{}
	var err error
	state.lastFull, err = s.fullRepo.FindLastFullAnyStatusByDatabase(config.DatabaseID)
	if err != nil || config.ForceFullRequestedAt != nil {
		return state, err
	}
	state.hasUnresolvedMismatch, err = hasUnresolvedSystemIdentifierMismatch(storage.GetDb(), config.DatabaseID)
	if err != nil || state.hasUnresolvedMismatch {
		return state, err
	}
	state.lastIncremental, err = s.incrRepo.FindLastByDatabase(config.DatabaseID)
	if err != nil {
		return state, err
	}
	if _, canReplace := decideImmediateReplacementFull(state.lastFull, state.lastIncremental); canReplace {
		return state, nil
	}
	isFullDue := config.FullBackupSchedule.IsDue(now, createdAtOrNil(state.lastFull))
	if !isIncrementalEnabled(config) || (config.ForceIncrementalRequestedAt == nil &&
		(isFullDue || !config.IncrementalBackupSchedule.IsDue(now, newestCreatedAt(state.lastFull, state.lastIncremental)))) {
		return state, nil
	}
	chain, err := s.chainViewService.FindLastExtendableChainByDatabase(config.DatabaseID)
	if err != nil {
		state.chainError = err
		logger.Error("failed to find extendable chain", "error", err)
		return state, nil
	}
	if chain != nil {
		state.extendableRootID = &chain.RootFull.ID
		state.incrementalParentID, state.parentError = s.resolveIncrParent(chain.RootFull.ID)
		if state.parentError != nil {
			logger.Error("failed to resolve incremental parent", "error", state.parentError)
		}
		return state, nil
	}
	if !isFullDue &&
		config.IncrementalBackupSchedule.IsDue(now, newestCreatedAt(state.lastFull, state.lastIncremental)) &&
		!isChainBrokenBySummarizerOff(state.lastIncremental) {
		state.recentFullBackups, state.recentFullsError = s.fullRepo.FindNewestAnyStatusByDatabase(
			config.DatabaseID,
			recentFullAttemptsWindow,
		)
		if state.recentFullsError != nil {
			logger.Error("failed to find recent full backups", "error", state.recentFullsError)
		}
	}
	return state, nil
}
