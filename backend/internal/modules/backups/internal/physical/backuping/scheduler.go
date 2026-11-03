package backuping

import (
	"context"
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
)

type Scheduler struct {
	lifecycle        *catalog.BackupLifecycle
	backupRepo       *catalog.BackupRepository
	repository       *catalog.Repository
	configs          *config.Service
	chainViewService *chains.Service
	backuper         *Backuper
	logger           *slog.Logger
	clock            clock.Source
	transactor       *database.Transactor
}

func (s *Scheduler) GetJob() background.Job {
	return background.Job{
		Name:     "backups_physical_scheduler",
		Interval: schedulerInterval,
		Run:      func(ctx context.Context, _ time.Time) error { return s.RunPass(ctx) },
	}
}

// GetRequestJob serves pending backup requests between full scheduler passes. It reads committed
// configs, so every writer of a request and a request left waiting by a running backup need no signal.
func (s *Scheduler) GetRequestJob() background.Job {
	return background.Job{
		Name:     "backups_physical_request_scheduler",
		Interval: requestSchedulerInterval,
		Run:      func(ctx context.Context, _ time.Time) error { return s.runRequestPass(ctx) },
	}
}

func (s *Scheduler) RunPass(ctx context.Context) error {
	enabledConfigs, err := s.configs.ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	return s.evaluateConfigs(ctx, enabledConfigs)
}

func (s *Scheduler) FailBackupsLeftInProgress(
	ctx context.Context,
	_ *slog.Logger,
) error {
	return s.lifecycle.FailLeftInProgress(ctx)
}

func (s *Scheduler) runRequestPass(ctx context.Context) error {
	pendingConfigs, err := s.configs.ListPhysicalBackupConfigsWithPendingRequest(ctx)
	if err != nil {
		return err
	}

	return s.evaluateConfigs(ctx, pendingConfigs)
}

func (s *Scheduler) evaluateConfigs(ctx context.Context, backupConfigs []config.PhysicalBackupConfig) error {
	if len(backupConfigs) == 0 {
		return nil
	}

	databaseIDs := make([]uuid.UUID, 0, len(backupConfigs))
	for _, backupConfig := range backupConfigs {
		databaseIDs = append(databaseIDs, backupConfig.DatabaseID)
	}

	newestStates, err := s.loadNewestStates(ctx, databaseIDs)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	for _, backupConfig := range backupConfigs {
		s.evaluateConfig(ctx, s.logger, configEvaluation{
			now:    now,
			config: backupConfig,
			state:  newestStates[backupConfig.DatabaseID],
		})
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
}

func (s *Scheduler) evaluateConfig(
	ctx context.Context,
	logger *slog.Logger,
	evaluation configEvaluation,
) {
	logger = logger.With(slog.Any("database_id", evaluation.config.DatabaseID))

	if evaluation.config.StorageID == nil {
		logger.ErrorContext(ctx, "physical backup config has no storage id; skipping")

		return
	}

	state := s.addChainFacts(ctx, logger, evaluation)

	choice := decidePhysicalBackup(state, evaluation.config, evaluation.now)
	if choice.shouldClearIncrementalRequest {
		s.clearIncrementalRequest(ctx, logger, evaluation.config)
	}

	if !choice.canSchedule {
		return
	}

	s.scheduleBackup(ctx, logger, evaluation.config)
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
	parent, _, err := s.backupRepo.FindLatestCompletedByRootFull(ctx, rootFullBackupID)
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
) {
	backupID := uuid.New()
	run, canStart := s.backuper.registerRun(backupRunSpec{BackupID: backupID, DatabaseID: backupConfig.DatabaseID})
	if !canStart {
		return
	}

	isDispatched := false
	defer func() {
		if !isDispatched {
			run.Finish()
		}
	}()

	logger = logger.With(slog.Any("backup_id", backupID))

	decision, claimed, err := s.claimAndInsert(ctx, backupConfig, backupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to claim and insert backup row", slog.Any("error", err))

		return
	}

	if !claimed {
		logger.DebugContext(ctx, "database already has a running backup; skipping")

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
	lastFull *catalog.Backup,
	lastIncr *catalog.Backup,
) (backupDecision, bool) {
	if lastFull != nil && lastFull.Status == catalog.BackupStatusError &&
		hasFullBackupErrorReason(lastFull, catalog.BackupErrorFailoverDuringBackup) {
		return backupDecision{
			kind:   catalog.BackupTypeFull,
			reason: "replacing FULL interrupted by failover",
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
	}, true
}

func hasFullBackupErrorReason(
	full *catalog.Backup,
	reason catalog.BackupErrorReason,
) bool {
	return full != nil && full.ErrorReason != nil && *full.ErrorReason == reason
}

func hasIncrementalBackupErrorReason(
	incremental *catalog.Backup,
	reason catalog.BackupErrorReason,
) bool {
	return incremental != nil && incremental.ErrorReason != nil && *incremental.ErrorReason == reason
}

func (s *Scheduler) claimAndInsert(
	ctx context.Context,
	backupConfig config.PhysicalBackupConfig,
	backupID uuid.UUID,
) (backupDecision, bool, error) {
	now := s.clock.Now()
	claimed := false
	var decision backupDecision

	txErr := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.backupRepo.AcquireBackupAndOrphanCleanupLock(ctx, backupConfig.DatabaseID); err != nil {
			return err
		}
		if _, err := s.configs.LockBackupConfig(ctx, backupConfig.DatabaseID); err != nil {
			return err
		}

		latestConfig, err := s.configs.GetPhysicalBackupConfig(ctx, backupConfig.DatabaseID)
		if err != nil {
			return err
		}
		if !latestConfig.IsBackupsEnabled || latestConfig.StorageID == nil {
			return nil
		}

		backupConfig = latestConfig
		state, err := s.loadSchedulingState(ctx, now, backupConfig)
		if err != nil {
			return err
		}

		choice := decidePhysicalBackup(state, backupConfig, now)
		if choice.shouldClearIncrementalRequest {
			if err := s.configs.ClearIncrementalBackupRequest(
				ctx,
				backupConfig.DatabaseID,
				backupConfig.ForceIncrementalRequestedAt,
			); err != nil {
				return err
			}
		}

		if !choice.canSchedule {
			return nil
		}
		decision = choice.decision

		backup := &catalog.Backup{
			ID: backupID, DatabaseID: backupConfig.DatabaseID, StorageID: *backupConfig.StorageID,
			Kind: catalog.BackupKind(decision.kind), CreatedAt: now, StorageGeneration: backupConfig.StorageGeneration,
		}
		if decision.kind == catalog.BackupTypeIncremental {
			backup.RootFullBackupID = &decision.incrRootFullBackupID
			backup.ParentBackupID = decision.incrParentIncrID
			if backup.ParentBackupID == nil {
				backup.ParentBackupID = backup.RootFullBackupID
			}
		}

		claimed, err = s.lifecycle.Start(ctx, backup)
		return err
	})
	if txErr != nil {
		return backupDecision{}, false, txErr
	}

	return decision, claimed, nil
}

func createdAtOrNil(full *catalog.Backup) *time.Time {
	if full == nil {
		return nil
	}

	return &full.CreatedAt
}

func newestCreatedAt(
	full *catalog.Backup,
	incr *catalog.Backup,
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

func isChainBrokenBySummarizerOff(lastIncr *catalog.Backup) bool {
	return lastIncr != nil &&
		lastIncr.Status == catalog.BackupStatusChainBroken &&
		lastIncr.ErrorReason != nil &&
		*lastIncr.ErrorReason == catalog.BackupErrorSummarizerOff
}

func hasNoRecentCompletedFull(recentFullBackups []*catalog.Backup) bool {
	return !slices.ContainsFunc(recentFullBackups,
		func(fullBackup *catalog.Backup) bool {
			return fullBackup.Status == catalog.BackupStatusCompleted
		})
}

func isIncrementalEnabled(backupConfig config.PhysicalBackupConfig) bool {
	return backupConfig.BackupType.IsRequiringWalSummary() && backupConfig.IncrementalBackupSchedule != nil
}
