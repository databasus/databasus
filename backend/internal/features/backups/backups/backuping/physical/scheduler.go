package backuping_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
)

// PhysicalBackupsScheduler drives the FULL/INCR decision, the cross-table
// single-in-flight claim + typed-row INSERT, and the in-process hand-off to the
// PhysicalBackuper. It is the "front half"; the backuper writes terminal
// status and releases the claim.
//
// Retry policy (no IsRetryIfFailed/MaxFailedTriesCount columns exist on the
// physical config, so retry is derived from chain state, not a counter):
//   - ERROR: the chain stays extendable, so the next cadence-due tick re-attempts
//     the SAME kind. A freshly-failed attempt is the newest row of its kind, so
//     the cadence check on its created_at prevents a tight retry loop.
//   - CHAIN_BROKEN: nothing is left to extend, so the incremental cadence re-anchors
//     with a new FULL rather than the tick going idle until the FULL cadence, unless
//     the source produced no completed FULL recently.
//   - CANCELED: neither COMPLETED nor CHAIN_BROKEN; the chain it belonged to stays
//     extendable and resumes on cadence — no immediate auto-retry.
type PhysicalBackupsScheduler struct {
	fullRepo                  *physical_repositories.PhysicalFullBackupRepository
	incrRepo                  *physical_repositories.PhysicalIncrementalBackupRepository
	inFlightRepo              *physical_repositories.PhysicalInFlightBackupRepository
	backupConfigService       *backups_config_physical.BackupConfigService
	chainViewService          *chain_view.ChainViewService
	taskCancellationRequester *tasks_cancellation.Requester
	// getStorages is read per call: each test application installs its own module.
	getStorages func() storages.Module
	backuper    *PhysicalBackuper

	lastTickTime atomicTime
	logger       *slog.Logger

	hasRun  atomic.Bool
	isReady atomic.Bool
}

// IsRunning reports whether the scheduler has finished startup recovery and is
// ready to dispatch work. Tests gate on it.
func (s *PhysicalBackupsScheduler) IsRunning() bool {
	return s.isReady.Load()
}

func (s *PhysicalBackupsScheduler) IsSchedulerRunning() bool {
	return s.lastTickTime.Load().After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *PhysicalBackupsScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	startupRecoveryLogger := s.logger.With("job_id", uuid.New(), "job_name", schedulerJobName)
	lifecycleLogger := s.logger.With("job_name", schedulerJobName)

	s.lastTickTime.Store(time.Now().UTC())

	if err := s.recoverInFlightBackupsOnRestart(ctx, startupRecoveryLogger); err != nil {
		startupRecoveryLogger.ErrorContext(ctx, "failed to recover in-flight physical backups on restart", "error", err)

		panic(err)
	}

	s.isReady.Store(true)
	defer s.isReady.Store(false)

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(schedulerTickInterval)
	defer ticker.Stop()

	lifecycleLogger.InfoContext(ctx, "physical backup scheduler started")

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "physical backup scheduler stopped")

			return
		case <-ticker.C:
			// A fresh job_id per tick: it is the correlation ID for one run, not for the process.
			tickLogger := s.logger.With("job_id", uuid.New(), "job_name", schedulerJobName)

			if err := s.runPendingBackups(ctx, tickLogger); err != nil {
				tickLogger.ErrorContext(ctx, "failed to run pending physical backups", "error", err)
			}

			s.lastTickTime.Store(time.Now().UTC())
		}
	}
}

func (s *PhysicalBackupsScheduler) runPendingBackups(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, backupConfig := range enabledConfigs {
		s.evaluateConfig(ctx, logger, now, backupConfig)
	}

	return nil
}

func (s *PhysicalBackupsScheduler) evaluateConfig(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	logger = logger.With("database_id", backupConfig.DatabaseID)

	if backupConfig.StorageID == nil {
		logger.ErrorContext(ctx, "physical backup config has no storage id; skipping")

		return
	}

	decision, ok := s.decideBackupKind(logger, now, backupConfig)
	if !ok {
		return
	}

	s.scheduleBackup(ctx, logger, backupConfig, decision)
}

// backupDecision is the outcome of the per-tick FULL-vs-INCR decision.
type backupDecision struct {
	kind physical_enums.PhysicalBackupType
	// Why this backup was scheduled - "forced" and "cadence due" produce an identical row, so
	// without it an operator cannot tell an out-of-cadence request from a normal tick.
	reason                      string
	incrRootFullBackupID        uuid.UUID
	incrParentIncrID            *uuid.UUID
	forceFullRequestedAt        *time.Time
	forceIncrementalRequestedAt *time.Time
	immediateTrigger            *backupTrigger
}

type backupTrigger struct {
	kind     physical_enums.PhysicalBackupType
	backupID uuid.UUID
}

// decideBackupKind picks FULL or INCR purely from catalog state and cadence, with
// no source-PG connection: the shipped executors handle summarizer and timeline
// reality at run time.
func (s *PhysicalBackupsScheduler) decideBackupKind(
	logger *slog.Logger,
	now time.Time,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) (backupDecision, bool) {
	lastFull, err := s.fullRepo.FindLastFullAnyStatusByDatabase(backupConfig.DatabaseID)
	if err != nil {
		logger.Error("failed to find last full backup", "error", err)

		return backupDecision{}, false
	}

	if backupConfig.ForceFullRequestedAt != nil {
		return backupDecision{
			kind:                 physical_enums.PhysicalBackupTypeFull,
			reason:               "full requested out of cadence",
			forceFullRequestedAt: backupConfig.ForceFullRequestedAt,
		}, true
	}

	hasUnresolvedMismatch, err := hasUnresolvedSystemIdentifierMismatch(storage.GetDb(), backupConfig.DatabaseID)
	if err != nil {
		logger.Error("failed to inspect system identifier mismatch state", "error", err)

		return backupDecision{}, false
	}
	if hasUnresolvedMismatch {
		logger.Debug("automatic scheduling suppressed after system identifier mismatch")

		return backupDecision{}, false
	}

	lastIncr, err := s.incrRepo.FindLastByDatabase(backupConfig.DatabaseID)
	if err != nil {
		logger.Error("failed to find last incremental backup", "error", err)

		return backupDecision{}, false
	}

	if decision, ok := decideImmediateReplacementFull(lastFull, lastIncr); ok {
		return decision, true
	}

	if backupConfig.ForceIncrementalRequestedAt != nil {
		if decision, ok := s.decideForcedIncremental(logger, backupConfig); ok {
			return decision, true
		}
	}

	if backupConfig.FullBackupInterval.ShouldTriggerBackup(now, createdAtOrNil(lastFull)) {
		return backupDecision{
			kind:   physical_enums.PhysicalBackupTypeFull,
			reason: "full cadence is due",
		}, true
	}

	if !isIncrementalEnabled(backupConfig) {
		logger.Debug("nothing due: full cadence is not due and incrementals are disabled")

		return backupDecision{}, false
	}

	lastBackupTime := newestCreatedAt(lastFull, lastIncr)

	if !backupConfig.IncrementalBackupInterval.ShouldTriggerBackup(now, lastBackupTime) {
		logger.Debug("nothing due: neither cadence has elapsed")

		return backupDecision{}, false
	}

	extendableChain, err := s.chainViewService.FindLastExtendableChainByDatabase(backupConfig.DatabaseID)
	if err != nil {
		logger.Error("failed to find extendable chain", "error", err)

		return backupDecision{}, false
	}

	if extendableChain == nil {
		return s.decideReAnchoringFull(logger, backupConfig, lastIncr)
	}

	parentIncrID, err := s.resolveIncrParent(extendableChain.RootFull.ID)
	if err != nil {
		logger.Error("failed to resolve incremental parent", "error", err)

		return backupDecision{}, false
	}

	return backupDecision{
		kind:                 physical_enums.PhysicalBackupTypeIncremental,
		reason:               "incremental cadence is due",
		incrRootFullBackupID: extendableChain.RootFull.ID,
		incrParentIncrID:     parentIncrID,
	}, true
}

// decideForcedIncremental honors an out-of-cadence incremental request. It can
// only be satisfied when incrementals are enabled and an extendable chain
// exists; otherwise the stale request is cleared (so it can't loop forever) and
// ok=false lets the caller fall through to normal cadence evaluation. The
// controller already rejects "incremental with no chain" up front, but the
// chain can disappear between request and tick, so the scheduler must cope.
func (s *PhysicalBackupsScheduler) decideForcedIncremental(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) (backupDecision, bool) {
	if !isIncrementalEnabled(backupConfig) {
		s.clearIncrementalRequest(logger, backupConfig)

		return backupDecision{}, false
	}

	extendableChain, err := s.chainViewService.FindLastExtendableChainByDatabase(backupConfig.DatabaseID)
	if err != nil {
		logger.Error("failed to find extendable chain for forced incremental", "error", err)

		return backupDecision{}, false
	}

	if extendableChain == nil {
		logger.Warn("forced incremental requested but no extendable chain exists; clearing request")
		s.clearIncrementalRequest(logger, backupConfig)

		return backupDecision{}, false
	}

	parentIncrID, err := s.resolveIncrParent(extendableChain.RootFull.ID)
	if err != nil {
		logger.Error("failed to resolve incremental parent for forced incremental", "error", err)

		return backupDecision{}, false
	}

	return backupDecision{
		kind:                        physical_enums.PhysicalBackupTypeIncremental,
		incrRootFullBackupID:        extendableChain.RootFull.ID,
		incrParentIncrID:            parentIncrID,
		forceIncrementalRequestedAt: backupConfig.ForceIncrementalRequestedAt,
	}, true
}

func (s *PhysicalBackupsScheduler) clearIncrementalRequest(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	if err := s.backupConfigService.ClearIncrementalBackupRequest(
		backupConfig.DatabaseID, backupConfig.ForceIncrementalRequestedAt,
	); err != nil {
		logger.Error("failed to clear incremental backup request", "error", err)
	}
}

// resolveIncrParent returns the latest COMPLETED INCR in the chain as the new
// INCR's parent, or nil when the chain has none yet (then the parent is the
// FULL, resolved from root_full_backup_id at read time).
func (s *PhysicalBackupsScheduler) resolveIncrParent(rootFullBackupID uuid.UUID) (*uuid.UUID, error) {
	parent, err := s.incrRepo.FindLatestCompletedByRootFull(rootFullBackupID)
	if err != nil {
		return nil, err
	}

	if parent == nil {
		return nil, nil
	}

	return &parent.ID, nil
}

func (s *PhysicalBackupsScheduler) scheduleBackup(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
	decision backupDecision,
) {
	backupID := uuid.New()
	logger = logger.With("backup_id", backupID, "backup_kind", decision.kind)

	claimed, err := s.claimAndInsert(backupConfig, backupID, decision)
	if err != nil {
		logger.ErrorContext(ctx, "failed to claim and insert backup row", "error", err)

		return
	}

	if !claimed {
		logger.DebugContext(ctx, "in-flight slot already claimed by another instance; skipping")

		return
	}

	if decision.forceFullRequestedAt != nil {
		if err := s.backupConfigService.ClearFullBackupRequest(
			backupConfig.DatabaseID,
			decision.forceFullRequestedAt,
		); err != nil {
			logger.ErrorContext(ctx, "failed to clear forced full request", "error", err)
		}
	}

	if decision.forceIncrementalRequestedAt != nil {
		if err := s.backupConfigService.ClearIncrementalBackupRequest(
			backupConfig.DatabaseID,
			decision.forceIncrementalRequestedAt,
		); err != nil {
			logger.ErrorContext(ctx, "failed to clear forced incremental request", "error", err)
		}
	}

	go s.backuper.MakeBackup(ctx, backupID, true)

	logger.InfoContext(ctx, fmt.Sprintf("scheduled physical %s backup: %s", decision.kind, decision.reason))
}

// claimAndInsert reserves the cross-table in-flight slot and inserts the typed
// IN_PROGRESS row in ONE transaction. Both use the tx handle so they commit
// together; a lost claim (another instance won) commits nothing and returns
// claimed=false. The typed INSERT goes through tx.Create, NOT repo.Save (which
// uses the global DB) — that is what makes the claim and the row atomic.
func (s *PhysicalBackupsScheduler) claimAndInsert(
	backupConfig *backups_config_physical.PhysicalBackupConfig,
	backupID uuid.UUID,
	decision backupDecision,
) (bool, error) {
	now := time.Now().UTC()
	claimed := false

	txErr := storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := physical_repositories.AcquireBackupAndOrphanCleanupLock(tx, backupConfig.DatabaseID); err != nil {
			return err
		}

		ok, claimErr := s.inFlightRepo.Claim(tx, physical_repositories.ClaimSpec{
			DatabaseID: backupConfig.DatabaseID,
			BackupType: decision.kind,
			BackupID:   backupID,
		})
		if claimErr != nil {
			return claimErr
		}

		if !ok {
			return nil
		}

		if decision.forceFullRequestedAt == nil {
			hasUnresolvedMismatch, err := hasUnresolvedSystemIdentifierMismatch(tx, backupConfig.DatabaseID)
			if err != nil {
				return err
			}
			if hasUnresolvedMismatch {
				return errBackupDecisionSuperseded
			}
		}

		if decision.immediateTrigger != nil {
			isCurrent, err := isImmediateReplacementTriggerCurrent(
				tx,
				backupConfig.DatabaseID,
				*decision.immediateTrigger,
			)
			if err != nil {
				return err
			}
			if !isCurrent {
				return errBackupDecisionSuperseded
			}
		}

		claimed = true

		if decision.kind == physical_enums.PhysicalBackupTypeFull {
			return tx.Create(&physical_models.PhysicalFullBackup{
				ID:         backupID,
				DatabaseID: backupConfig.DatabaseID,
				StorageID:  *backupConfig.StorageID,
				Status:     physical_enums.PhysicalBackupStatusInProgress,
				CreatedAt:  now,
			}).Error
		}

		return tx.Create(&physical_models.PhysicalIncrementalBackup{
			ID:                        backupID,
			DatabaseID:                backupConfig.DatabaseID,
			StorageID:                 *backupConfig.StorageID,
			RootFullBackupID:          decision.incrRootFullBackupID,
			ParentIncrementalBackupID: decision.incrParentIncrID,
			Status:                    physical_enums.PhysicalBackupStatusInProgress,
			CreatedAt:                 now,
		}).Error
	})
	if errors.Is(txErr, errBackupDecisionSuperseded) {
		return false, nil
	}
	if txErr != nil {
		return false, txErr
	}

	return claimed, nil
}

func decideImmediateReplacementFull(
	lastFull *physical_models.PhysicalFullBackup,
	lastIncr *physical_models.PhysicalIncrementalBackup,
) (backupDecision, bool) {
	if lastFull != nil && lastFull.Status == physical_enums.PhysicalBackupStatusError &&
		hasFullBackupErrorReason(lastFull, physical_enums.PhysicalBackupErrorFailoverDuringBackup) {
		return backupDecision{
			kind:   physical_enums.PhysicalBackupTypeFull,
			reason: "replacing FULL interrupted by failover",
			immediateTrigger: &backupTrigger{
				kind:     physical_enums.PhysicalBackupTypeFull,
				backupID: lastFull.ID,
			},
		}, true
	}

	if !hasIncrementalBackupErrorReason(lastIncr, physical_enums.PhysicalBackupErrorTimelineSwitchDetected) ||
		lastIncr.Status != physical_enums.PhysicalBackupStatusChainBroken ||
		lastIncr.CompletedAt == nil {
		return backupDecision{}, false
	}

	if lastFull != nil && lastFull.CreatedAt.After(*lastIncr.CompletedAt) {
		return backupDecision{}, false
	}

	return backupDecision{
		kind:   physical_enums.PhysicalBackupTypeFull,
		reason: "re-anchoring after timeline switch",
		immediateTrigger: &backupTrigger{
			kind:     physical_enums.PhysicalBackupTypeIncremental,
			backupID: lastIncr.ID,
		},
	}, true
}

func hasFullBackupErrorReason(
	full *physical_models.PhysicalFullBackup,
	reason physical_enums.PhysicalBackupErrorReason,
) bool {
	return full != nil && full.ErrorReason != nil && *full.ErrorReason == reason
}

func hasIncrementalBackupErrorReason(
	incremental *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
) bool {
	return incremental != nil && incremental.ErrorReason != nil && *incremental.ErrorReason == reason
}

func hasUnresolvedSystemIdentifierMismatch(db *gorm.DB, databaseID uuid.UUID) (bool, error) {
	var state struct {
		LatestMismatchAt      *time.Time
		LatestCompletedFullAt *time.Time
	}

	err := db.Raw(`
		SELECT
			(
				SELECT MAX(terminal_at)
				FROM (
					SELECT COALESCE(completed_at, created_at) AS terminal_at FROM physical_full_backups
					WHERE database_id = ? AND error_reason = ?
					UNION ALL
					SELECT COALESCE(completed_at, created_at) AS terminal_at FROM physical_incremental_backups
					WHERE database_id = ? AND error_reason = ?
				) mismatches
			) AS latest_mismatch_at,
			(
				SELECT MAX(completed_at) FROM physical_full_backups
				WHERE database_id = ? AND status = ?
			) AS latest_completed_full_at
	`,
		databaseID, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch,
		databaseID, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch,
		databaseID, physical_enums.PhysicalBackupStatusCompleted,
	).Scan(&state).Error
	if err != nil {
		return false, err
	}

	return state.LatestMismatchAt != nil &&
		(state.LatestCompletedFullAt == nil || !state.LatestCompletedFullAt.After(*state.LatestMismatchAt)), nil
}

func isImmediateReplacementTriggerCurrent(
	tx *gorm.DB,
	databaseID uuid.UUID,
	trigger backupTrigger,
) (bool, error) {
	hasUnresolvedMismatch, err := hasUnresolvedSystemIdentifierMismatch(tx, databaseID)
	if err != nil || hasUnresolvedMismatch {
		return false, err
	}

	var lastFull physical_models.PhysicalFullBackup
	fullErr := tx.Where("database_id = ?", databaseID).Order("created_at DESC").First(&lastFull).Error
	if fullErr != nil && !errors.Is(fullErr, gorm.ErrRecordNotFound) {
		return false, fullErr
	}

	if trigger.kind == physical_enums.PhysicalBackupTypeFull {
		return fullErr == nil && lastFull.ID == trigger.backupID &&
			lastFull.Status == physical_enums.PhysicalBackupStatusError &&
			hasFullBackupErrorReason(&lastFull, physical_enums.PhysicalBackupErrorFailoverDuringBackup), nil
	}

	var lastIncr physical_models.PhysicalIncrementalBackup
	if err := tx.Where("database_id = ?", databaseID).Order("created_at DESC").First(&lastIncr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}

		return false, err
	}

	if lastIncr.ID != trigger.backupID || lastIncr.CompletedAt == nil ||
		lastIncr.Status != physical_enums.PhysicalBackupStatusChainBroken ||
		!hasIncrementalBackupErrorReason(&lastIncr, physical_enums.PhysicalBackupErrorTimelineSwitchDetected) {
		return false, nil
	}

	return errors.Is(fullErr, gorm.ErrRecordNotFound) || !lastFull.CreatedAt.After(*lastIncr.CompletedAt), nil
}

// recoverInFlightBackupsOnRestart reconciles the IN_PROGRESS backups the DB still
// holds when this instance (re)started. In single-instance mode the goroutine that
// was running each backup died with the previous process, so every in-flight claim
// is orphaned: fail the row and release the claim, freeing the database's
// single-in-flight slot for a fresh attempt.
//
// Runs once at Run() entry. The claim is the source of truth for in-flight state —
// claimAndInsert writes the claim and the typed row in one transaction — so
// iterating claims covers every backup that was running.
func (s *PhysicalBackupsScheduler) recoverInFlightBackupsOnRestart(
	ctx context.Context,
	logger *slog.Logger,
) error {
	claims, err := s.inFlightRepo.FindAll()
	if err != nil {
		return err
	}

	if len(claims) > 0 {
		logger.InfoContext(ctx, fmt.Sprintf(
			"failing %d physical backups orphaned by the previous run", len(claims)))
	}

	claimedBackupIDs := make(map[uuid.UUID]struct{}, len(claims))

	for _, claim := range claims {
		claimedBackupIDs[claim.BackupID] = struct{}{}

		s.failOrphanedBackup(ctx, logger, s.describeOrphan(logger, claim))
	}

	return s.failClaimlessInProgressBackups(ctx, logger, claimedBackupIDs)
}

// failClaimlessInProgressBackups fails any IN_PROGRESS row that has no matching
// in-flight claim. The atomic claim+insert should never leave such a row, so this
// is a defensive sweep: without it a stray row could sit IN_PROGRESS forever after
// a restart, since claim-driven recovery would never see it.
func (s *PhysicalBackupsScheduler) failClaimlessInProgressBackups(
	ctx context.Context,
	logger *slog.Logger,
	claimedBackupIDs map[uuid.UUID]struct{},
) error {
	fulls, err := s.fullRepo.FindAllInProgress()
	if err != nil {
		return err
	}

	for _, full := range fulls {
		if _, ok := claimedBackupIDs[full.ID]; ok {
			continue
		}

		// The atomic claim+insert should make this unreachable, so a hit means the invariant broke.
		logger.WarnContext(ctx, "found an in-progress full backup with no in-flight claim",
			"backup_id", full.ID, "database_id", full.DatabaseID)

		s.failOrphanedBackup(ctx, logger, orphanedBackupSpec{
			Kind:             physical_enums.PhysicalBackupTypeFull,
			BackupID:         full.ID,
			DatabaseID:       full.DatabaseID,
			StorageID:        full.StorageID,
			FileName:         valueOrEmpty(full.FileName),
			ManifestFileName: valueOrEmpty(full.ManifestFileName),
		})
	}

	incrementals, err := s.incrRepo.FindAllInProgress()
	if err != nil {
		return err
	}

	for _, incremental := range incrementals {
		if _, ok := claimedBackupIDs[incremental.ID]; ok {
			continue
		}

		logger.WarnContext(ctx, "found an in-progress incremental backup with no in-flight claim",
			"backup_id", incremental.ID, "database_id", incremental.DatabaseID)

		s.failOrphanedBackup(ctx, logger, orphanedBackupSpec{
			Kind:             physical_enums.PhysicalBackupTypeIncremental,
			BackupID:         incremental.ID,
			DatabaseID:       incremental.DatabaseID,
			StorageID:        incremental.StorageID,
			FileName:         valueOrEmpty(incremental.FileName),
			ManifestFileName: valueOrEmpty(incremental.ManifestFileName),
		})
	}

	return nil
}

// A claim carries no file name of its own, and the sweep cannot hand back what it
// cannot name.
func (s *PhysicalBackupsScheduler) describeOrphan(
	logger *slog.Logger,
	claim *physical_models.PhysicalInFlightBackup,
) orphanedBackupSpec {
	spec := orphanedBackupSpec{
		Kind:       claim.BackupType,
		BackupID:   claim.BackupID,
		DatabaseID: claim.DatabaseID,
	}

	if claim.BackupType == physical_enums.PhysicalBackupTypeIncremental {
		row, err := s.incrRepo.FindByID(claim.BackupID)
		if err != nil || row == nil {
			logger.Warn("could not read an orphaned incremental backup row", "backup_id", claim.BackupID)

			return spec
		}

		spec.StorageID = row.StorageID
		spec.FileName = valueOrEmpty(row.FileName)
		spec.ManifestFileName = valueOrEmpty(row.ManifestFileName)

		return spec
	}

	row, err := s.fullRepo.FindByID(claim.BackupID)
	if err != nil || row == nil {
		logger.Warn("could not read an orphaned full backup row", "backup_id", claim.BackupID)

		return spec
	}

	spec.StorageID = row.StorageID
	spec.FileName = valueOrEmpty(row.FileName)
	spec.ManifestFileName = valueOrEmpty(row.ManifestFileName)

	return spec
}

// orphanedBackupSpec is one IN_PROGRESS row the previous run left behind, with
// the names its files carry so the sweep can hand them back.
type orphanedBackupSpec struct {
	Kind             physical_enums.PhysicalBackupType
	BackupID         uuid.UUID
	DatabaseID       uuid.UUID
	StorageID        uuid.UUID
	FileName         string
	ManifestFileName string
}

func (s *PhysicalBackupsScheduler) failOrphanedBackup(
	ctx context.Context,
	logger *slog.Logger,
	spec orphanedBackupSpec,
) {
	logger = logger.With("backup_id", spec.BackupID, "database_id", spec.DatabaseID)

	// Best-effort cancel of any locally-registered task; harmless if none
	// (a fresh process holds no registrations).
	if err := s.taskCancellationRequester.RequestCancellation(ctx, spec.BackupID); err != nil {
		logger.ErrorContext(ctx, "failed to cancel orphaned backup task", "error", err)
	}

	if err := s.failBackupAndReleaseClaim(ctx, spec,
		physical_enums.PhysicalBackupErrorApplicationRestart,
		"Backup was interrupted by an application restart and marked failed. Trigger a new backup.",
	); err != nil {
		logger.ErrorContext(ctx, "failed to fail orphaned backup on restart", "error", err)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"failed a %s backup orphaned by the previous run", spec.Kind))
}

// failBackupAndReleaseClaim flips one typed row to ERROR and deletes the
// matching in-flight claim in the SAME transaction. Pairing the two is
// essential: an orphan claim left behind would block every future tick from
// acquiring the cross-table single-in-flight slot for that DB, freezing it.
//
// The sweep hands the row's files back to cleanup in the same transaction, gated
// on the status flip taking effect.
func (s *PhysicalBackupsScheduler) failBackupAndReleaseClaim(
	ctx context.Context,
	spec orphanedBackupSpec,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) error {
	var typedModel any = &physical_models.PhysicalFullBackup{}
	if spec.Kind == physical_enums.PhysicalBackupTypeIncremental {
		typedModel = &physical_models.PhysicalIncrementalBackup{}
	}

	return storage.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, storage.GetDb())

		// Guard on IN_PROGRESS so a late completion that already wrote a terminal
		// status wins the race — the backuper's persist and this sweep both
		// conditionally transition the row, so at most one of them takes effect.
		transition := tx.Model(typedModel).
			Where("id = ? AND status = ?", spec.BackupID, physical_enums.PhysicalBackupStatusInProgress).
			Updates(map[string]any{
				"status":       physical_enums.PhysicalBackupStatusError,
				"error_reason": reason,
				"fail_message": message,
			})
		if transition.Error != nil {
			return transition.Error
		}

		// Only the sweep that actually moved the row owns its files. A late
		// completion that already wrote a terminal status keeps them.
		if transition.RowsAffected == 1 && spec.FileName != "" {
			references := []storages.StoredFileReference{
				{StorageID: spec.StorageID, FileName: spec.FileName},
				{StorageID: spec.StorageID, FileName: spec.FileName + metadataSuffix},
			}

			if spec.ManifestFileName != "" {
				references = append(references,
					storages.StoredFileReference{StorageID: spec.StorageID, FileName: spec.ManifestFileName})
			}

			if err := s.getStorages().RequestFileDeletions(ctx, references); err != nil {
				return err
			}
		}

		// Scope the claim delete to backup_id so failing a stale backup cannot
		// remove a newer backup's claim for the same database.
		return tx.Delete(
			&physical_models.PhysicalInFlightBackup{},
			"database_id = ? AND backup_id = ?",
			spec.DatabaseID,
			spec.BackupID,
		).Error
	})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func createdAtOrNil(full *physical_models.PhysicalFullBackup) *time.Time {
	if full == nil {
		return nil
	}

	return &full.CreatedAt
}

func newestCreatedAt(
	full *physical_models.PhysicalFullBackup,
	incr *physical_models.PhysicalIncrementalBackup,
) *time.Time {
	switch {
	case full == nil && incr == nil:
		return nil
	case incr == nil:
		return &full.CreatedAt
	case full == nil:
		return &incr.CreatedAt
	case incr.CreatedAt.After(full.CreatedAt):
		return &incr.CreatedAt
	default:
		return &full.CreatedAt
	}
}

// decideReAnchoringFull answers the incremental cadence when no chain is left to
// extend. Waiting for the FULL cadence instead would leave the database without
// new restore points for a whole FULL interval, which on a weekly FULL is a week.
// Two exceptions fall back to that wait anyway: a chain broken by SUMMARIZER_OFF
// (a fresh FULL cannot fix a server-side setting, so re-anchoring at incremental
// speed would just alternate full backups with broken incrementals forever) and
// a source that produced no completed FULL in its newest few attempts (the
// cluster is broken rather than merely chainless, and re-anchoring it at
// incremental speed only burns it).
func (s *PhysicalBackupsScheduler) decideReAnchoringFull(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
	lastIncr *physical_models.PhysicalIncrementalBackup,
) (backupDecision, bool) {
	if isChainBrokenBySummarizerOff(lastIncr) {
		logger.Debug("nothing due: chain broken by summarizer off, which a fresh FULL cannot fix")

		return backupDecision{}, false
	}

	recentFullBackups, err := s.fullRepo.FindNewestAnyStatusByDatabase(
		backupConfig.DatabaseID, recentFullAttemptsWindow)
	if err != nil {
		logger.Error("failed to find recent full backups", "error", err)

		return backupDecision{}, false
	}

	if hasNoRecentCompletedFull(recentFullBackups) {
		logger.Debug(fmt.Sprintf(
			"nothing due: no extendable chain and no completed full among the newest %d",
			recentFullAttemptsWindow))

		return backupDecision{}, false
	}

	return backupDecision{
		kind:   physical_enums.PhysicalBackupTypeFull,
		reason: "re-anchoring: no extendable chain",
	}, true
}

// The newest incremental row is the right witness: any completed INCR after it
// would have kept the chain extendable, so this branch would not run at all.
// Once the operator re-enables summarize_wal, incrementals resume with the next
// FULL on its own cadence (or a forced FULL), because the scheduler reads only
// the catalog and never sees the live GUC.
func isChainBrokenBySummarizerOff(lastIncr *physical_models.PhysicalIncrementalBackup) bool {
	return lastIncr != nil &&
		lastIncr.Status == physical_enums.PhysicalBackupStatusChainBroken &&
		lastIncr.ErrorReason != nil &&
		*lastIncr.ErrorReason == physical_enums.PhysicalBackupErrorSummarizerOff
}

// Asking for a completed FULL rather than counting failures also weighs cancels:
// a source whose newest attempts all failed or were cancelled has produced
// nothing to anchor on. A single cancel among completed neighbours does not
// engage the brake, matching the CANCELED policy of resuming on cadence. An empty history
// answers true, which only ever means "wait", never "hammer the source".
func hasNoRecentCompletedFull(recentFullBackups []*physical_models.PhysicalFullBackup) bool {
	return !slices.ContainsFunc(recentFullBackups,
		func(fullBackup *physical_models.PhysicalFullBackup) bool {
			return fullBackup.Status == physical_enums.PhysicalBackupStatusCompleted
		})
}

func isIncrementalEnabled(backupConfig *backups_config_physical.PhysicalBackupConfig) bool {
	if backupConfig.PostgresqlPhysical == nil {
		return false
	}

	backupType := backupConfig.PostgresqlPhysical.BackupType

	return backupType == postgresql_physical.BackupTypeFullAndIncremental ||
		backupType == postgresql_physical.BackupTypeFullIncrementalAndWalStream
}
