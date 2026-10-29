package configs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/database"
)

type (
	Repositories     struct{ configs *configRepository }
	configRepository struct{ db *gorm.DB }
)

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{configs: &configRepository{db: db}}
}

func (r *configRepository) getConfig(ctx context.Context, id uuid.UUID) (Config, bool, error) {
	var configs []Config
	err := database.Conn(ctx, r.db).Where("database_id = ?", id).Limit(1).Find(&configs).Error
	if err != nil {
		return Config{}, false, fmt.Errorf("get healthcheck config: %w", err)
	}

	if len(configs) == 0 {
		return Config{}, false, nil
	}

	return configs[0], true, nil
}

func (r *configRepository) save(ctx context.Context, config Config) error {
	err := database.Conn(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "database_id"}}, DoUpdates: clause.AssignmentColumns([]string{"is_healthcheck_enabled", "is_notification_enabled", "interval_minutes", "failed_attempts_before_unavailable", "attempt_retention_days"})}).
		Create(&config).
		Error
	if err != nil {
		return fmt.Errorf("save healthcheck config: %w", err)
	}

	return nil
}

func (r *configRepository) listDue(ctx context.Context, now time.Time) ([]Config, error) {
	configs := []Config{}
	err := database.Conn(ctx, r.db).Raw(`SELECT configs.* FROM healthcheck_configs configs
 LEFT JOIN (SELECT database_id, MAX(created_at) AS last_attempt_at FROM healthcheck_attempts GROUP BY database_id) attempts USING (database_id)
 WHERE configs.is_healthcheck_enabled AND (attempts.last_attempt_at IS NULL OR attempts.last_attempt_at + configs.interval_minutes * INTERVAL '1 minute' <= ?)`, now.Add(time.Second)).Scan(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("list due healthcheck configs: %w", err)
	}

	return configs, nil
}

func (r *configRepository) delete(ctx context.Context, id uuid.UUID) error {
	err := database.Conn(ctx, r.db).Where("database_id = ?", id).Delete(&Config{}).Error
	if err != nil {
		return fmt.Errorf("delete healthcheck config: %w", err)
	}

	return nil
}
