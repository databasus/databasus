package verifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/verification/internal/configs"
)

func (s *Service) GetJobs() []background.Job {
	return []background.Job{{Name: "verification_scheduler", Interval: 15 * time.Second, Run: s.runSchedulerPass}}
}

func (s *Service) runSchedulerPass(ctx context.Context, _ time.Time) error {
	for _, step := range []func(context.Context, *slog.Logger) error{s.createScheduledRuns, s.reapStaleRuns, s.cancelAutomaticVerificationsForDisabledSchedules} {
		if err := step(ctx, s.logger); err != nil {
			s.logger.ErrorContext(ctx, "verification scheduler step failed", slog.Any("error", err))
		}
	}
	return nil
}

func (s *Service) createScheduledRuns(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := s.configService.ListEnabledConfigs(ctx)
	if err != nil {
		return err
	}

	databaseIDs := make([]uuid.UUID, len(enabledConfigs))
	for index, config := range enabledConfigs {
		databaseIDs[index] = config.DatabaseID
	}
	foundDatabases, err := s.databases.GetDatabases(ctx, databaseIDs)
	if err != nil {
		return err
	}
	versions, err := s.databases.GetPostgresqlLogicalVersions(ctx, databaseIDs)
	if err != nil {
		return err
	}
	latestBackups, err := s.backups.GetLatestCompletedLogicalBackups(ctx, databaseIDs)
	if err != nil {
		return err
	}

	databasesByID := make(map[uuid.UUID]databases.Database, len(foundDatabases))
	for _, database := range foundDatabases {
		databasesByID[database.ID] = database
	}

	for _, config := range enabledConfigs {
		database, isFound := databasesByID[config.DatabaseID]
		backup, hasBackup := latestBackups[config.DatabaseID]
		if !isFound || !hasBackup || versions[database.ID] == "" {
			continue
		}
		if err := s.createScheduledRun(ctx, config, backup); err != nil {
			logger.ErrorContext(
				ctx,
				"failed to create scheduled verification",
				slog.Any("database_id", config.DatabaseID),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

func (s *Service) createScheduledRun(ctx context.Context, config configs.Config, backup backups.LogicalBackup) error {
	if config.ScheduleType == configs.VerificationScheduleAfterBackup {
		return nil
	}
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existing, err := s.verificationRepository.GetActiveForDatabase(ctx, config.DatabaseID)
		if err != nil {
			return err
		}
		for _, verification := range existing {
			if verification.Trigger == VerificationTriggerScheduled {
				return nil
			}
		}
		lastFinished, err := s.verificationRepository.GetLatestFinishedAt(ctx, config.DatabaseID)
		if err != nil {
			return err
		}
		if !config.Schedule.IsDue(s.clock.Now(), lastFinished) {
			return nil
		}
		return s.verificationRepository.Create(ctx, s.newPending(backup, VerificationTriggerScheduled))
	})
}

func (s *Service) reapStaleRuns(ctx context.Context, logger *slog.Logger) error {
	runningRows, err := s.verificationRepository.GetRunning(ctx)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	for _, row := range runningRows {
		if row.AgentID == nil {
			continue
		}

		agent, isFound, lookupErr := s.agents.GetAgent(ctx, *row.AgentID)
		if lookupErr != nil {
			logger.ErrorContext(ctx,
				"failed to load agent during reap",
				slog.Any("error", lookupErr),
				slog.Any("verification_id", row.ID),
				slog.Any("agent_id", row.AgentID),
			)

			continue
		}

		if !isFound || agent.IsDeleted() {
			// The owning agent is gone, but Requeue clears agent_id so a
			// different agent can pick the run up. If none does, the stale
			// PENDING reaper retires it after maxPendingDuration.
			if failErr := s.RequeueOrFail(ctx, row, FailureReasonAgentRemoved, nil); failErr != nil {
				logger.ErrorContext(ctx,
					"failed to requeue or fail RUNNING verification for removed agent",
					slog.Any("error", failErr),
					slog.Any("verification_id", row.ID),
				)
			}

			continue
		}

		if !hasFreshAcceptedHeartbeat(&agent, now) {
			if failErr := s.RequeueOrFail(ctx, row, FailureReasonAgentLostContact, nil); failErr != nil {
				logger.ErrorContext(ctx,
					"failed to requeue or fail stale RUNNING verification",
					slog.Any("error", failErr),
					slog.Any("verification_id", row.ID),
				)
			}
		}
	}

	stalePendingBefore := now.Add(-maxPendingDuration)

	stalePending, err := s.verificationRepository.GetStalePending(ctx, stalePendingBefore)
	if err != nil {
		return err
	}

	for _, row := range stalePending {
		if failErr := s.RequeueOrFail(ctx, row, FailureReasonUnclaimedTooLong, nil); failErr != nil {
			logger.ErrorContext(ctx,
				"failed to fail stale PENDING verification",
				slog.Any("error", failErr),
				slog.Any("verification_id", row.ID),
			)
		}
	}

	return nil
}

// Disabling a schedule is user initiated, so cancellation does not send a failure notification.
func (s *Service) cancelAutomaticVerificationsForDisabledSchedules(
	ctx context.Context,
	logger *slog.Logger,
) error {
	rows, err := s.verificationRepository.GetNonTerminalAutomaticVerificationsForDisabledSchedules(ctx)
	if err != nil {
		return err
	}

	if len(rows) > 0 {
		logger.InfoContext(
			ctx,
			"cancelling verifications whose config was disabled",
			slog.Int("verification_count", len(rows)),
		)
	}

	for _, row := range rows {
		if cancelErr := s.verificationRepository.MarkTerminal(
			ctx,
			TerminalUpdate{VerificationID: row.ID, Status: VerificationStatusCanceled, Fields: map[string]any{
				"fail_message": cancelMessageScheduleDisabled,
			}, FinishedAt: s.clock.Now()},
		); cancelErr != nil {
			logger.ErrorContext(ctx,
				"failed to mark verification CANCELED",
				slog.Any("error", cancelErr),
				slog.Any("verification_id", row.ID),
			)
		}
	}

	return nil
}
