package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type Repositories struct{ agents *agentRepository }

func NewRepositories(db *gorm.DB) Repositories { return Repositories{agents: &agentRepository{db: db}} }

type agentRepository struct{ db *gorm.DB }

func (r *agentRepository) insert(ctx context.Context, agent Agent) error {
	return r.wrap(database.Conn(ctx, r.db).Create(&agent).Error)
}

func (r *agentRepository) get(ctx context.Context, id uuid.UUID) (Agent, bool, error) {
	var agents []Agent
	err := database.Conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&agents).Error
	if err != nil || len(agents) == 0 {
		return Agent{}, false, r.wrap(err)
	}
	return agents[0], true, nil
}

func (r *agentRepository) list(ctx context.Context) ([]Agent, error) {
	agents := []Agent{}
	err := database.Conn(ctx, r.db).Where("deleted_at IS NULL").Order("created_at DESC, id DESC").Find(&agents).Error
	return agents, r.wrap(err)
}

func (r *agentRepository) page(ctx context.Context, limit int, after *database.TimeCursor) ([]Agent, error) {
	agents := []Agent{}
	query := database.Conn(ctx, r.db).Where("deleted_at IS NULL")
	if after != nil {
		query = query.Where("(created_at, id) < (?, ?)", after.Time, after.ID)
	}
	err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&agents).Error
	return agents, r.wrap(err)
}

func (r *agentRepository) count(ctx context.Context) (int64, error) {
	var count int64
	err := database.Conn(ctx, r.db).Model(&Agent{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, r.wrap(err)
}

func (r *agentRepository) update(ctx context.Context, id uuid.UUID, fields map[string]any) (bool, error) {
	updated := database.Conn(ctx, r.db).Model(&Agent{}).Where("id = ? AND deleted_at IS NULL", id).Updates(fields)
	return updated.RowsAffected > 0, r.wrap(updated.Error)
}

func (r *agentRepository) stale(ctx context.Context, before time.Time) ([]Agent, error) {
	agents := []Agent{}
	err := database.Conn(ctx, r.db).
		Where("deleted_at IS NULL AND COALESCE(last_seen_at, created_at) < ?", before).
		Find(&agents).
		Error
	return agents, r.wrap(err)
}

func (*agentRepository) wrap(err error) error {
	if err != nil {
		return fmt.Errorf("query verification agents: %w", err)
	}
	return nil
}
