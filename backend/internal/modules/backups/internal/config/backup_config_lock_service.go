package config

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) LockBackupConfig(ctx context.Context, databaseID uuid.UUID) (AnyBackupConfig, error) {
	if err := s.repository.LockPhysicalHistory(ctx, databaseID); err != nil {
		return nil, err
	}
	if err := s.databases.LockDatabaseForBackup(ctx, databaseID); err != nil {
		return nil, err
	}
	if err := s.repository.LockDatabaseConfig(ctx, databaseID); err != nil {
		return nil, err
	}
	if _, _, err := s.repository.GetBackupConfig(ctx, databaseID, ForUpdateRowLock); err != nil {
		return nil, err
	}
	return s.ReadBackupConfig(ctx, databaseID)
}
