package snapshot

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// producer never holds it.
type Repositories struct {
	snapshots *snapshotRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{snapshots: &snapshotRepository{db: db}}
}

type snapshotRepository struct {
	db *gorm.DB
}

func (r *snapshotRepository) withExportedSnapshot(
	ctx context.Context,
	use func(snapshot database.ExportedSnapshot) error,
) error {
	return database.WithExportedSnapshot(ctx, r.db, use)
}

func (r *snapshotRepository) getDatabaseSizeBytes(ctx context.Context) (int64, error) {
	var databaseSizes []int64

	err := database.Conn(ctx, r.db).Raw("SELECT pg_database_size(current_database())").Scan(&databaseSizes).Error
	if err != nil {
		return 0, fmt.Errorf("read the internal database size: %w", err)
	}

	if len(databaseSizes) == 0 {
		return 0, nil
	}

	return databaseSizes[0], nil
}
