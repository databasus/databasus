package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

// OnStorageTransferring refuses to move a storage while a database outside the target workspace backs
// up into it. A database transferred together with its storage has moved already, so it passes.
func (s *Service) OnStorageTransferring(ctx context.Context, event storages.StorageTransferring) error {
	databaseIDs, err := s.repository.listDatabaseIDsByStorage(ctx, event.StorageID)
	if err != nil {
		return err
	}

	if len(databaseIDs) == 0 {
		return nil
	}

	usingDatabases, err := s.databases.GetDatabases(ctx, databaseIDs)
	if err != nil {
		return err
	}

	for _, usingDatabase := range usingDatabases {
		if usingDatabase.WorkspaceID != event.TargetWorkspaceID {
			return ErrStorageUsedOutsideTargetWorkspace
		}
	}

	return nil
}

// OnDatabaseCopied gives a copied logical database the saved config of the original. A physical copy
// starts without a saved config and reads as the default.
func (s *Service) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	originalConfigs, err := s.repository.listLogical(ctx, []uuid.UUID{event.OriginalDatabaseID})
	if err != nil {
		return err
	}

	if len(originalConfigs) == 0 {
		return nil
	}

	return s.repository.UpsertLogical(ctx, originalConfigs[0].copyFor(event.CopiedDatabaseID))
}

// OnDatabaseBackupTypeChanged fits the saved physical config to the new type in the change's
// transaction, so a config the validation refuses is never left behind. It takes the lock a save takes,
// so a save that runs meanwhile either commits first and is fitted here or waits and sees the new type.
func (s *Service) OnDatabaseBackupTypeChanged(ctx context.Context, event databases.DatabaseBackupTypeChanged) error {
	err := s.repository.LockDatabaseConfig(ctx, event.DatabaseID)
	if err != nil {
		return err
	}

	physicalConfigs, err := s.repository.ListPhysical(ctx, []uuid.UUID{event.DatabaseID})
	if err != nil {
		return err
	}

	if len(physicalConfigs) == 0 {
		return nil
	}

	fittedConfig := physicalConfigs[0].coerceToBackupType(event.NewBackupType)

	err = fittedConfig.Validate()
	if err != nil {
		return fmt.Errorf("fit the backup config to the backup type %s: %w", event.NewBackupType, err)
	}

	err = s.repository.UpsertPhysical(ctx, fittedConfig)
	if err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "backup config fitted to the new backup type",
		slog.Any("database_id", event.DatabaseID),
		slog.Any("old_backup_type", event.OldBackupType),
		slog.Any("new_backup_type", event.NewBackupType),
	)

	return nil
}

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.repository.delete(ctx, event.DatabaseID)
}
