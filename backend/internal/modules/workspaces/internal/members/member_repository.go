package members

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/workspaces/internal/access"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// service never holds it.
type Repositories struct {
	members *memberRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{members: &memberRepository{db: db}}
}

type memberRepository struct {
	db *gorm.DB
}

const memberColumns = "id, workspace_id, user_id, role, created_at"

func (r *memberRepository) insert(ctx context.Context, member Member) error {
	err := database.Conn(ctx, r.db).Exec(
		"INSERT INTO workspaces_members (id, workspace_id, user_id, role, created_at) VALUES (?, ?, ?, ?, ?)",
		member.ID, member.WorkspaceID, member.UserID, member.Role, member.CreatedAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the member: %w", err)
	}

	return nil
}

// lockByID reads the member and locks the row until the transaction ends, so a concurrent change
// of the same member waits and then sees this one's result.
func (r *memberRepository) lockByID(ctx context.Context, memberID uuid.UUID) (Member, bool, error) {
	var foundMembers []Member

	err := database.Conn(ctx, r.db).Raw(
		"SELECT "+memberColumns+" FROM workspaces_members WHERE id = ? FOR UPDATE",
		memberID,
	).Scan(&foundMembers).Error
	if err != nil {
		return Member{}, false, fmt.Errorf("lock the member: %w", err)
	}

	if len(foundMembers) == 0 {
		return Member{}, false, nil
	}

	return foundMembers[0], true, nil
}

// lockOwner reads the owner of the workspace and locks the row until the transaction ends.
func (r *memberRepository) lockOwner(ctx context.Context, workspaceID uuid.UUID) (Member, bool, error) {
	var foundMembers []Member

	err := database.Conn(ctx, r.db).Raw(
		"SELECT "+memberColumns+" FROM workspaces_members WHERE workspace_id = ? AND role = ? FOR UPDATE",
		workspaceID, access.RoleOwner,
	).Scan(&foundMembers).Error
	if err != nil {
		return Member{}, false, fmt.Errorf("lock the owner of the workspace: %w", err)
	}

	if len(foundMembers) == 0 {
		return Member{}, false, nil
	}

	return foundMembers[0], true, nil
}

func (r *memberRepository) list(ctx context.Context, filter memberListFilter) ([]Member, error) {
	query := database.Conn(ctx, r.db).
		Table("workspaces_members").
		Select(memberColumns).
		Where("workspace_id = ?", filter.workspaceID).
		Order("created_at, id").
		Limit(filter.limit)

	if filter.after != nil {
		query = query.Where("(created_at, id) > (?, ?)", filter.after.createdAt, filter.after.memberID)
	}

	var listedMembers []Member

	err := query.Scan(&listedMembers).Error
	if err != nil {
		return nil, fmt.Errorf("list the members: %w", err)
	}

	return listedMembers, nil
}

// updateRole reports false when no member has the ID.
func (r *memberRepository) updateRole(ctx context.Context, memberID uuid.UUID, role access.Role) (bool, error) {
	result := database.Conn(ctx, r.db).Exec("UPDATE workspaces_members SET role = ? WHERE id = ?", role, memberID)
	if result.Error != nil {
		return false, fmt.Errorf("update the role of the member: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *memberRepository) delete(ctx context.Context, memberID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM workspaces_members WHERE id = ?", memberID).Error
	if err != nil {
		return fmt.Errorf("delete the member: %w", err)
	}

	return nil
}

type memberListFilter struct {
	workspaceID uuid.UUID
	limit       int
	after       *memberCursor
}
