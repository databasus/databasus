package workspaces_services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type MembershipService struct {
	membershipRepository *workspaces_repositories.MembershipRepository
	workspaceRepository  *workspaces_repositories.WorkspaceRepository
	auditLogService      *audit_logs.AuditLogService
	workspaceService     *WorkspaceService
	// getUsersModule is read per call: each test application installs its own module.
	getUsersModule func() users.Module
	emailSender    workspaces_interfaces.EmailSender
	logger         *slog.Logger
}

func (s *MembershipService) SetEmailSender(sender workspaces_interfaces.EmailSender) {
	s.emailSender = sender
}

func (s *MembershipService) GetMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	actor users.Actor,
) (*workspaces_dto.GetMembersResponseDTO, error) {
	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, workspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
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

	targetUser, isTargetUserFound, err := s.getUsersModule().GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}

	if !isTargetUserFound {
		// User doesn't exist, invite them
		canInviteUsers, err := s.getUsersModule().CanInviteUsers(ctx, addedBy)
		if err != nil {
			return nil, err
		}

		if !canInviteUsers {
			return nil, workspaces_errors.ErrInsufficientPermissionsToInviteUsers
		}

		inviter, isInviterFound, err := s.getUsersModule().GetUser(ctx, addedBy.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get inviter: %w", err)
		}

		if !isInviterFound {
			return nil, fmt.Errorf("inviter %s not found", addedBy.UserID)
		}

		// Get workspace details for email
		workspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace: %w", err)
		}

		invitedUser, err := s.getUsersModule().InviteUser(ctx, addedBy, users.Invitation{
			Email:       request.Email,
			WorkspaceID: workspaceID,
		})
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
			UserID:      invitedUser.ID,
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

	if existingMembership.Role == workspaces.RoleOwner {
		return workspaces_errors.ErrCannotChangeOwnerRole
	}

	targetUser, isTargetUserFound, err := s.getUsersModule().GetUser(ctx, memberUserID)
	if err != nil {
		return fmt.Errorf("failed to get the member: %w", err)
	}

	if !isTargetUserFound {
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
	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, removedBy, workspaceID, workspaces.ActionManageMembers)
	if err != nil {
		return err
	}

	if !isAllowed {
		return workspaces_errors.ErrInsufficientPermissionsToRemoveMembers
	}

	existingMembership, err := s.membershipRepository.GetMembershipByUserAndWorkspace(
		memberUserID,
		workspaceID,
	)
	if err != nil {
		return workspaces_errors.ErrUserNotMemberOfWorkspace
	}

	if existingMembership.Role == workspaces.RoleOwner {
		return workspaces_errors.ErrCannotRemoveWorkspaceOwner
	}

	if existingMembership.Role == workspaces.RoleAdmin {
		isAllowedToManageAdmins, err := s.workspaceService.IsActorAllowed(
			ctx,
			removedBy,
			workspaceID,
			workspaces.ActionManageAdmins,
		)
		if err != nil {
			return err
		}
		if !isAllowedToManageAdmins {
			return workspaces_errors.ErrOnlyOwnerCanRemoveAdmins
		}
	}

	targetUser, isTargetUserFound, err := s.getUsersModule().GetUser(ctx, memberUserID)
	if err != nil {
		return fmt.Errorf("failed to get the member: %w", err)
	}

	if !isTargetUserFound {
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
	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, workspaceID, workspaces.ActionTransferOwnership)
	if err != nil {
		return err
	}

	if !isAllowed {
		return workspaces_errors.ErrOnlyOwnerOrAdminCanTransferOwnership
	}

	newOwner, isNewOwnerFound, err := s.getUsersModule().GetUserByEmail(ctx, request.NewOwnerEmail)
	if err != nil {
		return fmt.Errorf("failed to get the new owner: %w", err)
	}

	if !isNewOwnerFound {
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
		workspaces.RoleOwner,
	); err != nil {
		return fmt.Errorf("failed to update new owner role: %w", err)
	}

	if err := s.membershipRepository.UpdateMemberRole(
		currentOwner.UserID,
		workspaceID,
		workspaces.RoleAdmin,
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
	changesRoleTo workspaces.Role,
) error {
	if changesRoleTo == workspaces.RoleAdmin {
		isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, workspaceID, workspaces.ActionManageAdmins)
		if err != nil {
			return err
		}
		if !isAllowed {
			return workspaces_errors.ErrOnlyOwnerCanAddManageAdmins
		}
		return nil
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, workspaceID, workspaces.ActionManageMembers)
	if err != nil {
		return err
	}

	if !isAllowed {
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
