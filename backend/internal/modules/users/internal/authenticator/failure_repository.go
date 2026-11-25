package authenticator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type failureRepository struct {
	db *gorm.DB
}

func (r *failureRepository) countSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var failureCount int64

	err := database.Conn(ctx, r.db).Model(&SecondFactorFailure{}).
		Where("user_id = ? AND failed_at > ?", userID, since).
		Count(&failureCount).Error
	if err != nil {
		return 0, fmt.Errorf("count the wrong codes of the user: %w", err)
	}

	return failureCount, nil
}

func (r *failureRepository) insert(ctx context.Context, failure SecondFactorFailure) error {
	err := database.Conn(ctx, r.db).Create(&failure).Error
	if err != nil {
		return fmt.Errorf("record a wrong code of the user: %w", err)
	}

	return nil
}

func (r *failureRepository) deleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_second_factor_failures WHERE failed_at < ?", cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the old wrong codes: %w", result.Error)
	}

	return result.RowsAffected, nil
}
