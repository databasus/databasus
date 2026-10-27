package verification_runs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	databases_bridge "databasus-backend/internal/features/databases"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

const jobName = "verification_scheduler"

var (
	schedulerTickInterval = 15 * time.Second
	maxPendingDuration    = 24 * time.Hour
)

type VerificationScheduler struct {
	repo                     *VerificationRepository
	service                  *VerificationService
	verificaionConfigService *verification_config.VerificationConfigService
	agentService             *verification_agents.AgentService
	databaseAccess           *databases_bridge.Access
	// getBackups and getDatabases are read per call: each test application installs its own modules.
	getBackups   func() backups.Module
	getDatabases func() databases.Module
	logger       *slog.Logger

	hasRun atomic.Bool
}

func (s *VerificationScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	ticker := time.NewTicker(schedulerTickInterval)
	defer ticker.Stop()

	lifecycleLogger := s.logger.With("job_name", jobName)

	lifecycleLogger.InfoContext(ctx, "verification scheduler started")

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "verification scheduler stopped")

			return
		case <-ticker.C:
			tickLogger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

			if err := s.createScheduledRuns(ctx, tickLogger); err != nil {
				tickLogger.ErrorContext(ctx, "failed to create scheduled verification runs", "error", err)
			}

			if err := s.reapStaleRuns(ctx, tickLogger); err != nil {
				tickLogger.ErrorContext(ctx, "failed to reap stale verification runs", "error", err)
			}

			if err := s.cancelAutomaticVerificationsForDisabledSchedules(ctx, tickLogger); err != nil {
				tickLogger.ErrorContext(ctx, "failed to cancel verifications for disabled configs", "error", err)
			}
		}
	}
}

func (s *VerificationScheduler) createScheduledRuns(ctx context.Context, logger *slog.Logger) error {
	configsWithEnabledVerifications, err := s.verificaionConfigService.ListEnabled()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, config := range configsWithEnabledVerifications {
		if err := s.createScheduledRunForConfig(ctx, logger, config, now); err != nil {
			logger.ErrorContext(ctx,
				"failed to evaluate scheduled run for database",
				"error", err,
				"database_id", config.DatabaseID,
			)
		}
	}

	return nil
}

func (s *VerificationScheduler) createScheduledRunForConfig(
	ctx context.Context,
	logger *slog.Logger,
	config *verification_config.BackupVerificationConfig,
	now time.Time,
) error {
	// After-backup configs are driven by the backup-completion listener
	if config.ScheduleType == verification_config.VerificationScheduleAfterBackup {
		return nil
	}

	existing, err := s.repo.FindByDatabaseID(config.DatabaseID)
	if err != nil {
		return err
	}

	for _, row := range existing {
		if row.Trigger != VerificationTriggerScheduled {
			continue
		}

		if row.Status == VerificationStatusPending || row.Status == VerificationStatusRunning {
			return nil
		}
	}

	lastFinishedAt, err := s.repo.FindLatestFinishedAt(config.DatabaseID)
	if err != nil {
		return err
	}

	if !config.VerificationInterval.ShouldTriggerBackup(now, lastFinishedAt) {
		return nil
	}

	backup, isFound, err := s.getBackups().GetLatestCompletedLogicalBackup(ctx, config.DatabaseID)
	if err != nil {
		return err
	}

	if !isFound {
		logger.DebugContext(ctx, "skipping scheduled verification: no verifiable backup yet",
			"database_id", config.DatabaseID)

		return nil
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, config.DatabaseID)
	if err != nil {
		return err
	}

	if _, err := readAssignedDatabase(ctx, s.getDatabases(), database); err != nil {
		if !errors.Is(err, errDatabaseNotVerifiable) {
			return err
		}

		logger.DebugContext(ctx, "skipping scheduled verification: database not verifiable",
			"database_id", config.DatabaseID, "error", err)

		return nil
	}

	verification := &RestoreVerification{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		BackupID:     backup.ID,
		Trigger:      VerificationTriggerScheduled,
		Status:       VerificationStatusPending,
		AttemptCount: 1,
		CreatedAt:    now,
	}

	return s.repo.Create(verification)
}

func (s *VerificationScheduler) reapStaleRuns(ctx context.Context, logger *slog.Logger) error {
	runningRows, err := s.repo.FindAllRunning()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	staleBefore := now.Add(-StaleAgentThreshold)

	for _, row := range runningRows {
		if row.AgentID == nil {
			continue
		}

		agent, lookupErr := s.agentService.GetAgentByID(*row.AgentID)
		if lookupErr != nil {
			logger.ErrorContext(ctx,
				"failed to load agent during reap",
				"error", lookupErr,
				"verification_id", row.ID,
				"agent_id", row.AgentID,
			)

			continue
		}

		if agent == nil || agent.IsDeleted() {
			// The owning agent is gone, but Requeue clears agent_id so a
			// different agent can pick the run up. If none does, the stale
			// PENDING reaper retires it after maxPendingDuration.
			if failErr := s.service.RequeueOrFail(ctx, row, FailureReasonAgentRemoved, nil); failErr != nil {
				logger.ErrorContext(ctx,
					"failed to requeue or fail RUNNING verification for removed agent",
					"error", failErr,
					"verification_id", row.ID,
				)
			}

			continue
		}

		if agent.LastSeenAt == nil || agent.LastSeenAt.Before(staleBefore) {
			if failErr := s.service.RequeueOrFail(ctx, row, FailureReasonAgentLostContact, nil); failErr != nil {
				logger.ErrorContext(ctx,
					"failed to requeue or fail stale RUNNING verification",
					"error", failErr,
					"verification_id", row.ID,
				)
			}
		}
	}

	stalePendingBefore := now.Add(-maxPendingDuration)

	stalePending, err := s.repo.FindStalePending(stalePendingBefore)
	if err != nil {
		return err
	}

	for _, row := range stalePending {
		if failErr := s.service.RequeueOrFail(ctx, row, FailureReasonUnclaimedTooLong, nil); failErr != nil {
			logger.ErrorContext(ctx,
				"failed to fail stale PENDING verification",
				"error", failErr,
				"verification_id", row.ID,
			)
		}
	}

	return nil
}

// Disabling a schedule is user initiated, so cancellation does not send a failure notification.
func (s *VerificationScheduler) cancelAutomaticVerificationsForDisabledSchedules(
	ctx context.Context,
	logger *slog.Logger,
) error {
	rows, err := s.repo.GetNonTerminalAutomaticVerificationsForDisabledSchedules()
	if err != nil {
		return err
	}

	if len(rows) > 0 {
		logger.InfoContext(ctx, fmt.Sprintf(
			"cancelling %d verifications whose config was disabled", len(rows)))
	}

	for _, row := range rows {
		if cancelErr := s.repo.MarkTerminal(nil, row.ID, VerificationStatusCanceled, map[string]any{
			"fail_message": cancelMessageScheduleDisabled,
		}); cancelErr != nil {
			logger.Error(
				"failed to mark verification CANCELED",
				"error", cancelErr,
				"verification_id", row.ID,
			)
		}
	}

	return nil
}
