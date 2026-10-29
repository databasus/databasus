package verifications

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

func (s *Service) getLogicalBackup(
	ctx context.Context,
	backupID uuid.UUID,
) (backups.LogicalBackup, bool, error) {
	foundBackups, err := s.backups.GetLogicalBackups(ctx, []uuid.UUID{backupID})
	if err != nil || len(foundBackups) == 0 {
		return backups.LogicalBackup{}, false, err
	}

	return foundBackups[0], true, nil
}

func (s *Service) pairWithBackups(
	ctx context.Context,
	verifications []*Verification,
) ([]ClaimCandidate, error) {
	backupIDs := make([]uuid.UUID, len(verifications))
	for i, verification := range verifications {
		backupIDs[i] = verification.BackupID
	}

	foundBackups, err := s.backups.GetLogicalBackups(ctx, backupIDs)
	if err != nil {
		return nil, err
	}

	backupByID := make(map[uuid.UUID]backups.LogicalBackup, len(foundBackups))
	for _, backup := range foundBackups {
		backupByID[backup.ID] = backup
	}

	candidates := make([]ClaimCandidate, 0, len(verifications))
	for _, verification := range verifications {
		backup, isFound := backupByID[verification.BackupID]
		if !isFound {
			continue
		}

		candidates = append(candidates, ClaimCandidate{Verification: verification, Backup: backup})
	}

	return candidates, nil
}

func (s *Service) getDatabase(ctx context.Context, id uuid.UUID) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{id})
	if err != nil {
		return databases.Database{}, err
	}
	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}
	return foundDatabases[0], nil
}
