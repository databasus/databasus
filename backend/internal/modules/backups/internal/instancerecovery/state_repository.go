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
	for _, statement := range []struct {
		sql    string
		values []any
	}{
		{
			sql: `UPDATE backups_logical SET status = 'FAILED', fail_message = ?, backup_size_mb = 0
				WHERE status = 'IN_PROGRESS'`,
			values: []any{failMessage},
		},
		{
			sql:    `UPDATE backups_restores SET status = 'FAILED', fail_message = ? WHERE status = 'IN_PROGRESS'`,
			values: []any{failMessage},
		},
		{
			sql: `UPDATE backups_physical
				SET status = 'ERROR', error_reason = 'APPLICATION_RESTART', fail_message = ?, completed_at = ?
				WHERE status = 'IN_PROGRESS'`,
			values: []any{failMessage, now},
		},
		{
			sql: `UPDATE backups_physical_wal_streamers
				SET status = 'FAILED', failing_since = COALESCE(failing_since, ?)
				WHERE status = 'RUNNING'`,
			values: []any{now},
		},
		{sql: `DELETE FROM backups_physical_wal_segments WHERE file_name IS NULL`, values: nil},
		{sql: `DELETE FROM backups_download_tokens`, values: nil},
		{
			sql: `UPDATE backups_configs_physical
				SET force_full_requested_at = ?, force_incremental_requested_at = NULL`,
			values: []any{now},
		},
	} {
		if err := database.Conn(ctx, r.db).Exec(statement.sql, statement.values...).Error; err != nil {
			return fmt.Errorf("end the recovered backup work: %w", err)
		}
	}

	return nil
}
