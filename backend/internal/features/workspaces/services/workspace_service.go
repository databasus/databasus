package workspaces_services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_enums "databasus-backend/internal/features/users/enums"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
	"databasus-backend/internal/modules/users"
)

type WorkspaceService struct {
	workspaceRepository        *workspaces_repositories.WorkspaceRepository
	membershipRepository       *workspaces_repositories.MembershipRepository
	userService                *users_services.UserService
	auditLogService            *audit_logs.AuditLogService
	settingsService            *users_services.SettingsService
	workspaceDeletionListeners []workspaces_interfaces.WorkspaceDeletionListener
}

func (s *WorkspaceService) AddWorkspaceDeletionListener(
	listener workspaces_interfaces.WorkspaceDeletionListener,
) {
	s.workspaceDeletionListeners = append(s.workspaceDeletionListeners, listener)
}

func (s *WorkspaceService) CreateWorkspace(
	ctx context.Context,
	request *workspaces_dto.CreateWorkspaceRequestDTO,
	creator users.Actor,
) (*workspaces_dto.WorkspaceResponseDTO, error) {
	canCreateWorkspaces, err := s.settingsService.CanCreateWorkspaces(ctx, creator)
	if err != nil {
		return nil, err
	}

	if !canCreateWorkspaces {
		return nil, workspaces_errors.ErrInsufficientPermissionsToCreateWorkspaces
	}

	workspace := &workspaces_models.Workspace{
		ID:        uuid.New(),
		Name:      request.Name,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.workspaceRepository.CreateWorkspace(workspace); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	membership := &workspaces_models.WorkspaceMembership{
		UserID:      creator.UserID,
		WorkspaceID: workspace.ID,
		Role:        users_enums.WorkspaceRoleOwner,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.membershipRepository.CreateMembership(membership); err != nil {
		return nil, fmt.Errorf("failed to create workspace membership: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Workspace created: %s", workspace.Name),
		UserID:      &creator.UserID,
		WorkspaceID: &workspace.ID,
	})

	ownerRole := users_enums.WorkspaceRoleOwner
	return &workspaces_dto.WorkspaceResponseDTO{
		ID:        workspace.ID,
		Name:      workspace.Name,
		CreatedAt: workspace.CreatedAt,
		UserRole:  &ownerRole,
	}, nil
}

func (s *WorkspaceService) GetWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (*workspaces_models.Workspace, error) {
	canView, _, err := s.CanUserAccessWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, workspaces_errors.ErrInsufficientPermissionsToViewWorkspace
	}

	return s.workspaceRepository.GetWorkspaceByID(workspaceID)
}

func (s *WorkspaceService) GetUserWorkspaces(
	ctx context.Context,
	actor users.Actor,
) (*workspaces_dto.ListWorkspacesResponseDTO, error) {
	workspaces, err := s.membershipRepository.GetWorkspacesWithRolesByUserID(ctx, actor.Role, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user workspaces: %w", err)
	}

	return &workspaces_dto.ListWorkspacesResponseDTO{
		Workspaces: workspaces,
	}, nil
}

func (s *WorkspaceService) UpdateWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	updateDTO *workspaces_models.Workspace,
	actor users.Actor,
) (*workspaces_models.Workspace, error) {
	canManage, err := s.CanUserManageWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, workspaces_errors.ErrInsufficientPermissionsToUpdateWorkspace
	}

	existingWorkspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	oldName := existingWorkspace.Name

	updateDTO.ID = workspaceID
	updateDTO.CreatedAt = existingWorkspace.CreatedAt

	existingWorkspace.UpdateFromDTO(updateDTO)

	if err := s.workspaceRepository.UpdateWorkspace(existingWorkspace); err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	if oldName != updateDTO.Name {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Workspace updated and renamed from '%s' to '%s'", oldName, updateDTO.Name),
			UserID:      &actor.UserID,
			WorkspaceID: &workspaceID,
		})
	} else {
		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Workspace updated: %s", updateDTO.Name),
			UserID:      &actor.UserID,
			WorkspaceID: &workspaceID,
		})
	}

	return existingWorkspace, nil
}

func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID, actor users.Actor) error {
	if actor.Role != users.RoleAdmin {
		userWorkspaceRole, err := s.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user role: %w", err)
		}

		if userWorkspaceRole == nil || *userWorkspaceRole != users_enums.WorkspaceRoleOwner {
			return workspaces_errors.ErrOnlyOwnerOrAdminCanDeleteWorkspace
		}
	}

	workspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	for _, listener := range s.workspaceDeletionListeners {
		if err := listener.OnBeforeWorkspaceDeletion(workspaceID); err != nil {
			return fmt.Errorf("failed to delete workspace: %w", err)
		}
	}

	if err := s.workspaceRepository.DeleteWorkspace(workspaceID); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Workspace deleted: %s", workspace.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &workspaceID,
	})

	return nil
}

func (s *WorkspaceService) GetUserWorkspaceRole(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (*users_enums.WorkspaceRole, error) {
	return s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, userID)
}

func (s *WorkspaceService) CanUserAccessWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (bool, *users_enums.WorkspaceRole, error) {
	if actor.Role == users.RoleAdmin {
		adminRole := users_enums.WorkspaceRoleOwner
		return true, &adminRole, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return false, nil, nil
	}

	return role != nil, role, nil
}

func (s *WorkspaceService) CanUserManageWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (bool, error) {
	if actor.Role == users.RoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner ||
		*role == users_enums.WorkspaceRoleAdmin, nil
}

func (s *WorkspaceService) CanUserManageDBs(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (bool, error) {
	if actor.Role == users.RoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner ||
		*role == users_enums.WorkspaceRoleAdmin || *role == users_enums.WorkspaceRoleMember, nil
}

func (s *WorkspaceService) CanUserManageMembership(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (bool, error) {
	if actor.Role == users.RoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner || *role == users_enums.WorkspaceRoleAdmin, nil
}

func (s *WorkspaceService) CanUserManageAdmins(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (bool, error) {
	if actor.Role == users.RoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner, nil
}

func (s *WorkspaceService) GetWorkspaceAuditLogs(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
	request *audit_logs.GetAuditLogsRequest,
) (*audit_logs.GetAuditLogsResponse, error) {
	canView, _, err := s.CanUserAccessWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, workspaces_errors.ErrInsufficientPermissionsToViewWorkspaceAuditLogs
	}

	return s.auditLogService.GetWorkspaceAuditLogs(ctx, workspaceID, request)
}

func (s *WorkspaceService) GetAllWorkspaces() ([]*workspaces_models.Workspace, error) {
	return s.workspaceRepository.GetAllWorkspaces()
}

func (s *WorkspaceService) GetWorkspaceByID(
	workspaceID uuid.UUID,
) (*workspaces_models.Workspace, error) {
	return s.workspaceRepository.GetWorkspaceByID(workspaceID)
}
