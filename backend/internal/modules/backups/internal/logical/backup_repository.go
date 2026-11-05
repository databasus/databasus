package logical

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
)

// inProgressIndexName is the partial unique index that allows one running backup per database.
const inProgressIndexName = "uq_backups_logical_in_progress_database"

const backupColumns = "id, database_id, storage_id, file_name, status, fail_message, backup_size_mb, " +
	"backup_raw_db_size_mb, backup_duration_ms, encryption, encryption_salt, encryption_iv, " +
	"timescaledb_version, restore_verification_status, created_at"

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	backups *backupRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{backups: &backupRepository{db: db}}
}

type backupRepository struct {
	db *gorm.DB
}

type backupListFilter struct {
	databaseID    uuid.UUID
	statuses      []Status
	createdBefore *time.Time
	after         *database.TimeCursor
	limit         int
}

// completion is what a published backup records besides its status.
type completion struct {
	backupID           uuid.UUID
	sizeMb             float64
	rawDatabaseSizeMb  float64
	durationMs         int64
	encryption         config.Encryption
	encryptionSalt     *string
	encryptionIV       *string
	timescaledbVersion string
}

// insert returns the unique violation of inProgressIndexName when a backup of the database runs.
func (r *backupRepository) insert(ctx context.Context, backup LogicalBackup) error {
	err := database.Conn(ctx, r.db).Exec(`INSERT INTO backups_logical
		(id, database_id, storage_id, file_name, status, encryption, restore_verification_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		backup.ID, backup.DatabaseID, backup.StorageID, backup.FileName, backup.Status, backup.Encryption,
		backup.RestoreVerificationStatus, backup.CreatedAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the logical backup: %w", err)
	}

	return nil
}

func (r *backupRepository) get(ctx context.Context, backupID uuid.UUID) (LogicalBackup, bool, error) {
	return r.getOne(ctx, "SELECT "+backupColumns+" FROM backups_logical WHERE id = ?", backupID)
}

// getForUpdate locks the backup's row until the caller's transaction ends. A restore whose insert
// references the row waits for the lock and then finds the backup gone.
func (r *backupRepository) getForUpdate(ctx context.Context, backupID uuid.UUID) (LogicalBackup, bool, error) {
	return r.getOne(ctx, "SELECT "+backupColumns+" FROM backups_logical WHERE id = ? FOR UPDATE", backupID)
}

func (r *backupRepository) getLatestCompleted(ctx context.Context, databaseID uuid.UUID) (LogicalBackup, bool, error) {
	return r.getOne(ctx, "SELECT "+backupColumns+` FROM backups_logical
		WHERE database_id = ? AND status = 'COMPLETED'
		ORDER BY created_at DESC, id DESC LIMIT 1`, databaseID)
}

func (r *backupRepository) listByIDs(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error) {
	var listedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+backupColumns+" FROM backups_logical WHERE id IN ?", backupIDs).
		Scan(&listedBackups).Error
	if err != nil {
		return nil, fmt.Errorf("list the logical backups by id: %w", err)
	}

	return listedBackups, nil
}

// list pages through the backups of a database newest first by (created_at, id).
func (r *backupRepository) list(ctx context.Context, filter backupListFilter) ([]LogicalBackup, error) {
	query := "SELECT " + backupColumns + " FROM backups_logical WHERE database_id = @databaseID"
	values := map[string]any{"databaseID": filter.databaseID, "limit": filter.limit}

	if len(filter.statuses) > 0 {
		query += " AND status IN @statuses"
		values["statuses"] = filter.statuses
	}

	if filter.createdBefore != nil {
		query += " AND created_at < @createdBefore"
		values["createdBefore"] = *filter.createdBefore
	}

	if filter.after != nil {
		query += " AND (created_at, id) < (@afterTime, @afterID)"
		values["afterTime"] = filter.after.Time
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT @limit"

	var listedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&listedBackups).Error
	if err != nil {
		return nil, fmt.Errorf("list the logical backups: %w", err)
	}

	return listedBackups, nil
}

// listNewestPerDatabase reads up to count newest backups of each database, newest first.
func (r *backupRepository) listNewestPerDatabase(
	ctx context.Context,
	databaseIDs []uuid.UUID,
	count int,
) ([]LogicalBackup, error) {
	var listedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+backupColumns+` FROM unnest(?::uuid[]) AS requested (requested_database_id)
		CROSS JOIN LATERAL (
			SELECT *
			FROM backups_logical
			WHERE database_id = requested.requested_database_id
			ORDER BY created_at DESC, id DESC
			LIMIT ?
		) AS newest
		ORDER BY database_id, created_at DESC, id DESC`, pq.Array(databaseIDs), count).
		Scan(&listedBackups).
		Error
	if err != nil {
		return nil, fmt.Errorf("list the newest logical backups per database: %w", err)
	}

	return listedBackups, nil
}

func (r *backupRepository) listByDatabase(ctx context.Context, databaseID uuid.UUID) ([]LogicalBackup, error) {
	var listedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+backupColumns+" FROM backups_logical WHERE database_id = ? ORDER BY created_at DESC, id DESC",
			databaseID).
		Scan(&listedBackups).Error
	if err != nil {
		return nil, fmt.Errorf("list the logical backups of the database: %w", err)
	}

	return listedBackups, nil
}

func (r *backupRepository) isInProgressForDatabase(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var isInProgress bool

	err := database.Conn(ctx, r.db).Raw(
		"SELECT EXISTS (SELECT 1 FROM backups_logical WHERE database_id = ? AND status = 'IN_PROGRESS')",
		databaseID,
	).Scan(&isInProgress).Error
	if err != nil {
		return false, fmt.Errorf("check for a running logical backup: %w", err)
	}

	return isInProgress, nil
}

// updateProgress leaves a backup that is no longer running as it is, so a late progress tick cannot
// overwrite its outcome.
func (r *backupRepository) updateProgress(
	ctx context.Context,
	backupID uuid.UUID,
	sizeMb float64,
	durationMs int64,
) (bool, error) {
	return r.updateInProgress(ctx, "update the progress of the logical backup",
		"backup_size_mb = ?, backup_duration_ms = ?", backupID, sizeMb, durationMs)
}

func (r *backupRepository) complete(ctx context.Context, backupCompletion completion) (bool, error) {
	return r.updateInProgress(ctx, "complete the logical backup",
		`status = 'COMPLETED', backup_size_mb = ?, backup_raw_db_size_mb = ?, backup_duration_ms = ?,
		encryption = ?, encryption_salt = ?, encryption_iv = ?, timescaledb_version = ?`,
		backupCompletion.backupID,
		backupCompletion.sizeMb, backupCompletion.rawDatabaseSizeMb, backupCompletion.durationMs,
		backupCompletion.encryption, backupCompletion.encryptionSalt, backupCompletion.encryptionIV,
		backupCompletion.timescaledbVersion)
}

func (r *backupRepository) fail(
	ctx context.Context,
	backupID uuid.UUID,
	failMessage string,
	durationMs int64,
) (bool, error) {
	return r.updateInProgress(ctx, "fail the logical backup",
		"status = 'FAILED', fail_message = ?, backup_size_mb = 0, backup_duration_ms = ?",
		backupID, failMessage, durationMs)
}

func (r *backupRepository) cancel(ctx context.Context, backupID uuid.UUID, durationMs int64) (bool, error) {
	return r.updateInProgress(ctx, "cancel the logical backup",
		"status = 'CANCELED', backup_size_mb = 0, backup_duration_ms = ?", backupID, durationMs)
}

// failAllInProgress fails every backup left running and returns the failed rows.
func (r *backupRepository) failAllInProgress(ctx context.Context, failMessage string) ([]LogicalBackup, error) {
	var failedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).Raw(`UPDATE backups_logical
		SET status = 'FAILED', fail_message = ?, backup_size_mb = 0
		WHERE status = 'IN_PROGRESS'
		RETURNING `+backupColumns, failMessage).
		Scan(&failedBackups).Error
	if err != nil {
		return nil, fmt.Errorf("fail the logical backups left in progress: %w", err)
	}

	return failedBackups, nil
}

func (r *backupRepository) updateVerificationStatus(
	ctx context.Context,
	backupID uuid.UUID,
	status VerificationStatus,
) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE backups_logical SET restore_verification_status = ? WHERE id = ?", status, backupID,
	)
	if result.Error != nil {
		return false, fmt.Errorf("update the verification status of the logical backup: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// deleteByIDs removes the backups, and through the module's cascade their restores and download
// tokens, and returns what the removed rows named.
func (r *backupRepository) deleteByIDs(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error) {
	var deletedBackups []LogicalBackup

	err := database.Conn(ctx, r.db).
		Raw("DELETE FROM backups_logical WHERE id IN ? RETURNING "+backupColumns, backupIDs).
		Scan(&deletedBackups).Error
	if err != nil {
		return nil, fmt.Errorf("delete the logical backups: %w", err)
	}

	return deletedBackups, nil
}

func (r *backupRepository) lockAndListBackupIDsOfDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
) ([]uuid.UUID, error) {
	var lockedIDs []uuid.UUID

	err := database.Conn(ctx, r.db).
		Raw("SELECT id FROM backups_logical WHERE database_id = ? FOR UPDATE", databaseID).
		Scan(&lockedIDs).Error
	if err != nil {
		return nil, fmt.Errorf("lock the logical backups of the database: %w", err)
	}

	return lockedIDs, nil
}

func (r *backupRepository) getOne(ctx context.Context, query string, values ...any) (LogicalBackup, bool, error) {
	var foundBackups []LogicalBackup

	err := database.Conn(ctx, r.db).Raw(query, values...).Scan(&foundBackups).Error
	if err != nil {
		return LogicalBackup{}, false, fmt.Errorf("get the logical backup: %w", err)
	}

	if len(foundBackups) == 0 {
		return LogicalBackup{}, false, nil
	}

	return foundBackups[0], true, nil
}

func (r *backupRepository) updateInProgress(
	ctx context.Context,
	operation, assignments string,
	backupID uuid.UUID,
	values ...any,
) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE backups_logical SET "+assignments+" WHERE id = ? AND status = 'IN_PROGRESS'",
		append(values, backupID)...,
	)
	if result.Error != nil {
		return false, fmt.Errorf("%s: %w", operation, result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *backupRepository) listLatestCompleted(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]LogicalBackup, error) {
	var backups []LogicalBackup
	err := database.Conn(ctx, r.db).
		Raw("SELECT DISTINCT ON (database_id) "+backupColumns+" FROM backups_logical WHERE database_id IN ? AND status = 'COMPLETED' ORDER BY database_id, created_at DESC, id DESC", ids).
		Scan(&backups).
		Error
	if err != nil {
		return nil, fmt.Errorf("list latest completed logical backups: %w", err)
	}

	byDatabaseID := make(map[uuid.UUID]LogicalBackup, len(backups))
	for _, backup := range backups {
		byDatabaseID[backup.DatabaseID] = backup
	}

	return byDatabaseID, nil
}
