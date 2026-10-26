package notifier

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	notifiers *notifierRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{notifiers: &notifierRepository{db: db}}
}

type notifierRepository struct {
	db *gorm.DB
}

type notifierListFilter struct {
	workspaceID uuid.UUID
	limit       int
	after       *database.NameCursor
}

const notifierColumns = "id, workspace_id, name, type, last_send_error"

// providerSettings pairs the settings of a notifier with the repository of their provider.
type providerSettings struct {
	repository providers.SettingsRepository
	settings   providers.Settings
}

// insertWithSettings expects the caller's transaction in ctx, so the notifier never exists
// without its provider row.
func (r *notifierRepository) insertWithSettings(
	ctx context.Context,
	createdNotifier Notifier,
	settings providerSettings,
) error {
	err := database.Conn(ctx, r.db).Exec(
		"INSERT INTO notifiers (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		createdNotifier.ID, createdNotifier.WorkspaceID, createdNotifier.Name, createdNotifier.Type,
	).Error
	if err != nil {
		return fmt.Errorf("insert the notifier: %w", err)
	}

	return settings.repository.Upsert(ctx, createdNotifier.ID, settings.settings)
}

func (r *notifierRepository) get(ctx context.Context, notifierID uuid.UUID) (Notifier, bool, error) {
	var notifiers []Notifier

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+notifierColumns+" FROM notifiers WHERE id = ?", notifierID).
		Scan(&notifiers).Error
	if err != nil {
		return Notifier{}, false, fmt.Errorf("get the notifier: %w", err)
	}

	if len(notifiers) == 0 {
		return Notifier{}, false, nil
	}

	return notifiers[0], true, nil
}

// lockForUpdate holds the row until the caller's transaction ends. FOR UPDATE conflicts with the
// FOR KEY SHARE that the reference check of a databases_notifiers insert takes, so an attach
// still in flight finishes first or waits for the caller.
func (r *notifierRepository) lockForUpdate(ctx context.Context, notifierID uuid.UUID) error {
	var lockedIDs []uuid.UUID

	err := database.Conn(ctx, r.db).
		Raw("SELECT id FROM notifiers WHERE id = ? FOR UPDATE", notifierID).
		Scan(&lockedIDs).Error
	if err != nil {
		return fmt.Errorf("lock the notifier: %w", err)
	}

	return nil
}

func (r *notifierRepository) listByIDs(ctx context.Context, notifierIDs []uuid.UUID) ([]Notifier, error) {
	var notifiers []Notifier

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+notifierColumns+" FROM notifiers WHERE id IN ? ORDER BY name, id", notifierIDs).
		Scan(&notifiers).Error
	if err != nil {
		return nil, fmt.Errorf("list the notifiers by id: %w", err)
	}

	return notifiers, nil
}

func (r *notifierRepository) list(ctx context.Context, filter notifierListFilter) ([]Notifier, error) {
	query := "SELECT " + notifierColumns + " FROM notifiers WHERE workspace_id = @workspaceID"
	values := map[string]any{"workspaceID": filter.workspaceID, "limit": filter.limit}

	if filter.after != nil {
		query += " AND (name, id) > (@afterName, @afterID)"
		values["afterName"] = filter.after.Name
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY name, id LIMIT @limit"

	var notifiers []Notifier

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&notifiers).Error
	if err != nil {
		return nil, fmt.Errorf("list the notifiers: %w", err)
	}

	return notifiers, nil
}

func (r *notifierRepository) listTypes(ctx context.Context) ([]notification.NotifierType, error) {
	var notifierTypes []notification.NotifierType

	err := database.Conn(ctx, r.db).Raw("SELECT type FROM notifiers").Scan(&notifierTypes).Error
	if err != nil {
		return nil, fmt.Errorf("list the notifier types: %w", err)
	}

	return notifierTypes, nil
}

// updateNameAndSettings expects the caller's transaction in ctx, like insertWithSettings.
func (r *notifierRepository) updateNameAndSettings(
	ctx context.Context,
	notifierID uuid.UUID,
	name string,
	settings providerSettings,
) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE notifiers SET name = ? WHERE id = ?", name, notifierID).Error
	if err != nil {
		return fmt.Errorf("update the name of the notifier: %w", err)
	}

	return settings.repository.Upsert(ctx, notifierID, settings.settings)
}

func (r *notifierRepository) updateWorkspace(ctx context.Context, notifierID, workspaceID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE notifiers SET workspace_id = ? WHERE id = ?", workspaceID, notifierID).Error
	if err != nil {
		return fmt.Errorf("update the workspace of the notifier: %w", err)
	}

	return nil
}

// updateLastSendError writes only the outcome of a send, so it never overwrites a change of the
// name or the settings saved meanwhile.
func (r *notifierRepository) updateLastSendError(
	ctx context.Context,
	notifierID uuid.UUID,
	lastSendError *string,
) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE notifiers SET last_send_error = ? WHERE id = ?", lastSendError, notifierID).Error
	if err != nil {
		return fmt.Errorf("update the last send error of the notifier: %w", err)
	}

	return nil
}

// delete removes the notifier and, through the module's own cascade, its provider row. A database
// that still uses it refuses the deletion with a foreign key violation.
func (r *notifierRepository) delete(ctx context.Context, notifierID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM notifiers WHERE id = ?", notifierID).Error
	if err != nil {
		return fmt.Errorf("delete the notifier: %w", err)
	}

	return nil
}

func (r *notifierRepository) deleteByWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM notifiers WHERE workspace_id = ?", workspaceID).Error
	if err != nil {
		return fmt.Errorf("delete the notifiers of the workspace: %w", err)
	}

	return nil
}
