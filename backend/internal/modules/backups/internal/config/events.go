package config

import (
	"context"
)

// PhysicalBackupConfigSaved is published after the transaction that saves a physical config commits.
type PhysicalBackupConfigSaved struct {
	// Previous is nil when no config was saved before.
	Previous *PhysicalBackupConfig
	Saved    PhysicalBackupConfig
}

// PhysicalBackupConfigSavedHandler runs after the save committed; its error becomes the save's error
// while the saved config stays.
type PhysicalBackupConfigSavedHandler func(ctx context.Context, event PhysicalBackupConfigSaved) error
