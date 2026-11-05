package config

import (
	"context"

	"databasus-backend/internal/lib/database"
)

func (s *Service) CheckStoragePublication(ctx context.Context, publication StoragePublication) error {
	if err := database.RequireTransaction(ctx); err != nil {
		return err
	}

	if err := s.databases.LockDatabaseForBackup(ctx, publication.DatabaseID); err != nil {
		return err
	}
	if err := s.repository.LockDatabaseConfig(ctx, publication.DatabaseID); err != nil {
		return err
	}

	current, isFound, err := s.repository.GetBackupConfig(ctx, publication.DatabaseID, ForShareRowLock)
	if err != nil {
		return err
	}

	if !isFound || current.StorageGeneration != publication.Generation || current.StorageID == nil ||
		*current.StorageID != publication.StorageID {
		return ErrStaleStoragePublication
	}
	return nil
}
