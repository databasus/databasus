package restore

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// inProgressIndexName is the partial unique index that allows one running restore per database.
const inProgressIndexName = "uq_backups_restores_in_progress_database"

const restoreColumns = "id, backup_id, database_id, status, fail_message, restore_duration_ms, created_at"

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	restores *restoreRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{restores: &restoreRepository{db: db}}
}

// IsRestoreRunningForBackups lets the logical backups leave alone a backup a restore reads.
func (r Repositories) IsRestoreRunningForBackups(ctx context.Context, backupIDs []uuid.UUID) (bool, error) {
	return r.restores.isRunningForBackups(ctx, backupIDs)
}

// IsRestoreRunningForDatabase lets a database deletion refuse while a restore reads its backups.
func (r Repositories) IsRestoreRunningForDatabase(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	return r.restores.isRunningForDatabase(ctx, databaseID)
}

type restoreRepository struct {
	db *gorm.DB
}

type restoreListFilter struct {
	backupID uuid.UUID
	after    *database.TimeCursor
	limit    int
}

// insert returns the unique violation of inProgressIndexName while a restore of the database runs,
// and a foreign key violation for a deleted backup.
func (r *restoreRepository) insert(ctx context.Context, restore Restore) error {
	err := database.Conn(ctx, r.db).Exec(`INSERT INTO backups_restores
		(id, backup_id, database_id, status, restore_duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		restore.ID, restore.BackupID, restore.DatabaseID, restore.Status, restore.RestoreDurationMs, restore.CreatedAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the restore: %w", err)
	}

	return nil
}

func (r *restoreRepository) get(ctx context.Context, restoreID uuid.UUID) (Restore, bool, error) {
	var foundRestores []Restore

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+restoreColumns+" FROM backups_restores WHERE id = ?", restoreID).
		Scan(&foundRestores).Error
	if err != nil {
		return Restore{}, false, fmt.Errorf("get the restore: %w", err)
	}

	if len(foundRestores) == 0 {
		return Restore{}, false, nil
	}

	return foundRestores[0], true, nil
}

// list pages through the restores of a backup newest first by (created_at, id).
func (r *restoreRepository) list(ctx context.Context, filter restoreListFilter) ([]Restore, error) {
	query := "SELECT " + restoreColumns + " FROM backups_restores WHERE backup_id = @backupID"
	values := map[string]any{"backupID": filter.backupID, "limit": filter.limit}

	if filter.after != nil {
		query += " AND (created_at, id) < (@afterTime, @afterID)"
		values["afterTime"] = filter.after.Time
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT @limit"

	var listedRestores []Restore

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&listedRestores).Error
	if err != nil {
		return nil, fmt.Errorf("list the restores: %w", err)
	}

	return listedRestores, nil
}

func (r *restoreRepository) isRunningForBackups(ctx context.Context, backupIDs []uuid.UUID) (bool, error) {
	var isRunning bool

	err := database.Conn(ctx, r.db).Raw(
		"SELECT EXISTS (SELECT 1 FROM backups_restores WHERE backup_id IN ? AND status = 'IN_PROGRESS')",
		backupIDs,
	).Scan(&isRunning).Error
	if err != nil {
		return false, fmt.Errorf("check for a running restore of the backups: %w", err)
	}

	return isRunning, nil
}

func (r *restoreRepository) isRunningForDatabase(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var isRunning bool

	err := database.Conn(ctx, r.db).Raw(
		"SELECT EXISTS (SELECT 1 FROM backups_restores WHERE database_id = ? AND status = 'IN_PROGRESS')",
		databaseID,
	).Scan(&isRunning).Error
	if err != nil {
		return false, fmt.Errorf("check for a running restore of the database: %w", err)
	}

	return isRunning, nil
}

func (r *restoreRepository) complete(ctx context.Context, restoreID uuid.UUID, durationMs int64) (bool, error) {
	return r.updateInProgress(ctx, "complete the restore",
		"status = 'COMPLETED', restore_duration_ms = ?", restoreID, durationMs)
}

func (r *restoreRepository) fail(
	ctx context.Context,
	restoreID uuid.UUID,
	failMessage string,
	durationMs int64,
) (bool, error) {
	return r.updateInProgress(ctx, "fail the restore",
		"status = ?, fail_message = ?, restore_duration_ms = ?", restoreID, StatusFailed, failMessage, durationMs)
}

func (r *restoreRepository) cancel(ctx context.Context, restoreID uuid.UUID, durationMs int64) (bool, error) {
	return r.updateInProgress(ctx, "cancel the restore",
		"status = ?, restore_duration_ms = ?", restoreID, StatusCanceled, durationMs)
}

func (r *restoreRepository) failAllInProgress(ctx context.Context, failMessage string) (int64, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE backups_restores SET status = ?, fail_message = ? WHERE status = ?",
		StatusFailed,
		failMessage,
		StatusInProgress,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("fail the restores left in progress: %w", result.Error)
	}

	return result.RowsAffected, nil
}

func (r *restoreRepository) updateInProgress(
	ctx context.Context,
	operation, assignments string,
	restoreID uuid.UUID,
	values ...any,
) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE backups_restores SET "+assignments+" WHERE id = ? AND status = ?",
		append(values, restoreID, StatusInProgress)...,
	)
	if result.Error != nil {
		return false, fmt.Errorf("%s: %w", operation, result.Error)
	}

	return result.RowsAffected > 0, nil
}
