package config

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/storageswitch"
)

func (s *Service) LockBackupConfig(ctx context.Context, databaseID uuid.UUID) (AnyBackupConfig, error) {
	if err := s.repository.lockPhysicalHistory(ctx, databaseID); err != nil {
		return nil, err
	}
	if err := s.databases.LockDatabaseForBackup(ctx, databaseID); err != nil {
		return nil, err
	}
	if err := s.repository.lockDatabaseConfig(ctx, databaseID); err != nil {
		return nil, err
	}
	if _, _, err := s.repository.get(ctx, databaseID, forUpdateRowLock); err != nil {
		return nil, err
	}
	return s.ReadBackupConfig(ctx, databaseID)
}

type StorageSwitchAssignment struct {
	DatabaseID  uuid.UUID
	StorageID   uuid.UUID
	WorkspaceID uuid.UUID
}

func (s *Service) SwitchStorage(
	ctx context.Context,
	assignment StorageSwitchAssignment,
) (*storageswitch.Outcome, error) {
	previous, err := s.LockBackupConfig(ctx, assignment.DatabaseID)
	if err != nil {
		return nil, err
	}
	scope := storageSwitchScope{WorkspaceID: assignment.WorkspaceID, IsTransfer: true}
	var saved SavedBackupConfig

	switch typed := previous.(type) {
	case LogicalBackupConfig:
		typed.StorageID = &assignment.StorageID
		saved, err = s.saveLogicalBackupConfig(ctx, scope, typed)
	case PhysicalBackupConfig:
		typed.StorageID = &assignment.StorageID
		saved, err = s.savePhysicalBackupConfig(ctx, scope, typed)
	default:
		return nil, ErrMethodMismatch
	}
	return saved.StorageSwitch, err
}
