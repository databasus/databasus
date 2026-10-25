package workspaces_repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	users_enums "databasus-backend/internal/features/users/enums"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

type MembershipRepository struct{}

func (r *MembershipRepository) CreateMembership(
	membership *workspaces_models.WorkspaceMembership,
) error {
	if membership.ID == uuid.Nil {
		membership.ID = uuid.New()
	}

	if membership.CreatedAt.IsZero() {
		membership.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(membership).Error
}

func (r *MembershipRepository) GetMembershipByUserAndWorkspace(
	userID, workspaceID uuid.UUID,
) (*workspaces_models.WorkspaceMembership, error) {
	var membership workspaces_models.WorkspaceMembership

	if err := storage.GetDb().
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&membership).Error; err != nil {
		return nil, err
	}

	return &membership, nil
}

func (r *MembershipRepository) GetWorkspaceMembers(
	workspaceID uuid.UUID,
) ([]*workspaces_dto.WorkspaceMemberResponseDTO, error) {
	var members []*workspaces_dto.WorkspaceMemberResponseDTO

	err := storage.GetDb().
		Table("workspace_memberships wm").
		Select("wm.id, wm.user_id, u.email, u.name, wm.role, wm.created_at").
		Joins("JOIN users u ON wm.user_id = u.id").
		Where("wm.workspace_id = ?", workspaceID).
		Order("wm.created_at ASC").
		Scan(&members).Error

	return members, err
}

func (r *MembershipRepository) UpdateMemberRole(
	userID, workspaceID uuid.UUID,
	role users_enums.WorkspaceRole,
) error {
	return storage.GetDb().
		Model(&workspaces_models.WorkspaceMembership{}).
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		Update("role", role).Error
}

func (r *MembershipRepository) RemoveMember(userID, workspaceID uuid.UUID) error {
	return storage.GetDb().
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		Delete(&workspaces_models.WorkspaceMembership{}).Error
}

func (r *MembershipRepository) GetUserWorkspaceRole(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*users_enums.WorkspaceRole, error) {
	var membership workspaces_models.WorkspaceMembership
	err := storage.GetDb().WithContext(ctx).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&membership).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &membership.Role, nil
}

func (r *MembershipRepository) GetWorkspaceOwner(
	workspaceID uuid.UUID,
) (*workspaces_models.WorkspaceMembership, error) {
	var membership workspaces_models.WorkspaceMembership

	err := storage.GetDb().
		Where("workspace_id = ? AND role = ?", workspaceID, users_enums.WorkspaceRoleOwner).
		First(&membership).Error
	if err != nil {
		return nil, err
	}

	return &membership, nil
}

func (r *MembershipRepository) GetWorkspacesWithRolesByUserID(
	ctx context.Context,
	userRole users.Role,
	userID uuid.UUID,
) ([]workspaces_dto.WorkspaceResponseDTO, error) {
	results := make([]workspaces_dto.WorkspaceResponseDTO, 0)

	if userRole == users.RoleAdmin {
		err := storage.GetDb().WithContext(ctx).Table("workspaces").Order("name ASC").Scan(&results).Error
		return results, err
	}

	err := storage.GetDb().WithContext(ctx).
		Table("workspaces w").
		Select("w.id, w.name, w.created_at, wm.role as user_role").
		Joins("JOIN workspace_memberships wm ON w.id = wm.workspace_id").
		Where("wm.user_id = ?", userID).
		Order("w.name ASC").
		Scan(&results).Error

	return results, err
}
