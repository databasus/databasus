// Package storage creates, lists, reads, changes, tests, transfers and deletes storages, and keeps
// their parent rows.
package storage

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

// Type is where a storage keeps its files. It is fixed when the storage is created.
type Type = providers.Type

// Storage is a place for backup files as other modules see it, without its settings.
type Storage struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Type        Type
}

// WithSettings is a storage with the settings of its provider. Secrets in them stay encrypted.
type WithSettings struct {
	Storage
	Settings providers.Settings
}

// Creation is a new storage; the type of its settings becomes the storage's type.
type Creation struct {
	WorkspaceID uuid.UUID
	Name        string
	Settings    providers.Settings
}

// Change replaces the name and the settings of a storage. A secret the settings leave out keeps its
// stored value while the resource address stays the stored one.
type Change struct {
	StorageID uuid.UUID
	Name      string
	Settings  providers.Settings
}
