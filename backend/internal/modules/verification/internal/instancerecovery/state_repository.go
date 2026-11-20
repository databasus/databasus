package instancerecovery

import (
	"context"
	"fmt"
	"time"

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

func (r *stateRepository) endRecoveredWork(ctx context.Context, now time.Time, failMessage string) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE verification_runs
		SET status = 'FAILED', fail_message = ?, finished_at = ?
		WHERE status = 'RUNNING'`, failMessage, now).Error
	if err != nil {
		return fmt.Errorf("fail the recovered verifications: %w", err)
	}

	err = database.Conn(ctx, r.db).Exec(`UPDATE verification_agents
		SET token_hash = encode(sha256(convert_to(gen_random_uuid()::text, 'UTF8')), 'hex')`).Error
	if err != nil {
		return fmt.Errorf("revoke the recovered agent tokens: %w", err)
	}

	return nil
}
