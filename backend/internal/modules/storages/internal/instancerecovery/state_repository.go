package instancerecovery

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type Repositories struct {
	state *stateRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{state: &stateRepository{db: db}}
}

type stateRepository struct {
	db *gorm.DB
}

func (r *stateRepository) requireLocalStorageValidation(ctx context.Context) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE storages SET recovery_validation_status = 'REQUIRES_VALIDATION' WHERE type = 'LOCAL'",
	).Error
	if err != nil {
		return fmt.Errorf("mark the recovered local storages: %w", err)
	}

	return nil
}

func (r *stateRepository) discardPendingFileWork(ctx context.Context) (int, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM storages_pending_file_deletions")
	if result.Error != nil {
		return 0, fmt.Errorf("discard the recovered file writes and deletions: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}
