package storageswitch

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type (
	Repositories            struct{ switches *storageSwitchRepository }
	storageSwitchRepository struct{ db *gorm.DB }
)

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{switches: &storageSwitchRepository{db: db}}
}

func (r *storageSwitchRepository) insert(ctx context.Context, operation StorageSwitch) error {
	if err := database.Conn(ctx, r.db).Create(&operation).Error; err != nil {
		return fmt.Errorf("record storage switch: %w", err)
	}
	return nil
}

func (r *storageSwitchRepository) get(ctx context.Context, id uuid.UUID) (StorageSwitch, bool, error) {
	var operation StorageSwitch
	err := database.Conn(ctx, r.db).Where("id = ?", id).Take(&operation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return StorageSwitch{}, false, nil
	}
	return operation, err == nil, err
}

func (r *storageSwitchRepository) list(ctx context.Context, query ListQuery, limit int) ([]StorageSwitch, error) {
	rows := database.Conn(ctx, r.db).Where("workspace_id = ?", query.WorkspaceID)
	if query.DatabaseID != nil {
		rows = rows.Where("database_id = ?", *query.DatabaseID)
	}
	if query.After != nil {
		cursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return nil, err
		}
		rows = rows.Where("(created_at,id) < (?,?)", cursor.Time, cursor.ID)
	}
	var operations []StorageSwitch
	err := rows.Order("created_at DESC, id DESC").Limit(limit).Find(&operations).Error
	return operations, err
}
