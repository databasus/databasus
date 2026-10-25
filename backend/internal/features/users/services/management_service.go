package users_services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	user_enums "databasus-backend/internal/features/users/enums"
	user_interfaces "databasus-backend/internal/features/users/interfaces"
	user_models "databasus-backend/internal/features/users/models"
	user_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/modules/users"
)

type UserManagementService struct {
	userRepository *user_repositories.UserRepository
	auditLogWriter user_interfaces.AuditLogWriter
}

func (s *UserManagementService) SetAuditLogWriter(writer user_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *UserManagementService) GetUsers(
	ctx context.Context,
	actor users.Actor,
	limit, offset int,
	beforeCreatedAt *time.Time,
	query string,
) ([]*user_models.User, int64, error) {
	if actor.Role != users.RoleAdmin {
		return nil, 0, errors.New("insufficient permissions to list users")
	}

	return s.userRepository.GetUsers(ctx, limit, offset, beforeCreatedAt, query)
}

func (s *UserManagementService) GetUserProfile(
	ctx context.Context,
	userID uuid.UUID,
	actor users.Actor,
) (*user_models.User, error) {
	// Users can view their own profile, admins can view any profile
	if userID != actor.UserID && actor.Role != users.RoleAdmin {
		return nil, errors.New("insufficient permissions to view user profile")
	}

	return s.userRepository.GetUserByID(ctx, userID)
}

func (s *UserManagementService) DeactivateUser(
	ctx context.Context,
	userID uuid.UUID,
	actor users.Actor,
) error {
	if actor.Role != users.RoleAdmin {
		return errors.New("insufficient permissions to deactivate users")
	}

	// Don't allow deactivating self
	if userID == actor.UserID {
		return errors.New("cannot deactivate your own account")
	}

	user, err := s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only the instance's bootstrap administrator can deactivate ADMIN accounts
	if user.Role == users.RoleAdmin {
		isRootAdmin, err := s.isRootAdmin(ctx, actor)
		if err != nil {
			return err
		}

		if !isRootAdmin {
			return errors.New("only the root admin user can deactivate admin accounts")
		}
	}

	if err := s.userRepository.UpdateUserStatus(userID, user_enums.UserStatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("User deactivated: %s", user.Email),
			UserID:      &actor.UserID,
			WorkspaceID: nil,
		})
	}

	return nil
}

func (s *UserManagementService) ActivateUser(
	ctx context.Context,
	userID uuid.UUID,
	actor users.Actor,
) error {
	if actor.Role != users.RoleAdmin {
		return errors.New("insufficient permissions to activate users")
	}

	user, err := s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only the instance's bootstrap administrator can reactivate ADMIN accounts
	if user.Role == users.RoleAdmin {
		isRootAdmin, err := s.isRootAdmin(ctx, actor)
		if err != nil {
			return err
		}

		if !isRootAdmin {
			return errors.New("only the root admin user can activate admin accounts")
		}
	}

	if err := s.userRepository.UpdateUserStatus(userID, user_enums.UserStatusActive); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("User activated: %s", user.Email),
			UserID:      &actor.UserID,
			WorkspaceID: nil,
		})
	}

	return nil
}

func (s *UserManagementService) ChangeUserRole(
	ctx context.Context,
	userID uuid.UUID,
	newRole users.Role,
	actor users.Actor,
) error {
	if actor.Role != users.RoleAdmin {
		return errors.New("insufficient permissions to change user roles")
	}

	// Validate role
	if !newRole.IsValid() {
		return errors.New("invalid user role")
	}

	// Don't allow changing own role
	if userID == actor.UserID {
		return errors.New("cannot change your own role")
	}

	user, err := s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only the instance's bootstrap administrator can grant or revoke the ADMIN role
	if newRole == users.RoleAdmin || user.Role == users.RoleAdmin {
		isRootAdmin, err := s.isRootAdmin(ctx, actor)
		if err != nil {
			return err
		}

		if !isRootAdmin {
			return errors.New(
				"only the root admin user can promote users to admin or demote admin users",
			)
		}
	}

	if err := s.userRepository.UpdateUserRole(userID, newRole); err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("User role changed: %s from %s to %s", user.Email, user.Role, newRole),
			UserID:      &actor.UserID,
			WorkspaceID: nil,
		})
	}

	return nil
}

func (s *UserManagementService) isRootAdmin(ctx context.Context, actor users.Actor) (bool, error) {
	actingUser, err := s.userRepository.GetUserByID(ctx, actor.UserID)
	if err != nil {
		return false, fmt.Errorf("failed to get acting user: %w", err)
	}

	return actingUser.IsRootAdmin, nil
}
