package configsave

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
)

type StorageSwitchAssignment struct {
	DatabaseID  uuid.UUID
	StorageID   uuid.UUID
	WorkspaceID uuid.UUID
}

func (s *Service) SwitchStorage(
	ctx context.Context,
	assignment StorageSwitchAssignment,
) error {
	previous, err := s.configs.LockBackupConfig(ctx, assignment.DatabaseID)
	if err != nil {
		return err
	}
	scope := storageSwitchScope{WorkspaceID: assignment.WorkspaceID, IsTransfer: true}

	switch typed := previous.(type) {
	case config.LogicalBackupConfig:
		typed.StorageID = &assignment.StorageID
		_, err = s.saveLogicalBackupConfig(ctx, scope, typed)
	case config.PhysicalBackupConfig:
		typed.StorageID = &assignment.StorageID
		_, err = s.savePhysicalBackupConfig(ctx, scope, typed)
	default:
		return config.ErrMethodMismatch
	}
	return err
}

// checkStorageInWorkspace answers a storage that does not exist like one of another workspace, so a
// save reveals nothing about storages outside the database's workspace.
func (s *Service) checkStorageInWorkspace(ctx context.Context, storageID *uuid.UUID, workspaceID uuid.UUID) error {
	if storageID == nil {
		return nil
	}

	configStorage, isFound, err := s.storages.GetStorageBlockingTransfer(ctx, *storageID)
	if err != nil {
		return err
	}

	if !isFound || configStorage.WorkspaceID != workspaceID {
		return config.ErrStorageNotInDatabaseWorkspace
	}

	return nil
}

func isSameStorage(first, second *uuid.UUID) bool {
	if first == nil || second == nil {
		return first == second
	}

	return *first == *second
}
