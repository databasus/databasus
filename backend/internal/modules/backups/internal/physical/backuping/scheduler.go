package backuping

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/storages"
)

type Scheduler struct {
	fullRepo                 *catalog.FullBackupRepository
	incrRepo                 *catalog.IncrementalBackupRepository
	inFlightRepo             *catalog.InFlightBackupRepository
	repository               *catalog.Repository
	configs                  *config.Service
	chainViewService         *chains.Service
	taskCancellationRegistry *background.CancellationRegistry
	storages                 storages.Module
	backuper                 *Backuper
	logger                   *slog.Logger
	clock                    clock.Source
	transactor               *database.Transactor
}

func (s *Scheduler) GetJob() background.Job {
	return background.Job{
		Name:     "backups_physical_scheduler",
		Interval: time.Second,
		Run:      func(ctx context.Context, _ time.Time) error { return s.RunPass(ctx) },
	}
}
func (s *Scheduler) RunPass(ctx context.Context) error { return s.runPendingBackups(ctx, s.logger) }

func (s *Scheduler) FailBackupsLeftInProgress(
	ctx context.Context,
	logger *slog.Logger,
) error {
	claims, err := s.inFlightRepo.FindAll(ctx)
	if err != nil {
		return err
	}

	if len(claims) > 0 {
		logger.InfoContext(
			ctx,
			"failing physical backups orphaned by the previous run", slog.Any("backup_count", len(claims)),
		)
	}

	claimedBackupIDs := make(map[uuid.UUID]struct{}, len(claims))

	for _, claim := range claims {
		claimedBackupIDs[claim.BackupID] = struct{}{}

		s.failOrphanedBackup(ctx, logger, s.describeOrphan(ctx, logger, claim))
	}

	return s.failClaimlessInProgressBackups(ctx, logger, claimedBackupIDs)
}

func (s *Scheduler) runPendingBackups(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := s.configs.ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	for _, backupConfig := range enabledConfigs {
		s.evaluateConfig(ctx, logger, now, backupConfig)
	}

	return nil
}

type backupDecision struct {
	kind catalog.BackupType

	reason                      string
	incrRootFullBackupID        uuid.UUID
	incrParentIncrID            *uuid.UUID
	forceFullRequestedAt        *time.Time
	forceIncrementalRequestedAt *time.Time
	immediateTrigger            *backupTrigger
}

type backupTrigger struct {
	kind     catalog.BackupType
	backupID uuid.UUID
}

func (s *Scheduler) evaluateConfig(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	backupConfig config.PhysicalBackupConfig,
) {
	logger = logger.With(slog.Any("database_id", backupConfig.DatabaseID))

	if backupConfig.StorageID == nil {
		logger.ErrorContext(ctx, "physical backup config has no storage id; skipping")

		return
	}

	decision, ok := s.decideBackupKind(ctx, logger, now, backupConfig)
	if !ok {
		return
	}

	s.scheduleBackup(ctx, logger, backupConfig, decision)
}

func (s *Scheduler) decideBackupKind(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	backupConfig config.PhysicalBackupConfig,
) (backupDecision, bool) {
	state, err := s.loadSchedulingState(ctx, logger, now, backupConfig)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load physical backup scheduling state", slog.Any("error", err))
		return backupDecision{}, false
	}
	choice := decidePhysicalBackup(state, backupConfig, now)
	if choice.shouldClearIncrementalRequest {
		s.clearIncrementalRequest(ctx, logger, backupConfig)
	}
	return choice.decision, choice.canSchedule
}

func (s *Scheduler) clearIncrementalRequest(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) {
	if err := s.configs.ClearIncrementalBackupRequest(ctx,
		backupConfig.DatabaseID, backupConfig.ForceIncrementalRequestedAt,
	); err != nil {
		logger.ErrorContext(ctx, "failed to clear incremental backup request", slog.Any("error", err))
	}
}

func (s *Scheduler) resolveIncrParent(ctx context.Context, rootFullBackupID uuid.UUID) (*uuid.UUID, error) {
	parent, _, err := s.incrRepo.FindLatestCompletedByRootFull(ctx, rootFullBackupID)
	if err != nil {
		return nil, err
	}

	if parent == nil {
		return nil, nil
	}

	return &parent.ID, nil
}

func (s *Scheduler) scheduleBackup(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
	decision backupDecision,
) {
	backupID := uuid.New()
	run := s.backuper.registerRun(backupID)
	isDispatched := false
	defer func() {
		if !isDispatched {
			run.Finish()
		}
	}()
	logger = logger.With(slog.Any("backup_id", backupID), slog.Any("backup_kind", decision.kind))

	claimed, err := s.claimAndInsert(ctx, backupConfig, backupID, decision)
	if err != nil {
		logger.ErrorContext(ctx, "failed to claim and insert backup row", slog.Any("error", err))

		return
	}

	if !claimed {
		logger.DebugContext(ctx, "in-flight slot already claimed by another instance; skipping")

		return
	}

	if decision.forceFullRequestedAt != nil {
		if err := s.configs.ClearFullBackupRequest(ctx,
			backupConfig.DatabaseID,
			decision.forceFullRequestedAt,
		); err != nil {
			logger.ErrorContext(ctx, "failed to clear forced full request", slog.Any("error", err))
		}
	}

	if decision.forceIncrementalRequestedAt != nil {
		if err := s.configs.ClearIncrementalBackupRequest(ctx,
			backupConfig.DatabaseID,
			decision.forceIncrementalRequestedAt,
		); err != nil {
			logger.ErrorContext(ctx, "failed to clear forced incremental request", slog.Any("error", err))
		}
	}

	s.backuper.dispatch(run, true)
	isDispatched = true

	logger.InfoContext(
		ctx,
		"scheduled physical backup", slog.Any("backup_kind", decision.kind), slog.Any("reason", decision.reason),
	)
}

func decideImmediateReplacementFull(
	lastFull *catalog.FullBackup,
	lastIncr *catalog.IncrementalBackup,
) (backupDecision, bool) {
	if lastFull != nil && lastFull.Status == catalog.BackupStatusError &&
		hasFullBackupErrorReason(lastFull, catalog.BackupErrorFailoverDuringBackup) {
		return backupDecision{
			kind:   catalog.BackupTypeFull,
			reason: "replacing FULL interrupted by failover",
			immediateTrigger: &backupTrigger{
				kind:     catalog.BackupTypeFull,
				backupID: lastFull.ID,
			},
		}, true
	}

	if !hasIncrementalBackupErrorReason(lastIncr, catalog.BackupErrorTimelineSwitchDetected) ||
		lastIncr.Status != catalog.BackupStatusChainBroken ||
		lastIncr.CompletedAt == nil {
		return backupDecision{}, false
	}

	if lastFull != nil && lastFull.CreatedAt.After(*lastIncr.CompletedAt) {
		return backupDecision{}, false
	}

	return backupDecision{
		kind:   catalog.BackupTypeFull,
		reason: "re-anchoring after timeline switch",
		immediateTrigger: &backupTrigger{
			kind:     catalog.BackupTypeIncremental,
			backupID: lastIncr.ID,
		},
	}, true
}

func hasFullBackupErrorReason(
	full *catalog.FullBackup,
	reason catalog.BackupErrorReason,
) bool {
	return full != nil && full.ErrorReason != nil && *full.ErrorReason == reason
}

func hasIncrementalBackupErrorReason(
	incremental *catalog.IncrementalBackup,
	reason catalog.BackupErrorReason,
) bool {
	return incremental != nil && incremental.ErrorReason != nil && *incremental.ErrorReason == reason
}

func (s *Scheduler) claimAndInsert(
	ctx context.Context,
	backupConfig config.PhysicalBackupConfig,
	backupID uuid.UUID,
	decision backupDecision,
) (bool, error) {
	now := s.clock.Now()
	claimed := false

	txErr := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.inFlightRepo.AcquireBackupAndOrphanCleanupLock(ctx, backupConfig.DatabaseID); err != nil {
			return err
		}

		ok, claimErr := s.inFlightRepo.Claim(ctx, catalog.ClaimSpec{
			DatabaseID: backupConfig.DatabaseID,
			BackupType: decision.kind,
			BackupID:   backupID,
		}, now)
		if claimErr != nil {
			return claimErr
		}

		if !ok {
			return nil
		}

		if decision.forceFullRequestedAt == nil {
			hasUnresolvedMismatch, err := s.repository.HasUnresolvedSystemIdentifierMismatch(
				ctx,
				backupConfig.DatabaseID,
			)
			if err != nil {
				return err
			}
			if hasUnresolvedMismatch {
				return errBackupDecisionSuperseded
			}
		}

		if decision.immediateTrigger != nil {
			isCurrent, err := s.isImmediateReplacementTriggerCurrent(
				ctx,
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

		if decision.kind == catalog.BackupTypeFull {
			return s.fullRepo.Create(ctx, &catalog.FullBackup{
				ID:         backupID,
				DatabaseID: backupConfig.DatabaseID,
				StorageID:  *backupConfig.StorageID,
				Status:     catalog.BackupStatusInProgress,
				CreatedAt:  now,
			})
		}

		return s.incrRepo.Create(ctx, &catalog.IncrementalBackup{
			ID:                        backupID,
			DatabaseID:                backupConfig.DatabaseID,
			StorageID:                 *backupConfig.StorageID,
			RootFullBackupID:          decision.incrRootFullBackupID,
			ParentIncrementalBackupID: decision.incrParentIncrID,
			Status:                    catalog.BackupStatusInProgress,
			CreatedAt:                 now,
		})
	})
	if errors.Is(txErr, errBackupDecisionSuperseded) {
		return false, nil
	}
	if txErr != nil {
		return false, txErr
	}

	return claimed, nil
}

func (s *Scheduler) isImmediateReplacementTriggerCurrent(
	ctx context.Context,
	databaseID uuid.UUID,
	trigger backupTrigger,
) (bool, error) {
	hasMismatch, err := s.repository.HasUnresolvedSystemIdentifierMismatch(ctx, databaseID)
	if err != nil || hasMismatch {
		return false, err
	}
	lastFull, _, err := s.fullRepo.FindLastFullAnyStatusByDatabase(ctx, databaseID)
	if err != nil {
		return false, err
	}
	if trigger.kind == catalog.BackupTypeFull {
		return lastFull != nil && lastFull.ID == trigger.backupID && lastFull.Status == catalog.BackupStatusError &&
			hasFullBackupErrorReason(lastFull, catalog.BackupErrorFailoverDuringBackup), nil
	}
	lastIncremental, _, err := s.incrRepo.FindLastByDatabase(ctx, databaseID)
	if err != nil {
		return false, err
	}
	if lastIncremental == nil || lastIncremental.ID != trigger.backupID || lastIncremental.CompletedAt == nil ||
		lastIncremental.Status != catalog.BackupStatusChainBroken ||
		!hasIncrementalBackupErrorReason(lastIncremental, catalog.BackupErrorTimelineSwitchDetected) {
		return false, nil
	}
	return lastFull == nil || !lastFull.CreatedAt.After(*lastIncremental.CompletedAt), nil
}

func (s *Scheduler) failClaimlessInProgressBackups(
	ctx context.Context,
	logger *slog.Logger,
	claimedBackupIDs map[uuid.UUID]struct{},
) error {
	fulls, err := s.fullRepo.FindAllInProgress(ctx)
	if err != nil {
		return err
	}

	for _, full := range fulls {
		if _, ok := claimedBackupIDs[full.ID]; ok {
			continue
		}

		logger.WarnContext(
			ctx,
			"found an in-progress full backup with no in-flight claim",
			slog.Any("backup_id", full.ID),
			slog.Any("database_id", full.DatabaseID),
		)

		s.failOrphanedBackup(ctx, logger, orphanedBackupSpec{
			Kind:             catalog.BackupTypeFull,
			BackupID:         full.ID,
			DatabaseID:       full.DatabaseID,
			StorageID:        full.StorageID,
			FileName:         valueOrEmpty(full.FileName),
			ManifestFileName: valueOrEmpty(full.ManifestFileName),
		})
	}

	incrementals, err := s.incrRepo.FindAllInProgress(ctx)
	if err != nil {
		return err
	}

	for _, incremental := range incrementals {
		if _, ok := claimedBackupIDs[incremental.ID]; ok {
			continue
		}

		logger.WarnContext(
			ctx,
			"found an in-progress incremental backup with no in-flight claim",
			slog.Any("backup_id", incremental.ID),
			slog.Any("database_id", incremental.DatabaseID),
		)

		s.failOrphanedBackup(ctx, logger, orphanedBackupSpec{
			Kind:             catalog.BackupTypeIncremental,
			BackupID:         incremental.ID,
			DatabaseID:       incremental.DatabaseID,
			StorageID:        incremental.StorageID,
			FileName:         valueOrEmpty(incremental.FileName),
			ManifestFileName: valueOrEmpty(incremental.ManifestFileName),
		})
	}

	return nil
}

func (s *Scheduler) describeOrphan(
	ctx context.Context,
	logger *slog.Logger,
	claim *catalog.InFlightBackup,
) orphanedBackupSpec {
	spec := orphanedBackupSpec{
		Kind:       claim.BackupType,
		BackupID:   claim.BackupID,
		DatabaseID: claim.DatabaseID,
	}

	if claim.BackupType == catalog.BackupTypeIncremental {
		row, _, err := s.incrRepo.FindByID(ctx, claim.BackupID)
		if err != nil || row == nil {
			logger.WarnContext(
				ctx,
				"could not read an orphaned incremental backup row",
				slog.Any("backup_id", claim.BackupID),
			)

			return spec
		}

		spec.StorageID = row.StorageID
		spec.FileName = valueOrEmpty(row.FileName)
		spec.ManifestFileName = valueOrEmpty(row.ManifestFileName)

		return spec
	}

	row, _, err := s.fullRepo.FindByID(ctx, claim.BackupID)
	if err != nil || row == nil {
		logger.WarnContext(ctx, "could not read an orphaned full backup row", slog.Any("backup_id", claim.BackupID))

		return spec
	}

	spec.StorageID = row.StorageID
	spec.FileName = valueOrEmpty(row.FileName)
	spec.ManifestFileName = valueOrEmpty(row.ManifestFileName)

	return spec
}

type orphanedBackupSpec struct {
	Kind             catalog.BackupType
	BackupID         uuid.UUID
	DatabaseID       uuid.UUID
	StorageID        uuid.UUID
	FileName         string
	ManifestFileName string
}

func (s *Scheduler) failOrphanedBackup(
	ctx context.Context,
	logger *slog.Logger,
	spec orphanedBackupSpec,
) {
	logger = logger.With(slog.Any("backup_id", spec.BackupID), slog.Any("database_id", spec.DatabaseID))

	s.taskCancellationRegistry.Cancel(spec.BackupID)

	if err := s.failBackupAndReleaseClaim(ctx, spec,
		catalog.BackupErrorApplicationRestart,
		"Backup was interrupted by an application restart and marked failed. Trigger a new backup.",
	); err != nil {
		logger.ErrorContext(ctx, "failed to fail orphaned backup on restart", slog.Any("error", err))

		return
	}

	logger.InfoContext(ctx, "failed backup orphaned by the previous run", slog.Any("backup_kind", spec.Kind))
}

func (s *Scheduler) failBackupAndReleaseClaim(
	ctx context.Context,
	spec orphanedBackupSpec,
	reason catalog.BackupErrorReason,
	message string,
) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		transitioned, err := s.repository.FailInProgressBackup(
			ctx,
			catalog.BackupFailure{
				Kind:     spec.Kind,
				BackupID: spec.BackupID,
				Reason:   reason,
				Message:  message,
				Now:      s.clock.Now(),
			},
		)
		if err != nil {
			return err
		}
		if transitioned && spec.FileName != "" {
			references := []storages.StoredFileReference{
				{StorageID: spec.StorageID, FileName: spec.FileName},
				{StorageID: spec.StorageID, FileName: spec.FileName + catalog.MetadataSuffix},
			}
			if spec.ManifestFileName != "" {
				references = append(
					references,
					storages.StoredFileReference{StorageID: spec.StorageID, FileName: spec.ManifestFileName},
				)
			}
			if err := s.storages.RequestFileDeletions(ctx, references); err != nil {
				return err
			}
		}
		return s.inFlightRepo.ReleaseOwned(ctx, spec.DatabaseID, spec.BackupID)
	})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func createdAtOrNil(full *catalog.FullBackup) *time.Time {
	if full == nil {
		return nil
	}

	return &full.CreatedAt
}

func newestCreatedAt(
	full *catalog.FullBackup,
	incr *catalog.IncrementalBackup,
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

func isChainBrokenBySummarizerOff(lastIncr *catalog.IncrementalBackup) bool {
	return lastIncr != nil &&
		lastIncr.Status == catalog.BackupStatusChainBroken &&
		lastIncr.ErrorReason != nil &&
		*lastIncr.ErrorReason == catalog.BackupErrorSummarizerOff
}

func hasNoRecentCompletedFull(recentFullBackups []*catalog.FullBackup) bool {
	return !slices.ContainsFunc(recentFullBackups,
		func(fullBackup *catalog.FullBackup) bool {
			return fullBackup.Status == catalog.BackupStatusCompleted
		})
}

func isIncrementalEnabled(backupConfig config.PhysicalBackupConfig) bool {
	return backupConfig.BackupType.IsRequiringWalSummary() && backupConfig.IncrementalBackupSchedule != nil
}
