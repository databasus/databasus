package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages/internal/providers"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	storages *storageRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{storages: &storageRepository{db: db}}
}

type storageRepository struct {
	db *gorm.DB
}

type storageListFilter struct {
	workspaceID uuid.UUID
	limit       int
	after       *database.NameCursor
}

// rowLock is the lock a read takes on the storage's row until the caller's transaction ends.
type rowLock string

const (
	noRowLock rowLock = ""
	// forShareRowLock keeps a transfer, which takes forUpdateRowLock, waiting.
	forShareRowLock  rowLock = " FOR SHARE"
	forUpdateRowLock rowLock = " FOR UPDATE"
)

const storageColumns = "id, workspace_id, name, type"

// providerSettings pairs the settings of a storage with the repository of their provider.
type providerSettings struct {
	repository providers.SettingsRepository
	settings   providers.Settings
}

// insertWithSettings expects the caller's transaction in ctx, so the storage never exists without
// its provider row.
func (r *storageRepository) insertWithSettings(
	ctx context.Context,
	createdStorage Storage,
	settings providerSettings,
) error {
	err := database.Conn(ctx, r.db).Exec(
		"INSERT INTO storages (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		createdStorage.ID, createdStorage.WorkspaceID, createdStorage.Name, createdStorage.Type,
	).Error
	if err != nil {
		return fmt.Errorf("insert the storage: %w", err)
	}

	return settings.repository.Upsert(ctx, createdStorage.ID, settings.settings)
}

func (r *storageRepository) get(ctx context.Context, storageID uuid.UUID, lock rowLock) (Storage, bool, error) {
	var storages []Storage

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+storageColumns+" FROM storages WHERE id = ?"+string(lock), storageID).
		Scan(&storages).Error
	if err != nil {
		return Storage{}, false, fmt.Errorf("get the storage: %w", err)
	}

	if len(storages) == 0 {
		return Storage{}, false, nil
	}

	return storages[0], true, nil
}

func (r *storageRepository) listByIDs(ctx context.Context, storageIDs []uuid.UUID) ([]Storage, error) {
	var storages []Storage

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+storageColumns+" FROM storages WHERE id IN ? ORDER BY name, id", storageIDs).
		Scan(&storages).Error
	if err != nil {
		return nil, fmt.Errorf("list the storages by id: %w", err)
	}

	return storages, nil
}

func (r *storageRepository) listByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Storage, error) {
	var storages []Storage

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+storageColumns+" FROM storages WHERE workspace_id = ? ORDER BY name, id", workspaceID).
		Scan(&storages).Error
	if err != nil {
		return nil, fmt.Errorf("list the storages of the workspace: %w", err)
	}

	return storages, nil
}

func (r *storageRepository) list(ctx context.Context, filter storageListFilter) ([]Storage, error) {
	query := "SELECT " + storageColumns + " FROM storages WHERE workspace_id = @workspaceID"
	values := map[string]any{"workspaceID": filter.workspaceID, "limit": filter.limit}

	if filter.after != nil {
		query += " AND (name, id) > (@afterName, @afterID)"
		values["afterName"] = filter.after.Name
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY name, id LIMIT @limit"

	var storages []Storage

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&storages).Error
	if err != nil {
		return nil, fmt.Errorf("list the storages: %w", err)
	}

	return storages, nil
}

func (r *storageRepository) listTypes(ctx context.Context) ([]Type, error) {
	var storageTypes []Type

	err := database.Conn(ctx, r.db).Raw("SELECT type FROM storages").Scan(&storageTypes).Error
	if err != nil {
		return nil, fmt.Errorf("list the storage types: %w", err)
	}

	return storageTypes, nil
}

// updateNameAndSettings expects the caller's transaction in ctx, like insertWithSettings.
func (r *storageRepository) updateNameAndSettings(
	ctx context.Context,
	storageID uuid.UUID,
	name string,
	settings providerSettings,
) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE storages SET name = ? WHERE id = ?", name, storageID).Error
	if err != nil {
		return fmt.Errorf("update the name of the storage: %w", err)
	}

	return settings.repository.Upsert(ctx, storageID, settings.settings)
}

func (r *storageRepository) updateWorkspace(ctx context.Context, storageID, workspaceID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE storages SET workspace_id = ? WHERE id = ?", workspaceID, storageID).Error
	if err != nil {
		return fmt.Errorf("update the workspace of the storage: %w", err)
	}

	return nil
}

// delete removes the storage and, through the module's own cascade, its provider row and its pending
// deletions. A backup config or a backup that still references it refuses the deletion with a foreign
// key violation.
func (r *storageRepository) delete(ctx context.Context, storageID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM storages WHERE id = ?", storageID).Error
	if err != nil {
		return fmt.Errorf("delete the storage: %w", err)
	}

	return nil
}
