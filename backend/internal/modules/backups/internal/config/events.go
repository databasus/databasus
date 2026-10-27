package config

import (
	"context"

	"github.com/google/uuid"
)

// LogicalBackupConfigStorageChanging is published inside the transaction that saves a logical config
// with another storage, so the logical backups of the previous storage leave together with the save.
type LogicalBackupConfigStorageChanging struct {
	DatabaseID uuid.UUID
	// PreviousStorageID is nil when the config named no storage or was never saved.
	PreviousStorageID *uuid.UUID
	StorageID         uuid.UUID
}

// PhysicalBackupConfigSaved is published after the transaction that saves a physical config commits.
type PhysicalBackupConfigSaved struct {
	// Previous is nil when no config was saved before.
	Previous *PhysicalBackupConfig
	Saved    PhysicalBackupConfig
}

// LogicalBackupConfigStorageChangingHandler runs in the save's transaction; an error rolls the save
// back.
type LogicalBackupConfigStorageChangingHandler func(ctx context.Context, event LogicalBackupConfigStorageChanging) error

// PhysicalBackupConfigSavedHandler runs after the save committed; its error becomes the save's error
// while the saved config stays.
type PhysicalBackupConfigSavedHandler func(ctx context.Context, event PhysicalBackupConfigSaved) error
