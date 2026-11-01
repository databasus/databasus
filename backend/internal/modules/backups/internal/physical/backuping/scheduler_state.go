package backuping

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type schedulingState struct {
	lastFull              *catalog.Backup
	lastIncremental       *catalog.Backup
	hasUnresolvedMismatch bool
	extendableRootID      *uuid.UUID
	incrementalParentID   *uuid.UUID
	recentFullBackups     []*catalog.Backup
	chainError            error
	parentError           error
	recentFullsError      error
}

func (s *Scheduler) loadSchedulingState(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	config config.PhysicalBackupConfig,
) (schedulingState, error) {
	state := schedulingState{}
	var err error

	state.lastFull, _, err = s.backupRepo.FindLastFullAnyStatusByDatabase(ctx, config.DatabaseID)
	if err != nil || config.ForceFullRequestedAt != nil {
		return state, err
	}

	state.hasUnresolvedMismatch, err = s.repository.HasUnresolvedSystemIdentifierMismatch(ctx, config.DatabaseID)
	if err != nil || state.hasUnresolvedMismatch {
		return state, err
	}

	state.lastIncremental, _, err = s.backupRepo.FindLastIncrementalByDatabase(ctx, config.DatabaseID)
	if err != nil {
		return state, err
	}

	if !decidePhysicalBackupFromNewest(state, config, now).needsChain {
		return state, nil
	}

	chain, err := s.chainViewService.FindLastExtendableChainByDatabase(ctx, config.DatabaseID)
	if err != nil {
		state.chainError = err
		logger.ErrorContext(ctx, "failed to find extendable chain", slog.Any("error", err))
		return state, nil
	}

	if chain != nil {
		state.extendableRootID = &chain.RootFull.ID
		state.incrementalParentID, state.parentError = s.resolveIncrParent(ctx, chain.RootFull.ID)
		if state.parentError != nil {
			logger.ErrorContext(ctx, "failed to resolve incremental parent", slog.Any("error", state.parentError))
		}
		return state, nil
	}

	state.recentFullBackups, state.recentFullsError = s.backupRepo.FindNewestFullAnyStatusByDatabase(
		ctx,
		config.DatabaseID,
		recentFullAttemptsWindow,
	)
	if state.recentFullsError != nil {
		logger.ErrorContext(ctx, "failed to find recent full backups", slog.Any("error", state.recentFullsError))
	}

	return state, nil
}
