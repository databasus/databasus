package logical

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// These reads serve verification and telemetry.
// They check no access, because their callers act for the system, not for a user.

func (s *Service) GetBackups(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error) {
	if len(backupIDs) == 0 {
		return []LogicalBackup{}, nil
	}

	return s.repository.listByIDs(ctx, backupIDs)
}

func (s *Service) GetLatestCompletedBackup(ctx context.Context, databaseID uuid.UUID) (LogicalBackup, bool, error) {
	return s.repository.getLatestCompleted(ctx, databaseID)
}

func (s *Service) HasCompletedBackupSince(ctx context.Context, databaseID uuid.UUID, since time.Time) (bool, error) {
	return s.repository.hasCompletedSince(ctx, databaseID, since)
}

// OpenBackupFileByID opens the decrypted file of the backup for a verification run.
func (s *Service) OpenBackupFileByID(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error) {
	backup, isFound, err := s.repository.get(ctx, backupID)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, ErrBackupNotFound
	}

	return s.OpenBackupFile(ctx, backup)
}

// SetVerificationStatus records the outcome of the last verification run of the backup. A backup
// deleted meanwhile has nothing to record it on.
func (s *Service) SetVerificationStatus(ctx context.Context, backupID uuid.UUID, status VerificationStatus) error {
	_, err := s.repository.updateVerificationStatus(ctx, backupID, status)

	return err
}

func (s *Service) GetLatestCompletedBackups(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]LogicalBackup, error) {
	return s.repository.listLatestCompleted(ctx, ids)
}
