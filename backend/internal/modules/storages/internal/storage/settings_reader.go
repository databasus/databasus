package storage

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

type SettingsReaderDeps struct {
	Repositories Repositories
	Registry     *providers.Registry
}

// SettingsReader loads the provider settings of storages. The file store locates a storage through
// it, so it stands apart from the services that delete storages through that store.
type SettingsReader struct {
	repository *storageRepository
	registry   *providers.Registry
}

func NewSettingsReader(deps SettingsReaderDeps) *SettingsReader {
	return &SettingsReader{repository: deps.Repositories.storages, registry: deps.Registry}
}

// GetSettings answers ErrStorageNotFound for a storage that does not exist.
func (r *SettingsReader) GetSettings(ctx context.Context, storageID uuid.UUID) (providers.Settings, error) {
	foundStorage, isFound, err := r.repository.get(ctx, storageID, noRowLock)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, ErrStorageNotFound
	}

	storageWithSettings, err := r.loadOne(ctx, foundStorage)
	if err != nil {
		return nil, err
	}

	return storageWithSettings.Settings, nil
}

func (r *SettingsReader) loadOne(ctx context.Context, foundStorage Storage) (WithSettings, error) {
	storagesWithSettings, err := r.load(ctx, []Storage{foundStorage})
	if err != nil {
		return WithSettings{}, err
	}

	return storagesWithSettings[0], nil
}

// load reads the settings of the storages with one query per provider type among them, keeping their
// order.
func (r *SettingsReader) load(ctx context.Context, storages []Storage) ([]WithSettings, error) {
	storageIDsByType := map[Type][]uuid.UUID{}
	for _, listedStorage := range storages {
		storageIDsByType[listedStorage.Type] = append(storageIDsByType[listedStorage.Type], listedStorage.ID)
	}

	settingsByStorageID := map[uuid.UUID]providers.Settings{}

	for storageType, storageIDs := range storageIDsByType {
		provider, err := r.registry.GetProvider(storageType)
		if err != nil {
			return nil, err
		}

		settingsOfType, err := provider.GetRepository().ListByStorageIDs(ctx, storageIDs)
		if err != nil {
			return nil, err
		}

		maps.Copy(settingsByStorageID, settingsOfType)
	}

	storagesWithSettings := make([]WithSettings, 0, len(storages))
	for _, listedStorage := range storages {
		settings, isFound := settingsByStorageID[listedStorage.ID]
		if !isFound {
			return nil, fmt.Errorf("%w: %s", errSettingsMissing, listedStorage.ID)
		}

		storagesWithSettings = append(storagesWithSettings, WithSettings{Storage: listedStorage, Settings: settings})
	}

	return storagesWithSettings, nil
}
