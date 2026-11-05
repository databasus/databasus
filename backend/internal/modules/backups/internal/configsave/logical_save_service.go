package configsave

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
)

func (s *Service) saveLogicalBackupConfig(
	ctx context.Context,
	scope storageSwitchScope,
	savedConfig config.LogicalBackupConfig,
) (config.AnyBackupConfig, error) {
	savedConfig.NotificationTriggers = config.NormalizeTriggers(savedConfig.NotificationTriggers)

	err := savedConfig.Validate()
	if err != nil {
		return nil, err
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.databases.LockDatabaseForBackup(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		if err := s.repository.LockDatabaseConfig(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		previousConfig, isSaved, err := s.repository.GetBackupConfig(
			ctx,
			savedConfig.DatabaseID,
			config.ForUpdateRowLock,
		)
		if err != nil {
			return err
		}

		err = s.checkStorageInWorkspace(ctx, savedConfig.StorageID, scope.WorkspaceID)
		if err != nil {
			return err
		}

		var previousStorageID *uuid.UUID
		if isSaved {
			previousStorageID = previousConfig.StorageID
		}

		hasReassignedStorage, err := s.requestStorageChangeCleanup(
			ctx,
			storageSwitchSave{Scope: scope, PreviousStorageID: previousStorageID, Config: savedConfig},
		)
		if err != nil {
			return err
		}

		if err := s.repository.UpsertLogical(ctx, savedConfig); err != nil {
			return err
		}

		if hasReassignedStorage {
			return s.repository.IncrementStorageGeneration(ctx, savedConfig.DatabaseID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedConfig, nil
}
