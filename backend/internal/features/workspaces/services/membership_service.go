package workspaces_services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
	"databasus-backend/internal/modules/users"
)

type MembershipService struct {
	membershipRepository *workspaces_repositories.MembershipRepository
	workspaceRepository  *workspaces_repositories.WorkspaceRepository
	userService          *users_services.UserService
	auditLogService      *audit_logs.AuditLogService
	workspaceService     *WorkspaceService
	settingsService      *users_services.SettingsService
	emailSender          workspaces_interfaces.EmailSender
	logger               *slog.Logger
}

func (s *MembershipService) SetEmailSender(sender workspaces_interfaces.EmailSender) {
	s.emailSender = sender
}

func (s *MembershipService) GetMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (*workspaces_dto.GetMembersResponseDTO, error) {
	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, workspaces_errors.ErrInsufficientPermissionsToViewMembers
	}

	members, err := s.membershipRepository.GetWorkspaceMembers(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace members: %w", err)
	}

	membersList := make([]workspaces_dto.WorkspaceMemberResponseDTO, len(members))
	for i, member := range members {
		membersList[i] = *member
	}

	return &workspaces_dto.GetMembersResponseDTO{
		Members: membersList,
	}, nil
}

func (s *MembershipService) AddMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	request *workspaces_dto.AddMemberRequestDTO,
	addedBy users.Actor,
) (*workspaces_dto.AddMemberResponseDTO, error) {
	if err := s.validateCanManageMembership(ctx, workspaceID, addedBy, request.Role); err != nil {
		return nil, err
	}

	targetUser, err := s.userService.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}

	if targetUser == nil {
		// User doesn't exist, invite them
		canInviteUsers, err := s.settingsService.CanInviteUsers(ctx, addedBy)
		if err != nil {
			return nil, err
		}

		if !canInviteUsers {
			return nil, workspaces_errors.ErrInsufficientPermissionsToInviteUsers
		}

		inviter, err := s.userService.GetUserByID(ctx, addedBy.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get inviter: %w", err)
		}

		// Get workspace details for email
		workspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace: %w", err)
		}

		inviteRequest := &users_dto.InviteUserRequestDTO{
			Email:                 request.Email,
			IntendedWorkspaceID:   &workspaceID,
			IntendedWorkspaceRole: &request.Role,
		}

		inviteResponse, err := s.userService.InviteUser(ctx, inviteRequest, addedBy)
		if err != nil {
			return nil, err
		}

		// Send invitation email
		subject := fmt.Sprintf("You've been invited to %s workspace", workspace.Name)
		body := s.buildInvitationEmailHTML(workspace.Name, inviter.Name, string(request.Role))

		if err := s.emailSender.SendEmail(ctx, request.Email, subject, body); err != nil {
			s.logger.ErrorContext(ctx, "failed to send invitation email", "email", request.Email, "error", err)
		}

		membership := &workspaces_models.WorkspaceMembership{
			UserID:      inviteResponse.ID,
			WorkspaceID: workspaceID,
			Role:        request.Role,
		}

		if err := s.membershipRepository.CreateMembership(membership); err != nil {
			return nil, fmt.Errorf("failed to add member: %w", err)
		}

		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("User invited to workspace: %s and added as %s", request.Email, request.Role),
			UserID:      &addedBy.UserID,
			WorkspaceID: &workspaceID,
		})

		return &workspaces_dto.AddMemberResponseDTO{
			Status: workspaces_dto.AddStatusInvited,
		}, nil
	}

	existingMembership, _ := s.membershipRepository.GetMembershipByUserAndWorkspace(
		targetUser.ID,
		workspaceID,
	)
	if existingMembership != nil {
		return nil, workspaces_errors.ErrUserAlreadyMember
	}

	membership := &workspaces_models.WorkspaceMembership{
		UserID:      targetUser.ID,
		WorkspaceID: workspaceID,
		Role:        request.Role,
	}

	if err := s.membershipRepository.CreateMembership(membership); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("User added to workspace: %s as %s", targetUser.Email, request.Role),
		UserID:      &addedBy.UserID,
		WorkspaceID: &workspaceID,
	})

	return &workspaces_dto.AddMemberResponseDTO{
		Status: workspaces_dto.AddStatusAdded,
	}, nil
}

func (s *MembershipService) ChangeMemberRole(
	ctx context.Context,
	workspaceID uuid.UUID,
	memberUserID uuid.UUID,
	request *workspaces_dto.ChangeMemberRoleRequestDTO,
	changedBy users.Actor,
) error {
	if err := s.validateCanManageMembership(ctx, workspaceID, changedBy, request.Role); err != nil {
		return err
	}

	if memberUserID == changedBy.UserID {
		return workspaces_errors.ErrCannotChangeOwnRole
	}

	existingMembership, err := s.membershipRepository.GetMembershipByUserAndWorkspace(
		memberUserID,
		workspaceID,
	)
	if err != nil {
		return workspaces_errors.ErrUserNotMemberOfWorkspace
	}

	if existingMembership.Role == users_enums.WorkspaceRoleOwner {
		return workspaces_errors.ErrCannotChangeOwnerRole
	}

	targetUser, err := s.userService.GetUserByID(ctx, memberUserID)
	if err != nil {
		return workspaces_errors.ErrUserNotFound
	}

	if err := s.membershipRepository.UpdateMemberRole(
		memberUserID,
		workspaceID,
		request.Role,
	); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Member role changed: %s from %s to %s",
			targetUser.Email,
			existingMembership.Role,
			request.Role,
		),
		UserID:      &changedBy.UserID,
		WorkspaceID: &workspaceID,
	})

	return nil
}

func (s *MembershipService) RemoveMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	memberUserID uuid.UUID,
	removedBy users.Actor,
) error {
	canManage, err := s.workspaceService.CanUserManageMembership(ctx, workspaceID, removedBy)
	if err != nil {
		return err
	}

	if !canManage {
		return workspaces_errors.ErrInsufficientPermissionsToRemoveMembers
	}

	existingMembership, err := s.membershipRepository.GetMembershipByUserAndWorkspace(
		memberUserID,
		workspaceID,
	)
	if err != nil {
		return workspaces_errors.ErrUserNotMemberOfWorkspace
	}

	if existingMembership.Role == users_enums.WorkspaceRoleOwner {
		return workspaces_errors.ErrCannotRemoveWorkspaceOwner
	}

	if existingMembership.Role == users_enums.WorkspaceRoleAdmin {
		canManageAdmins, err := s.workspaceService.CanUserManageAdmins(ctx, workspaceID, removedBy)
		if err != nil {
			return err
		}
		if !canManageAdmins {
			return workspaces_errors.ErrOnlyOwnerCanRemoveAdmins
		}
	}

	targetUser, err := s.userService.GetUserByID(ctx, memberUserID)
	if err != nil {
		return workspaces_errors.ErrUserNotFound
	}

	if err := s.membershipRepository.RemoveMember(memberUserID, workspaceID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Member removed from workspace: %s", targetUser.Email),
		UserID:      &removedBy.UserID,
		WorkspaceID: &workspaceID,
	})

	return nil
}

func (s *MembershipService) TransferOwnership(
	ctx context.Context,
	workspaceID uuid.UUID,
	request *workspaces_dto.TransferOwnershipRequestDTO,
	actor users.Actor,
) error {
	currentRole, err := s.membershipRepository.GetUserWorkspaceRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return fmt.Errorf("failed to get current user role: %w", err)
	}

	if actor.Role != users.RoleAdmin &&
		(currentRole == nil || *currentRole != users_enums.WorkspaceRoleOwner) {
		return workspaces_errors.ErrOnlyOwnerOrAdminCanTransferOwnership
	}

	newOwner, err := s.userService.GetUserByEmail(ctx, request.NewOwnerEmail)
	if err != nil {
		return workspaces_errors.ErrNewOwnerNotFound
	}

	if newOwner == nil {
		return workspaces_errors.ErrNewOwnerNotFound
	}

	_, err = s.membershipRepository.GetMembershipByUserAndWorkspace(newOwner.ID, workspaceID)
	if err != nil {
		return workspaces_errors.ErrNewOwnerMustBeMember
	}

	currentOwner, err := s.membershipRepository.GetWorkspaceOwner(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to find current workspace owner: %w", err)
	}

	if currentOwner == nil {
		return workspaces_errors.ErrNoCurrentWorkspaceOwner
	}

	if err := s.membershipRepository.UpdateMemberRole(
		newOwner.ID,
		workspaceID,
		users_enums.WorkspaceRoleOwner,
	); err != nil {
		return fmt.Errorf("failed to update new owner role: %w", err)
	}

	if err := s.membershipRepository.UpdateMemberRole(
		currentOwner.UserID,
		workspaceID,
		users_enums.WorkspaceRoleAdmin,
	); err != nil {
		return fmt.Errorf("failed to update previous owner role: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Workspace ownership transferred to: %s", newOwner.Email),
		UserID:      &actor.UserID,
		WorkspaceID: &workspaceID,
	})

	return nil
}

func (s *MembershipService) validateCanManageMembership(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
	changesRoleTo users_enums.WorkspaceRole,
) error {
	if changesRoleTo == users_enums.WorkspaceRoleAdmin {
		canManageAdmins, err := s.workspaceService.CanUserManageAdmins(ctx, workspaceID, actor)
		if err != nil {
			return err
		}
		if !canManageAdmins {
			return workspaces_errors.ErrOnlyOwnerCanAddManageAdmins
		}
		return nil
	}

	canManageMembership, err := s.workspaceService.CanUserManageMembership(ctx, workspaceID, actor)
	if err != nil {
		return err
	}

	if !canManageMembership {
		return workspaces_errors.ErrInsufficientPermissionsToManageMembers
	}

	return nil
}

func (s *MembershipService) buildInvitationEmailHTML(
	workspaceName, inviterName, role string,
) string {
	env := config.GetEnv()
	signUpLink := ""
	if env.DatabasusURL != "" {
		signUpLink = fmt.Sprintf(`<p style="margin: 20px 0;">
			<a href="%s" style="display: inline-block; padding: 12px 24px; background-color: #0d6efd; color: white; text-decoration: none; border-radius: 4px;">
				Sign up
			</a>
		</p>`, env.DatabasusURL)
	} else {
		signUpLink = `<p style="margin: 20px 0; color: #666;">
			Please visit your Databasus instance to sign up and access the workspace.
		</p>`
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
	<div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px; margin: 20px 0;">
		<h1 style="color: #0d6efd; margin-top: 0;">Workspace Invitation</h1>
		
		<p style="font-size: 16px; margin: 20px 0;">
			<strong>%s</strong> has invited you to join the <strong>%s</strong> workspace as a <strong>%s</strong>.
		</p>
		
		%s
		
		<hr style="border: none; border-top: 1px solid #dee2e6; margin: 30px 0;">
		
		<p style="font-size: 14px; color: #6c757d; margin: 0;">
			This is an automated message from Databasus. If you didn't expect this invitation, you can safely ignore this email.
		</p>
	</div>
</body>
</html>
	`, inviterName, workspaceName, role, signUpLink)
}
