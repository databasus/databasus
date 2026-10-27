package config

import (
	"context"

	"github.com/google/uuid"
)

// LockBackupConfig reads the config for a database transfer and locks its saved row until the caller's
// transaction ends, so no save changes the storage the transfer acts on.
func (s *Service) LockBackupConfig(ctx context.Context, databaseID uuid.UUID) (AnyBackupConfig, error) {
	_, _, err := s.repository.get(ctx, databaseID, forUpdateRowLock)
	if err != nil {
		return nil, err
	}

	return s.ReadBackupConfig(ctx, databaseID)
}

// SwitchStorage points the config at a storage the caller checked, in the caller's transaction. A
// config that was never saved is saved as the default of its method with that storage. The backups in
// the previous storage stay, so no storage-change event is published.
func (s *Service) SwitchStorage(ctx context.Context, databaseID, storageID uuid.UUID) error {
	_, isSaved, err := s.repository.get(ctx, databaseID, forUpdateRowLock)
	if err != nil {
		return err
	}

	if isSaved {
		return s.repository.updateStorage(ctx, databaseID, storageID)
	}

	defaultConfig, err := s.ReadBackupConfig(ctx, databaseID)
	if err != nil {
		return err
	}

	switch typedConfig := defaultConfig.(type) {
	case LogicalBackupConfig:
		typedConfig.StorageID = &storageID

		return s.repository.upsertLogical(ctx, typedConfig)

	case PhysicalBackupConfig:
		typedConfig.StorageID = &storageID

		return s.repository.upsertPhysical(ctx, typedConfig)

	default:
		return ErrMethodMismatch
	}
}
