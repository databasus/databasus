package app

import (
	"fmt"

	"gorm.io/gorm"

	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	verification_runs "databasus-backend/internal/features/verification/runs"
)

// GORM writes into the cached schema of a related model while it parses a relation for the first
// time, so two goroutines that preload related models at startup race on it. Parsing every model
// the old code preloads before any request or loop runs leaves only cached reads for them.
func parseModelSchemas(db *gorm.DB) error {
	models := []any{
		&databases.Database{},
		&storages.Storage{},
		&notifiers.Notifier{},
		&backups_config_logical.LogicalBackupConfig{},
		&backups_config_physical.PhysicalBackupConfig{},
		&restores_core.Restore{},
		&verification_runs.RestoreVerification{},
	}

	for _, model := range models {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(model); err != nil {
			return fmt.Errorf("parse the schema of %T: %w", model, err)
		}
	}

	return nil
}
