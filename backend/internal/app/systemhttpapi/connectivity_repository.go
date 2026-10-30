package systemhttpapi

import (
	"context"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type connectivityRepository struct{ db *gorm.DB }

func (r *connectivityRepository) check(ctx context.Context) error {
	return database.Conn(ctx, r.db).Exec("SELECT 1").Error
}
