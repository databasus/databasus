package app

import (
	"fmt"

	"gorm.io/gorm"

	restores_core "databasus-backend/internal/features/restores/core"
	verification_runs "databasus-backend/internal/features/verification/runs"
)

// GORM writes into the cached schema of a related model while it parses a relation for the first
// time, so two goroutines that preload related models at startup race on it. Parsing every model
// the old code preloads before any request or loop runs leaves only cached reads for them.
func parseModelSchemas(db *gorm.DB) error {
	models := []any{
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
