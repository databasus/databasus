package accounts

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users/internal/identity"
)

// invitedUserName is the name of an invited user until they finish sign-up with their own.
const invitedUserName = "User"

// Invitation names the address to invite and the workspace the invitation is for.
type Invitation struct {
	Email       string
	WorkspaceID uuid.UUID
}

// InviteUser creates an invited user, who finishes sign-up with a password or an identity
// provider. The actor needs the permission CanInviteUsers grants.
func (s *Service) InviteUser(ctx context.Context, actor identity.Actor, invitation Invitation) (identity.User, error) {
	if !identity.IsEmailAddress(invitation.Email) {
		return identity.User{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{
			{Field: "email", Detail: "is not an email address"},
		})
	}

	canInviteUsers, err := s.settings.CanInviteUsers(ctx, actor)
	if err != nil {
		return identity.User{}, err
	}

	if !canInviteUsers {
		return identity.User{}, ErrInvitationNotAllowed
	}

	var invitedUser UserRecord

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		inviter, isFound, err := s.repository.getByID(ctx, actor.UserID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrUserNotFound
		}

		_, isTaken, err := s.repository.getByEmail(ctx, invitation.Email)
		if err != nil {
			return err
		}

		if isTaken {
			return ErrEmailTaken
		}

		now := s.clock.Now()
		invitedUser = UserRecord{
			ID:                   uuid.New(),
			Email:                invitation.Email,
			Name:                 invitedUserName,
			HashedPassword:       nil,
			PasswordCreationTime: now,
			PasswordVersion:      uuid.New(),
			Role:                 identity.RoleMember,
			Status:               identity.StatusInvited,
			GitHubOAuthID:        nil,
			GoogleOAuthID:        nil,
			IsBootstrapAdmin:     false,
			CreatedAt:            now,
		}

		err = s.repository.insert(ctx, invitedUser)
		if err != nil {
			if isEmailTaken(err) {
				return ErrEmailTaken
			}

			return err
		}

		return s.auditLog.Record(ctx, audit.Event{
			Author:      inviter.ToUser().GetAuditAuthor(),
			WorkspaceID: &invitation.WorkspaceID,
			Message:     "User invited: " + invitedUser.Email + " for workspace",
		})
	})
	if err != nil {
		return identity.User{}, err
	}

	return invitedUser.ToUser(), nil
}
