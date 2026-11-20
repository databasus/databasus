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

func (r *stateRepository) discardPendingFileWork(ctx context.Context) (int, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM storages_pending_file_deletions")
	if result.Error != nil {
		return 0, fmt.Errorf("discard the recovered file writes and deletions: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}
