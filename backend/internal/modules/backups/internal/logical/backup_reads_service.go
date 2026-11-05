package logical

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) GetBackups(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error) {
	if len(backupIDs) == 0 {
		return []LogicalBackup{}, nil
	}

	return s.repository.listByIDs(ctx, backupIDs)
}

func (s *Service) GetLatestCompletedBackup(ctx context.Context, databaseID uuid.UUID) (LogicalBackup, bool, error) {
	return s.repository.getLatestCompleted(ctx, databaseID)
}

func (s *Service) GetLatestCompletedBackups(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]LogicalBackup, error) {
	return s.repository.listLatestCompleted(ctx, ids)
}
