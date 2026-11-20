package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/postgresql/internaldb"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/users"
)

// StateNormalizer ends, in a database a recovery staged, what one module had in progress on the
// lost instance, and returns how many unfinished file writes and deletions it discarded.
type StateNormalizer func(ctx context.Context, db *gorm.DB, now time.Time) (int, error)

// databaseRepository reads and changes a database the application does not run on: the target a
// startup recovery would replace, or the staging database it imports into. It holds its own pool,
// which close releases before publication renames the database.
type databaseRepository struct {
	db *gorm.DB
}

func openDatabaseRepository(ctx context.Context, connection internaldb.Connection) (*databaseRepository, error) {
	db, err := database.Open(ctx, connection.ConnString())
	if err != nil {
		return nil, err
	}

	return &databaseRepository{db: db}, nil
}

func (r *databaseRepository) close() {
	closeDatabase(r.db)
}

func (r *databaseRepository) readRecordedVersions(ctx context.Context) ([]int64, error) {
	return database.ReadRecordedMigrationVersions(ctx, r.db)
}

func (r *databaseRepository) readAppliedVersions(ctx context.Context) ([]int64, error) {
	return database.ReadAppliedMigrationVersions(ctx, r.db)
}

func (r *databaseRepository) listPopulatedTables(ctx context.Context) ([]string, error) {
	return database.ListPopulatedTables(ctx, r.db)
}

func (r *databaseRepository) inspectUsers(ctx context.Context) (users.RecoveryTargetState, error) {
	return users.InspectRecoveryTarget(ctx, r.db)
}

func (r *databaseRepository) inspectInstanceBackupSettings(
	ctx context.Context,
) (configs.RecoveryTargetSettings, error) {
	return r.getRecoveryState().InspectTarget(ctx)
}

func (r *databaseRepository) markRecovered(ctx context.Context, operationID uuid.UUID, recoveredAt time.Time) error {
	return r.getRecoveryState().MarkRecovered(ctx, operationID, recoveredAt)
}

func (r *databaseRepository) getRecoveredByOperationID(ctx context.Context) (*uuid.UUID, error) {
	return r.getRecoveryState().GetRecoveredByOperationID(ctx)
}

// normalize runs every module's normalization and then ends this module's own work, and returns how
// many unfinished file writes and deletions were discarded.
func (r *databaseRepository) normalize(
	ctx context.Context,
	normalizers []StateNormalizer,
	now time.Time,
) (int, error) {
	discardedCount := 0

	for _, normalizer := range normalizers {
		discarded, err := normalizer(ctx, r.db, now)
		if err != nil {
			return 0, err
		}

		discardedCount += discarded
	}

	return discardedCount, r.endInstanceBackupWork(ctx, now)
}

// endInstanceBackupWork interrupts the Databasus backups the lost instance had running, forgets
// its spool files and revokes its download tokens.
func (r *databaseRepository) endInstanceBackupWork(ctx context.Context, now time.Time) error {
	for _, statement := range []struct {
		sql    string
		values []any
	}{
		{
			sql: `UPDATE instancebackups_backups SET dump_status = 'INTERRUPTED' WHERE dump_status = 'RUNNING'`,
		},
		{
			sql: `UPDATE instancebackups_backups
				SET status = 'INTERRUPTED', completed_at = ?, failure_reason = COALESCE(failure_reason, 'INTERRUPTED')
				WHERE status = 'IN_PROGRESS'`,
			values: []any{now},
		},
		{sql: `UPDATE instancebackups_backups SET spool_path = NULL, spool_expires_at = NULL`},
		{
			sql: `UPDATE instancebackups_copies SET status = 'INTERRUPTED', next_attempt_at = NULL
				WHERE status IN ('PENDING', 'UPLOADING', 'RETRYING')`,
		},
		{sql: `DELETE FROM instancebackups_tokens`},
	} {
		if err := database.Conn(ctx, r.db).Exec(statement.sql, statement.values...).Error; err != nil {
			return fmt.Errorf("end the recovered Databasus backup work: %w", err)
		}
	}

	return nil
}

func (r *databaseRepository) getRecoveryState() *configs.RecoveryStateService {
	return configs.NewRecoveryStateService(configs.NewRepositories(r.db))
}

func closeDatabase(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
