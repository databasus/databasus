package logical

import (
	"context"
	"io"

	"github.com/google/uuid"
)

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
