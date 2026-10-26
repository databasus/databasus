package files

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

// FileProvider is the part of a provider the file store uses.
type FileProvider interface {
	SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error
	OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, file providers.FileAccess) error
}

// LocatedStorage is the provider of a storage and the location it works at.
type LocatedStorage struct {
	Provider FileProvider
	Location providers.Location
}

type StorageLocator interface {
	LocateStorage(ctx context.Context, storageID uuid.UUID) (LocatedStorage, error)
}

// SettingsReader reads the settings of a storage; the parent repository implements it.
type SettingsReader interface {
	GetSettings(ctx context.Context, storageID uuid.UUID) (providers.Settings, error)
}

type LocatorDeps struct {
	Registry *providers.Registry
	Settings SettingsReader
}

// Locator finds the provider of a storage through the registry.
type Locator struct {
	registry *providers.Registry
	settings SettingsReader
}

func NewLocator(deps LocatorDeps) *Locator {
	return &Locator{registry: deps.Registry, settings: deps.Settings}
}

func (l *Locator) LocateStorage(ctx context.Context, storageID uuid.UUID) (LocatedStorage, error) {
	settings, err := l.settings.GetSettings(ctx, storageID)
	if err != nil {
		return LocatedStorage{}, fmt.Errorf("read storage settings: %w", err)
	}

	provider, err := l.registry.GetProvider(settings.GetType())
	if err != nil {
		return LocatedStorage{}, err
	}

	return LocatedStorage{
		Provider: provider,
		Location: providers.Location{StorageID: storageID, Settings: settings},
	}, nil
}

func (s LocatedStorage) getFileAccess(fileName string, logger *slog.Logger) providers.FileAccess {
	return providers.FileAccess{
		Location: s.Location,
		FileName: fileName,
		Logger: logger.With(
			slog.String("storage_id", s.Location.StorageID.String()),
			slog.String("file_name", fileName),
		),
	}
}
