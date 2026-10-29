package attempts

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/databases"
)

func (s *Service) GetJobs() []background.Job {
	return []background.Job{{Name: "healthcheck_probe", Interval: time.Minute, Run: s.runPass}}
}

func (s *Service) runPass(ctx context.Context, now time.Time) error {
	dueConfigs, err := s.configs.ListDue(ctx, now)
	if err != nil {
		return err
	}

	databaseIDs := make([]uuid.UUID, len(dueConfigs))
	for index, config := range dueConfigs {
		databaseIDs[index] = config.DatabaseID
	}

	dueDatabases, err := s.databases.GetDatabases(ctx, databaseIDs)
	if err != nil {
		return err
	}

	group := background.New(background.Deps{Logger: s.logger})
	slots := make(chan struct{}, 32)
probes:
	for _, userDatabase := range dueDatabases {
		if ctx.Err() != nil {
			break
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			break probes
		}

		probeContext := logging.With(ctx, slog.String("database_id", userDatabase.ID.String()))
		group.Go(probeContext, "healthcheck_probe", func(ctx context.Context) error {
			defer func() { <-slots }()
			return s.checkDatabase(ctx, userDatabase.ID)
		})
	}

	return group.Wait(context.WithoutCancel(ctx))
}

func (s *Service) checkDatabase(ctx context.Context, databaseID uuid.UUID) error {
	probeError := s.databases.ProbeLiveness(ctx, databaseID)
	if ctx.Err() != nil || errors.Is(probeError, databases.ErrDatabaseNotFound) {
		return nil
	}

	status := databases.HealthStatusAvailable
	if probeError != nil {
		status = databases.HealthStatusUnavailable
	}

	now := s.clock.Now()
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		userDatabase, isFound, err := s.databases.GetDatabaseForUpdate(ctx, databaseID)
		if err != nil {
			return err
		}

		if !isFound {
			return databases.ErrDatabaseNotFound
		}

		config, err := s.configs.GetCurrentConfig(ctx, databaseID)
		if err != nil {
			return err
		}
		if !config.IsHealthcheckEnabled {
			return nil
		}

		attempt := Attempt{ID: uuid.New(), DatabaseID: config.DatabaseID, Status: status, CreatedAt: now}
		if err := s.repository.insert(ctx, attempt); err != nil {
			return err
		}

		recentAttempts, err := s.repository.list(
			ctx,
			ListInput{
				DatabaseID: config.DatabaseID,
				Since:      time.Time{},
				Limit:      config.FailedAttemptsBeforeUnavailable,
				After:      nil,
			},
			nil,
		)
		if err != nil {
			return err
		}

		transition := getTransition(
			TransitionInput{
				PreviousStatus:                  userDatabase.HealthStatus,
				Status:                          status,
				FailedAttemptsBeforeUnavailable: config.FailedAttemptsBeforeUnavailable,
				RecentAttempts:                  recentAttempts,
			},
		)
		if transition != nil {
			if err := s.databases.SetHealthStatus(ctx, config.DatabaseID, transition); err != nil {
				return err
			}

			if config.IsNotificationEnabled {
				if err := database.AfterCommit(
					ctx,
					func(ctx context.Context) { s.notify(ctx, userDatabase, *transition) },
				); err != nil {
					return err
				}
			}
		}

		return s.repository.deleteBefore(ctx, config.DatabaseID, now.AddDate(0, 0, -config.AttemptRetentionDays))
	})
	if errors.Is(err, databases.ErrDatabaseNotFound) || database.IsForeignKeyViolation(err) {
		return nil
	}

	return err
}
