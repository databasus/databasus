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

// configEvaluation is one database's turn in a scheduler pass.
type configEvaluation struct {
	now    time.Time
	config config.PhysicalBackupConfig
	state  schedulingState
}

func (s *Scheduler) loadNewestStates(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]schedulingState, error) {
	newestBackups, err := s.backupRepo.GetNewestBackupsByDatabases(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	unresolvedDatabases, err := s.repository.GetDatabasesWithUnresolvedMismatch(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	states := make(map[uuid.UUID]schedulingState, len(databaseIDs))
	for _, databaseID := range databaseIDs {
		states[databaseID] = schedulingState{
			lastFull:              newestBackups[databaseID].Full,
			lastIncremental:       newestBackups[databaseID].Incremental,
			hasUnresolvedMismatch: unresolvedDatabases[databaseID],
		}
	}

	return states, nil
}

// loadSchedulingState reads one database's whole state for the decision repeated under the lock.
func (s *Scheduler) loadSchedulingState(
	ctx context.Context,
	now time.Time,
	backupConfig config.PhysicalBackupConfig,
) (schedulingState, error) {
	newestStates, err := s.loadNewestStates(ctx, []uuid.UUID{backupConfig.DatabaseID})
	if err != nil {
		return schedulingState{}, err
	}

	return s.addChainFacts(ctx, s.logger, configEvaluation{
		now:    now,
		config: backupConfig,
		state:  newestStates[backupConfig.DatabaseID],
	}), nil
}

// addChainFacts reads the chain only when the decision from the newest backups asks for it. A failed
// read is recorded in the state, so the decision can still choose a backup that needs no chain.
func (s *Scheduler) addChainFacts(
	ctx context.Context,
	logger *slog.Logger,
	evaluation configEvaluation,
) schedulingState {
	state := evaluation.state
	if !decidePhysicalBackupFromNewest(state, evaluation.config, evaluation.now).needsChain {
		return state
	}

	databaseID := evaluation.config.DatabaseID

	chain, err := s.chainViewService.FindLastExtendableChainByDatabase(ctx, databaseID)
	if err != nil {
		state.chainError = err
		logger.ErrorContext(ctx, "failed to find extendable chain", slog.Any("error", err))

		return state
	}

	if chain != nil {
		state.extendableRootID = &chain.RootFull.ID
		state.incrementalParentID, state.parentError = s.resolveIncrParent(ctx, chain.RootFull.ID)
		if state.parentError != nil {
			logger.ErrorContext(ctx, "failed to resolve incremental parent", slog.Any("error", state.parentError))
		}

		return state
	}

	state.recentFullBackups, state.recentFullsError = s.backupRepo.FindNewestFullAnyStatusByDatabase(
		ctx,
		databaseID,
		recentFullAttemptsWindow,
	)
	if state.recentFullsError != nil {
		logger.ErrorContext(ctx, "failed to find recent full backups", slog.Any("error", state.recentFullsError))
	}

	return state
}
