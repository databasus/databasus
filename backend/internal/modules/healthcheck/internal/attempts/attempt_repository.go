package attempts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type (
	Repositories      struct{ attempts *attemptRepository }
	attemptRepository struct{ db *gorm.DB }
)

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{attempts: &attemptRepository{db: db}}
}

func (r *attemptRepository) insert(ctx context.Context, attempt Attempt) error {
	if err := database.Conn(ctx, r.db).Create(&attempt).Error; err != nil {
		return fmt.Errorf("insert healthcheck attempt: %w", err)
	}

	return nil
}

func (r *attemptRepository) list(ctx context.Context, input ListInput, cursor *database.TimeCursor) ([]Attempt, error) {
	query := database.Conn(ctx, r.db).Where("database_id = ? AND created_at >= ?", input.DatabaseID, input.Since)
	if cursor != nil {
		query = query.Where("(created_at,id) < (?,?)", cursor.Time, cursor.ID)
	}

	attempts := []Attempt{}
	err := query.Order("created_at DESC, id DESC").Limit(input.Limit).Find(&attempts).Error
	if err != nil {
		return nil, fmt.Errorf("list healthcheck attempts: %w", err)
	}

	return attempts, nil
}

func (r *attemptRepository) hasCursor(ctx context.Context, input ListInput, cursor database.TimeCursor) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).
		Model(&Attempt{}).
		Where("database_id = ? AND id = ? AND created_at = ? AND created_at >= ?", input.DatabaseID, cursor.ID, cursor.Time, input.Since).
		Count(&count).
		Error
	return count > 0, err
}

func (r *attemptRepository) deleteBefore(ctx context.Context, id uuid.UUID, before time.Time) error {
	if err := database.Conn(ctx, r.db).
		Where("database_id = ? AND created_at < ?", id, before).
		Delete(&Attempt{}).
		Error; err != nil {
		return fmt.Errorf("delete expired healthcheck attempts: %w", err)
	}

	return nil
}

func (r *attemptRepository) delete(ctx context.Context, id uuid.UUID) error {
	if err := database.Conn(ctx, r.db).Where("database_id = ?", id).Delete(&Attempt{}).Error; err != nil {
		return fmt.Errorf("delete healthcheck attempts: %w", err)
	}

	return nil
}
