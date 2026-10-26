package userdatabase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecordBackupSuccess runs in the transaction that publishes the backup, so a publication that rolls
// back never shows. An older time leaves a newer one in place.
func (s *Service) RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error {
	isUpdated, err := s.repository.recordBackupSuccess(ctx, databaseID, backupTime)
	if err != nil {
		return err
	}

	if !isUpdated {
		return ErrDatabaseNotFound
	}

	return nil
}

func (s *Service) RecordBackupFailure(ctx context.Context, databaseID uuid.UUID, message string) error {
	isUpdated, err := s.repository.updateLastBackupErrorMessage(ctx, databaseID, message)
	if err != nil {
		return err
	}

	if !isUpdated {
		return ErrDatabaseNotFound
	}

	return nil
}

func (s *Service) SetHealthStatus(ctx context.Context, databaseID uuid.UUID, status *HealthStatus) error {
	isUpdated, err := s.repository.updateHealthStatus(ctx, databaseID, status)
	if err != nil {
		return err
	}

	if !isUpdated {
		return ErrDatabaseNotFound
	}

	return nil
}
