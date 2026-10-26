package healthcheck_attempt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
)

type CheckDatabaseHealthUseCase struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	healthcheckAttemptSender     HealthcheckAttemptSender
	databases                    HealthcheckedDatabases
}

func (uc *CheckDatabaseHealthUseCase) Execute(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
) error {
	database, err := uc.databases.GetDatabaseByID(ctx, healthcheckConfig.DatabaseID)
	if err != nil {
		return err
	}

	isExecuteNewAttempt, err := uc.isReadyForNewAttempt(
		now,
		database,
		healthcheckConfig,
	)
	if err != nil {
		return err
	}

	if !isExecuteNewAttempt {
		return nil
	}

	healthcheckAttempt, err := uc.healthcheckDatabase(ctx, logger, now, database)
	if err != nil {
		return err
	}

	err = uc.healthcheckAttemptRepository.Insert(healthcheckAttempt)
	if err != nil {
		return err
	}

	err = uc.updateDatabaseHealthStatusIfChanged(ctx,
		database,
		healthcheckConfig,
		healthcheckAttempt,
	)
	if err != nil {
		return err
	}

	err = uc.healthcheckAttemptRepository.DeleteOlderThan(
		database.ID,
		time.Now().UTC().Add(-time.Duration(healthcheckConfig.StoreAttemptsDays)*24*time.Hour),
	)
	if err != nil {
		return err
	}

	return nil
}

func (uc *CheckDatabaseHealthUseCase) updateDatabaseHealthStatusIfChanged(
	ctx context.Context,
	database databases.Database,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
	healthcheckAttempt *HealthcheckAttempt,
) error {
	if database.HealthStatus != nil && *database.HealthStatus == healthcheckAttempt.Status {
		return nil
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusUnavailable) &&
		healthcheckAttempt.Status == databases.HealthStatusAvailable {
		err := uc.databases.SetHealthStatus(ctx, database.ID, &healthcheckAttempt.Status)
		if err != nil {
			return err
		}

		uc.sendDbStatusNotification(ctx,
			healthcheckConfig,
			database,
			healthcheckAttempt.Status,
		)
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusAvailable) &&
		healthcheckAttempt.Status == databases.HealthStatusUnavailable {
		if healthcheckConfig.AttemptsBeforeConcideredAsDown > 1 {
			lastHealthcheckAttempts, err := uc.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				healthcheckConfig.AttemptsBeforeConcideredAsDown,
			)
			if err != nil {
				return err
			}

			if len(lastHealthcheckAttempts) < healthcheckConfig.AttemptsBeforeConcideredAsDown {
				return nil
			}

			for _, attempt := range lastHealthcheckAttempts {
				if attempt.Status == databases.HealthStatusAvailable {
					return nil
				}
			}
		}

		err := uc.databases.SetHealthStatus(ctx, database.ID, &healthcheckAttempt.Status)
		if err != nil {
			return err
		}

		uc.sendDbStatusNotification(ctx,
			healthcheckConfig,
			database,
			databases.HealthStatusUnavailable,
		)
	}

	return nil
}

func (uc *CheckDatabaseHealthUseCase) healthcheckDatabase(
	ctx context.Context,
	logger *slog.Logger,
	now time.Time,
	database databases.Database,
) (*HealthcheckAttempt, error) {
	healthStatus := databases.HealthStatusAvailable

	// A database being down is the condition this probe exists to detect, and the caller reports it
	// too, so it is a tolerated finding here rather than something needing attention.
	if err := uc.databases.ProbeLiveness(ctx, database.ID); err != nil {
		healthStatus = databases.HealthStatusUnavailable

		logger.WarnContext(ctx, "database healthcheck probe failed", "error", err)
	}

	attempt := &HealthcheckAttempt{
		ID:         uuid.New(),
		DatabaseID: database.ID,
		Status:     healthStatus,
		CreatedAt:  now,
	}

	return attempt, nil
}

func (uc *CheckDatabaseHealthUseCase) isReadyForNewAttempt(
	now time.Time,
	database databases.Database,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
) (bool, error) {
	lastHealthcheckAttempt, err := uc.healthcheckAttemptRepository.FindLastByDatabaseID(database.ID)
	if err != nil {
		// If no attempts found, it's ready for first attempt
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}

		return false, err
	}

	// Check if enough time has passed since last attempt
	intervalDuration := time.Duration(healthcheckConfig.IntervalMinutes) * time.Minute
	nextAttemptTime := lastHealthcheckAttempt.CreatedAt.Add(intervalDuration)

	return now.After(nextAttemptTime.Add(-1 * time.Second)), nil
}

func (uc *CheckDatabaseHealthUseCase) sendDbStatusNotification(
	ctx context.Context,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
	database databases.Database,
	newHealthStatus databases.HealthStatus,
) {
	if !healthcheckConfig.IsSentNotificationWhenUnavailable {
		return
	}

	notification := notifiers.Notification{
		Category: notifiers.CategoryHealthcheckFailed,
		Heading:  fmt.Sprintf("❌ [%s] DB is unavailable", database.Name),
		Message:  fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name),
	}

	if newHealthStatus == databases.HealthStatusAvailable {
		notification = notifiers.Notification{
			Category: notifiers.CategoryHealthcheckSuccess,
			Heading:  fmt.Sprintf("✅ [%s] DB is online", database.Name),
			Message:  fmt.Sprintf("✅ [%s] DB is back online", database.Name),
		}
	}

	uc.healthcheckAttemptSender.Notify(ctx, database.NotifierIDs, notification)
}
