package config

import "databasus-backend/internal/modules/backups/internal/storageswitch"

type SavedBackupConfig struct {
	Config        AnyBackupConfig
	StorageSwitch *storageswitch.Outcome
}
