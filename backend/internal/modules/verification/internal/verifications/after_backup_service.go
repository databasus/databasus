package verifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/verification/internal/configs"
)

func (s *Service) scheduleVerificationForCompletedBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) error {
	logger := s.logger.With(slog.Any("database_id", databaseID))

	config, err := s.configService.GetConfigWithoutAuth(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("load verification config on backup completion: %w", err)
	}

	if !config.IsAutomaticVerificationEnabled ||
		config.ScheduleType != configs.VerificationScheduleAfterBackup {
		return nil
	}

	verifiableBackup, isFound, err := s.backups.GetLatestCompletedLogicalBackup(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("load latest completed backup: %w", err)
	}

	if !isFound {
		logger.InfoContext(ctx, "skipping after-backup verification: no verifiable backup yet")
		return nil
	}

	database, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("load database for after-backup verification: %w", err)
	}

	if _, err := readAssignedDatabase(ctx, s.databases, database); err != nil {
		if !errors.Is(err, errDatabaseNotVerifiable) {
			return err
		}

		logger.WarnContext(ctx, "skipping after-backup verification: database not verifiable", slog.Any("error", err))

		return nil
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if cancelErr := s.cancelPendingAutoScheduledVerificationsByDatabaseID(ctx, databaseID); cancelErr != nil {
			return cancelErr
		}

		return s.verificationRepository.Create(ctx, &Verification{
			ID:           uuid.New(),
			DatabaseID:   databaseID,
			BackupID:     verifiableBackup.ID,
			Trigger:      VerificationTriggerAfterBackup,
			Status:       VerificationStatusPending,
			AttemptCount: 1,
			CreatedAt:    s.clock.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("enqueue after-backup verification: %w", err)
	}

	return nil
}

func (s *Service) cancelPendingAutoScheduledVerificationsByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
) error {
	existing, err := s.verificationRepository.GetActiveForDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	for _, row := range existing {
		if row.Trigger == VerificationTriggerManual {
			continue
		}

		if row.Status == VerificationStatusPending {
			if cancelErr := s.verificationRepository.MarkTerminal(
				ctx,
				TerminalUpdate{VerificationID: row.ID, Status: VerificationStatusCanceled, Fields: map[string]any{
					"fail_message": cancelMessageSupersededByBackup,
				}, FinishedAt: s.clock.Now()},
			); cancelErr != nil {
				return cancelErr
			}
		}
	}

	return nil
}
