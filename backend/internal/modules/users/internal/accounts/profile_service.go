package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users/internal/identity"
)

// emailIndexName is the unique index on the address. Naming it tells a concurrent change to the
// same address apart from other failures.
const emailIndexName = "users_email_idx"

type ProfileUpdate struct {
	Name  string
	Email string
}

// UpdateProfile replaces the name and the address of the actor's own user. The session survives
// the change: the token names the user by ID, not by address.
func (s *Service) UpdateProfile(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	update ProfileUpdate,
) (identity.User, error) {
	if actor.UserID != userID {
		return identity.User{}, ErrNotOwnUser
	}

	var updatedUser UserRecord

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		user, isFound, err := s.repository.getByID(ctx, userID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrUserNotFound
		}

		err = s.validateProfileUpdate(ctx, user, update)
		if err != nil {
			return err
		}

		err = s.repository.updateProfile(ctx, user.ID, update.Name, update.Email)
		if err != nil {
			if isEmailTaken(err) {
				return ErrEmailTaken
			}

			return err
		}

		updatedUser = user
		updatedUser.Name = update.Name
		updatedUser.Email = update.Email

		return s.recordProfileChanges(ctx, user, updatedUser)
	})
	if err != nil {
		return identity.User{}, err
	}

	return updatedUser.ToUser(), nil
}

func (s *Service) ChangePassword(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	newPassword string,
) error {
	if actor.UserID != userID {
		return ErrNotOwnUser
	}

	if !identity.IsPasswordLongEnough(newPassword) {
		return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "newPassword",
			Detail: fmt.Sprintf("must be at least %d characters long", identity.MinPasswordLength),
		}})
	}

	hashedPassword, err := identity.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		user, isFound, err := s.repository.getByID(ctx, userID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrUserNotFound
		}

		return s.ReplacePassword(ctx, user, hashedPassword, "Password changed")
	})
}

// Run inside the caller's transaction so an audit failure also rolls back the password version.
func (s *Service) ReplacePassword(ctx context.Context, user UserRecord, hashedPassword, auditMessage string) error {
	err := s.repository.updatePassword(ctx, passwordReplacement{
		userID: user.ID, hashedPassword: hashedPassword, createdAt: s.clock.Now(),
	})
	if err != nil {
		return err
	}

	return s.recordAuditEvent(ctx, user, auditMessage)
}

// validateProfileUpdate checks a new address only: an upgraded instance's administrator still
// holding the placeholder address can change their name without changing the address.
func (s *Service) validateProfileUpdate(ctx context.Context, user UserRecord, update ProfileUpdate) error {
	var fieldErrors []apierror.FieldError

	if strings.TrimSpace(update.Name) == "" {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "name", Detail: "is required"})
	}

	isEmailChanged := update.Email != user.Email
	if isEmailChanged && !identity.IsEmailAddress(update.Email) {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "email", Detail: "is not an email address"})
	}

	if len(fieldErrors) > 0 {
		return apierror.NewInvalidFieldsProblem(fieldErrors)
	}

	if !isEmailChanged {
		return nil
	}

	_, isTaken, err := s.repository.getByEmail(ctx, update.Email)
	if err != nil {
		return err
	}

	if isTaken {
		return ErrEmailTaken
	}

	return nil
}

func (s *Service) recordProfileChanges(ctx context.Context, previousUser, updatedUser UserRecord) error {
	if previousUser.Email != updatedUser.Email {
		err := s.recordAuditEvent(ctx, updatedUser,
			fmt.Sprintf("Email changed from '%s' to '%s'", previousUser.Email, updatedUser.Email))
		if err != nil {
			return err
		}
	}

	if previousUser.Name != updatedUser.Name {
		err := s.recordAuditEvent(ctx, updatedUser,
			fmt.Sprintf("Name changed from '%s' to '%s'", previousUser.Name, updatedUser.Name))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) recordAuditEvent(ctx context.Context, author UserRecord, message string) error {
	return s.auditLog.Record(ctx, audit.Event{
		Author:      author.ToUser().GetAuditAuthor(),
		WorkspaceID: nil,
		Message:     message,
	})
}

func isEmailTaken(err error) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.ConstraintName == emailIndexName
}
