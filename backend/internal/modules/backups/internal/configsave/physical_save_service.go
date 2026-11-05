package configsave

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
)

type physicalSave struct {
	Config         config.PhysicalBackupConfig
	PreviousConfig *config.PhysicalBackupConfig
}

func (s *Service) savePhysicalBackupConfig(
	ctx context.Context,
	scope storageSwitchScope,
	savedConfig config.PhysicalBackupConfig,
) (config.AnyBackupConfig, error) {
	savedConfig.NotificationTriggers = config.NormalizeTriggers(savedConfig.NotificationTriggers)

	if err := savedConfig.Validate(); err != nil {
		return nil, err
	}

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		save, err := s.preparePhysicalSave(ctx, savedConfig)
		if err != nil {
			return err
		}

		if err := s.checkStorageInWorkspace(ctx, save.Config.StorageID, scope.WorkspaceID); err != nil {
			return err
		}

		hasReassignedStorage, err := s.requestStorageChangeCleanup(ctx, storageSwitchSave{
			Scope: scope, PreviousStorageID: save.GetPreviousStorageID(), Config: save.Config,
		})
		if err != nil {
			return err
		}

		if err := s.persistPhysicalSave(ctx, save, hasReassignedStorage); err != nil {
			return err
		}

		savedConfig = save.Config

		return s.registerPhysicalSaveReaction(ctx, save)
	})
	if err != nil {
		return nil, err
	}

	return savedConfig, nil
}

func (s *Service) preparePhysicalSave(
	ctx context.Context,
	savedConfig config.PhysicalBackupConfig,
) (physicalSave, error) {
	if err := s.repository.LockPhysicalHistory(ctx, savedConfig.DatabaseID); err != nil {
		return physicalSave{}, err
	}
	if err := s.databases.LockDatabaseForBackup(ctx, savedConfig.DatabaseID); err != nil {
		return physicalSave{}, err
	}
	if err := s.repository.LockDatabaseConfig(ctx, savedConfig.DatabaseID); err != nil {
		return physicalSave{}, err
	}
	if _, _, err := s.repository.GetBackupConfig(ctx, savedConfig.DatabaseID, config.ForUpdateRowLock); err != nil {
		return physicalSave{}, err
	}

	// A type change takes these locks too, so validation uses the type committed with this save.
	backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, []uuid.UUID{savedConfig.DatabaseID})
	if err != nil {
		return physicalSave{}, err
	}

	savedConfig.BackupType = backupTypes[savedConfig.DatabaseID]
	if err := savedConfig.Validate(); err != nil {
		return physicalSave{}, err
	}

	previousConfigs, err := s.repository.ListPhysical(ctx, []uuid.UUID{savedConfig.DatabaseID})
	if err != nil {
		return physicalSave{}, err
	}

	save := physicalSave{Config: savedConfig, PreviousConfig: nil}
	if len(previousConfigs) > 0 {
		save.PreviousConfig = &previousConfigs[0]
		save.PreviousConfig.BackupType = savedConfig.BackupType
		save.Config.ForceFullRequestedAt = save.PreviousConfig.ForceFullRequestedAt
		save.Config.ForceIncrementalRequestedAt = save.PreviousConfig.ForceIncrementalRequestedAt
	}

	return save, nil
}

func (s *Service) persistPhysicalSave(ctx context.Context, save physicalSave, hasReassignedStorage bool) error {
	if err := s.repository.UpsertPhysical(ctx, save.Config); err != nil {
		return err
	}

	if hasReassignedStorage || !isSameStorage(save.GetPreviousStorageID(), save.Config.StorageID) {
		if err := s.repository.IncrementStorageGeneration(ctx, save.Config.DatabaseID); err != nil {
			return err
		}
	}

	if hasReassignedStorage {
		if err := s.databases.ResetBackupHistory(ctx, save.Config.DatabaseID); err != nil {
			return err
		}

		return s.repository.ResetPhysicalRequests(ctx, config.PhysicalRequestsReset{
			DatabaseID: save.Config.DatabaseID, RequestedAt: s.clock.Now(),
		})
	}

	return nil
}

// The reaction waits for the outermost commit, including a workspace transfer's transaction.
func (s *Service) registerPhysicalSaveReaction(ctx context.Context, save physicalSave) error {
	return database.AfterCommit(ctx, func(ctx context.Context) {
		if err := s.physicalBackups.ReactToSavedConfig(ctx, save.PreviousConfig, save.Config); err != nil {
			s.logger.ErrorContext(ctx, "failed to react to saved physical config", slog.Any("error", err))
		}
	})
}

func (s physicalSave) GetPreviousStorageID() *uuid.UUID {
	if s.PreviousConfig == nil {
		return nil
	}

	return s.PreviousConfig.StorageID
}
