package usage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type Repositories struct {
	usage *usageRepository
}

type usageRepository struct {
	db *gorm.DB
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{usage: &usageRepository{db: db}}
}

func (r *usageRepository) getDatabaseUsage(ctx context.Context, databaseID uuid.UUID) (float64, error) {
	return r.getTotal(ctx, "database_id", databaseID)
}

func (r *usageRepository) getStorageUsage(ctx context.Context, storageID uuid.UUID) (float64, error) {
	return r.getTotal(ctx, "storage_id", storageID)
}

func (r *usageRepository) getTotal(ctx context.Context, scopeColumn string, scopeID uuid.UUID) (float64, error) {
	query := fmt.Sprintf(`SELECT COALESCE(SUM(size_mb), 0) FROM (
		SELECT backup_size_mb AS size_mb FROM backups_logical
		WHERE %s = @scopeID AND status = 'COMPLETED'
		UNION ALL
		SELECT backup_size_mb AS size_mb FROM backups_physical
		WHERE %s = @scopeID AND status = 'COMPLETED'
		UNION ALL
		SELECT compressed_size_mb AS size_mb FROM backups_physical_wal_segments
		WHERE %s = @scopeID AND file_name IS NOT NULL
	) AS stored_backups`, scopeColumn, scopeColumn, scopeColumn)
	var totalUsageMb float64

	err := database.Conn(ctx, r.db).Raw(query, map[string]any{"scopeID": scopeID}).Scan(&totalUsageMb).Error
	if err != nil {
		return 0, fmt.Errorf("read recorded backup usage: %w", err)
	}

	return totalUsageMb, nil
}
