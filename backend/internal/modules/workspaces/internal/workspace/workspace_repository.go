package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/workspaces/internal/access"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// service never holds it.
type Repositories struct {
	workspaces *workspaceRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{workspaces: &workspaceRepository{db: db}}
}

type workspaceRepository struct {
	db *gorm.DB
}

type listedWorkspaceRow struct {
	ID              uuid.UUID
	Name            string
	CreatedAt       time.Time
	CurrentUserRole *access.Role
}

func (r listedWorkspaceRow) toListedWorkspace() ListedWorkspace {
	return ListedWorkspace{
		Workspace:       Workspace{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt},
		CurrentUserRole: r.CurrentUserRole,
	}
}

func (r *workspaceRepository) insert(ctx context.Context, workspace Workspace) error {
	err := database.Conn(ctx, r.db).Exec(
		"INSERT INTO workspaces (id, name, created_at) VALUES (?, ?, ?)",
		workspace.ID, workspace.Name, workspace.CreatedAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the workspace: %w", err)
	}

	return nil
}

func (r *workspaceRepository) insertOwner(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error {
	err := database.Conn(ctx, r.db).Exec(
		"INSERT INTO workspaces_members (id, user_id, workspace_id, role, created_at) VALUES (?, ?, ?, ?, ?)",
		uuid.New(), userID, workspaceID, access.RoleOwner, now,
	).Error
	if err != nil {
		return fmt.Errorf("insert the owner of the workspace: %w", err)
	}

	return nil
}

// getForUser reads the workspace with the user's role in it, nil when the user is not a member.
func (r *workspaceRepository) getForUser(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (ListedWorkspace, bool, error) {
	var rows []listedWorkspaceRow

	err := database.Conn(ctx, r.db).Raw(
		`SELECT w.id, w.name, w.created_at, m.role AS current_user_role
		FROM workspaces w
		LEFT JOIN workspaces_members m ON m.workspace_id = w.id AND m.user_id = ?
		WHERE w.id = ?`,
		userID, workspaceID,
	).Scan(&rows).Error
	if err != nil {
		return ListedWorkspace{}, false, fmt.Errorf("get the workspace: %w", err)
	}

	if len(rows) == 0 {
		return ListedWorkspace{}, false, nil
	}

	return rows[0].toListedWorkspace(), true, nil
}

// lockForDeletion takes a lock that does not conflict with the key-share lock a new row of another
// table takes on the workspace through its foreign key, so content added meanwhile is caught by
// the foreign key when the row goes.
func (r *workspaceRepository) lockForDeletion(ctx context.Context, workspaceID uuid.UUID) (Workspace, bool, error) {
	var workspaces []Workspace

	err := database.Conn(ctx, r.db).
		Raw("SELECT id, name, created_at FROM workspaces WHERE id = ? FOR NO KEY UPDATE", workspaceID).
		Scan(&workspaces).Error
	if err != nil {
		return Workspace{}, false, fmt.Errorf("lock the workspace for deletion: %w", err)
	}

	if len(workspaces) == 0 {
		return Workspace{}, false, nil
	}

	return workspaces[0], true, nil
}

func (r *workspaceRepository) listByIDs(ctx context.Context, workspaceIDs []uuid.UUID) ([]Workspace, error) {
	var workspaces []Workspace

	err := database.Conn(ctx, r.db).
		Raw("SELECT id, name, created_at FROM workspaces WHERE id IN ?", workspaceIDs).
		Scan(&workspaces).Error
	if err != nil {
		return nil, fmt.Errorf("list the workspaces by id: %w", err)
	}

	return workspaces, nil
}

// list reads the workspaces the user is a member of, or every workspace when isEveryWorkspace is
// set, each with the user's role in it.
func (r *workspaceRepository) list(ctx context.Context, filter workspaceListFilter) ([]ListedWorkspace, error) {
	query := `SELECT w.id, w.name, w.created_at, m.role AS current_user_role
		FROM workspaces w
		LEFT JOIN workspaces_members m ON m.workspace_id = w.id AND m.user_id = @userID
		WHERE (@isEveryWorkspace OR m.id IS NOT NULL)`
	values := map[string]any{
		"userID":           filter.userID,
		"isEveryWorkspace": filter.isEveryWorkspace,
		"limit":            filter.limit,
	}

	if filter.after != nil {
		query += " AND (w.name, w.id) > (@afterName, @afterID)"
		values["afterName"] = filter.after.Name
		values["afterID"] = filter.after.ID
	}

	query += " ORDER BY w.name, w.id LIMIT @limit"

	var rows []listedWorkspaceRow

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the workspaces: %w", err)
	}

	listedWorkspaces := make([]ListedWorkspace, 0, len(rows))
	for _, row := range rows {
		listedWorkspaces = append(listedWorkspaces, row.toListedWorkspace())
	}

	return listedWorkspaces, nil
}

func (r *workspaceRepository) updateName(ctx context.Context, workspaceID uuid.UUID, name string) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE workspaces SET name = ? WHERE id = ?", name, workspaceID).Error
	if err != nil {
		return fmt.Errorf("update the name of the workspace: %w", err)
	}

	return nil
}

// delete removes the workspace and, through the module's own cascade, its members.
func (r *workspaceRepository) delete(ctx context.Context, workspaceID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM workspaces WHERE id = ?", workspaceID).Error
	if err != nil {
		return fmt.Errorf("delete the workspace: %w", err)
	}

	return nil
}

type workspaceListFilter struct {
	userID           uuid.UUID
	isEveryWorkspace bool
	limit            int
	after            *database.NameCursor
}
