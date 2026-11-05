package userdatabase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	databases           *databaseRepository
	notifierAttachments *notifierAttachmentRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		databases:           &databaseRepository{db: db},
		notifierAttachments: &notifierAttachmentRepository{db: db},
	}
}

// databaseRepository keeps the parent rows in databases. NotifierIDs stay empty here; the
// attachment repository fills them.
type databaseRepository struct {
	db *gorm.DB
}

// rowLock is the lock a read takes on the database's row until the caller's transaction ends.
type rowLock string

const (
	noRowLock        rowLock = ""
	forUpdateRowLock rowLock = " FOR UPDATE"
)

const databaseColumns = "id, workspace_id, name, type, last_backup_time, last_backup_error_message, health_status"

type databaseRow struct {
	ID                     uuid.UUID
	WorkspaceID            uuid.UUID
	Name                   string
	Type                   Type
	LastBackupTime         *time.Time
	LastBackupErrorMessage *string
	HealthStatus           *HealthStatus
}

func (r databaseRow) toDatabase() Database {
	return Database{
		ID:                     r.ID,
		WorkspaceID:            r.WorkspaceID,
		Name:                   r.Name,
		Type:                   r.Type,
		NotifierIDs:            []uuid.UUID{},
		LastBackupTime:         r.LastBackupTime,
		LastBackupErrorMessage: r.LastBackupErrorMessage,
		HealthStatus:           r.HealthStatus,
	}
}

type databaseListFilter struct {
	workspaceID uuid.UUID
	// notifierID is nil for every database of the workspace.
	notifierID *uuid.UUID
	limit      int
	after      *database.NameCursor
}

// insert writes the parent row; the database gets its ID from the database, so a request can never
// name an existing row. The health status starts as the column's default.
func (r *databaseRepository) insert(
	ctx context.Context,
	workspaceID uuid.UUID,
	name string,
	databaseType Type,
) (Database, error) {
	var rows []databaseRow

	err := database.Conn(ctx, r.db).Raw(
		"INSERT INTO databases (workspace_id, name, type) VALUES (?, ?, ?) RETURNING "+databaseColumns,
		workspaceID, name, databaseType,
	).Scan(&rows).Error
	if err != nil {
		return Database{}, fmt.Errorf("insert the database: %w", err)
	}

	return rows[0].toDatabase(), nil
}

// insertCopy writes a copy with the original's health status and no backup history.
func (r *databaseRepository) insertCopy(ctx context.Context, original Database, name string) (Database, error) {
	var rows []databaseRow

	err := database.Conn(ctx, r.db).Raw(
		"INSERT INTO databases (workspace_id, name, type, health_status) VALUES (?, ?, ?, ?) RETURNING "+databaseColumns,
		original.WorkspaceID, name, original.Type, original.HealthStatus,
	).
		Scan(&rows).Error
	if err != nil {
		return Database{}, fmt.Errorf("insert the copy of the database: %w", err)
	}

	return rows[0].toDatabase(), nil
}

func (r *databaseRepository) get(ctx context.Context, databaseID uuid.UUID, lock rowLock) (Database, bool, error) {
	var rows []databaseRow

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+databaseColumns+" FROM databases WHERE id = ?"+string(lock), databaseID).
		Scan(&rows).Error
	if err != nil {
		return Database{}, false, fmt.Errorf("get the database: %w", err)
	}

	if len(rows) == 0 {
		return Database{}, false, nil
	}

	return rows[0].toDatabase(), true, nil
}

func (r *databaseRepository) listByIDs(ctx context.Context, databaseIDs []uuid.UUID) ([]Database, error) {
	return r.listWhere(ctx, "id IN ?", databaseIDs)
}

func (r *databaseRepository) listAll(ctx context.Context) ([]Database, error) {
	return r.listWhere(ctx, "TRUE")
}

func (r *databaseRepository) listWhere(ctx context.Context, condition string, values ...any) ([]Database, error) {
	var rows []databaseRow

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+databaseColumns+" FROM databases WHERE "+condition+" ORDER BY name, id", values...).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the databases: %w", err)
	}

	return toDatabases(rows), nil
}

func (r *databaseRepository) list(ctx context.Context, filter databaseListFilter) ([]Database, error) {
	query := "SELECT " + databaseColumns + " FROM databases WHERE workspace_id = @workspaceID"
	values := map[string]any{"workspaceID": filter.workspaceID, "limit": filter.limit}

	if filter.notifierID != nil {
		query += " AND id IN (SELECT database_id FROM databases_notifiers WHERE notifier_id = @notifierID)"
		values["notifierID"] = *filter.notifierID
	}

	if filter.after != nil {
		query += " AND (name, id) > (@afterName, @afterID)"
		values["afterName"] = filter.after.Name
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY name, id LIMIT @limit"

	var rows []databaseRow

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the databases: %w", err)
	}

	return toDatabases(rows), nil
}

func (r *databaseRepository) updateName(ctx context.Context, databaseID uuid.UUID, name string) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE databases SET name = ? WHERE id = ?", name, databaseID).Error
	if err != nil {
		return fmt.Errorf("update the name of the database: %w", err)
	}

	return nil
}

func (r *databaseRepository) updateWorkspace(ctx context.Context, databaseID, workspaceID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE databases SET workspace_id = ? WHERE id = ?", workspaceID, databaseID).Error
	if err != nil {
		return fmt.Errorf("update the workspace of the database: %w", err)
	}

	return nil
}

// recordBackupSuccess never moves the time backwards and clears the last backup error.
func (r *databaseRepository) recordBackupSuccess(
	ctx context.Context,
	databaseID uuid.UUID,
	backupTime time.Time,
) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		`UPDATE databases
		SET last_backup_time = GREATEST(COALESCE(last_backup_time, @backupTime), @backupTime),
			last_backup_error_message = NULL
		WHERE id = @databaseID`,
		map[string]any{"backupTime": backupTime.UTC(), "databaseID": databaseID},
	)
	if result.Error != nil {
		return false, fmt.Errorf("record the backup success of the database: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *databaseRepository) updateLastBackupErrorMessage(
	ctx context.Context,
	databaseID uuid.UUID,
	message string,
) (bool, error) {
	result := database.Conn(ctx, r.db).
		Exec("UPDATE databases SET last_backup_error_message = ? WHERE id = ?", message, databaseID)
	if result.Error != nil {
		return false, fmt.Errorf("update the last backup error of the database: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *databaseRepository) updateHealthStatus(
	ctx context.Context,
	databaseID uuid.UUID,
	status *HealthStatus,
) (bool, error) {
	result := database.Conn(ctx, r.db).
		Exec("UPDATE databases SET health_status = ? WHERE id = ?", status, databaseID)
	if result.Error != nil {
		return false, fmt.Errorf("update the health status of the database: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// countUsingNotifierOutsideWorkspace counts the databases outside the workspace that use the notifier.
func (r *databaseRepository) countUsingNotifierOutsideWorkspace(
	ctx context.Context,
	notifierID, workspaceID uuid.UUID,
) (int64, error) {
	var count int64

	err := database.Conn(ctx, r.db).Raw(
		`SELECT COUNT(*) FROM databases
		WHERE workspace_id <> ? AND id IN (SELECT database_id FROM databases_notifiers WHERE notifier_id = ?)`,
		workspaceID, notifierID,
	).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count the databases outside the workspace that use the notifier: %w", err)
	}

	return count, nil
}

func (r *databaseRepository) existsInWorkspace(ctx context.Context, workspaceID uuid.UUID) (bool, error) {
	var isExisting bool

	err := database.Conn(ctx, r.db).
		Raw("SELECT EXISTS (SELECT 1 FROM databases WHERE workspace_id = ?)", workspaceID).
		Scan(&isExisting).Error
	if err != nil {
		return false, fmt.Errorf("check whether the workspace has databases: %w", err)
	}

	return isExisting, nil
}

// The module's own cascade removes the database's settings row and attachments.
func (r *databaseRepository) delete(ctx context.Context, databaseID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM databases WHERE id = ?", databaseID).Error
	if err != nil {
		return fmt.Errorf("delete the database: %w", err)
	}

	return nil
}

func toDatabases(rows []databaseRow) []Database {
	databases := make([]Database, 0, len(rows))
	for _, row := range rows {
		databases = append(databases, row.toDatabase())
	}

	return databases
}

func (r *databaseRepository) lockDatabaseForBackup(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var ids []uuid.UUID
	if err := database.Conn(ctx, r.db).
		Raw("SELECT id FROM databases WHERE id=? FOR UPDATE", databaseID).
		Scan(&ids).
		Error; err != nil {
		return false, fmt.Errorf("lock database for backup: %w", err)
	}
	return len(ids) > 0, nil
}

func (r *databaseRepository) resetBackupHistory(ctx context.Context, databaseID uuid.UUID) error {
	for _, column := range []string{"last_backup_time", "last_backup_error_message"} {
		if err := database.Conn(ctx, r.db).
			Exec("UPDATE databases SET "+column+"=NULL WHERE id=?", databaseID).
			Error; err != nil {
			return fmt.Errorf("reset backup history indicators: %w", err)
		}
	}
	return nil
}
