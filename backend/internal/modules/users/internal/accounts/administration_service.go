package accounts

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users/internal/identity"
)

// ListUsers pages through the users, the newest first.
func (s *Service) ListUsers(ctx context.Context, actor identity.Actor, query UserListQuery) (UserPage, error) {
	if !actor.IsAdmin() {
		return UserPage{}, ErrNotAdmin
	}

	limit := defaultUserPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxUserPageLimit {
		return UserPage{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxUserPageLimit),
		}})
	}

	var after *userCursor

	if query.After != nil {
		cursor, err := decodeUserCursor(*query.After)
		if err != nil {
			return UserPage{}, err
		}

		after = &cursor
	}

	records, err := s.repository.list(ctx, userListFilter{limit: limit + 1, after: after, search: query.Search})
	if err != nil {
		return UserPage{}, err
	}

	page := UserPage{Users: make([]identity.User, 0, min(len(records), limit)), NextCursor: nil}

	for _, record := range records[:min(len(records), limit)] {
		page.Users = append(page.Users, record.ToUser())
	}

	if len(records) > limit {
		lastRecord := records[limit-1]
		nextCursor := userCursor{createdAt: lastRecord.CreatedAt, userID: lastRecord.ID}.encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

func (s *Service) ActivateUser(ctx context.Context, actor identity.Actor, userID uuid.UUID) (identity.User, error) {
	return s.administerUser(ctx, actor, userID, func(ctx context.Context, admin, user UserRecord) (UserRecord, error) {
		if user.Role == identity.RoleAdmin && !admin.IsBootstrapAdmin {
			return UserRecord{}, ErrOnlyBootstrapAdminManagesAdmins
		}

		err := s.repository.updateStatus(ctx, user.ID, identity.StatusActive)
		if err != nil {
			return UserRecord{}, err
		}

		user.Status = identity.StatusActive

		return user, s.recordAuditEvent(ctx, admin, "User activated: "+user.Email)
	})
}

// DeactivateUser ends every session of the user, because a session of a user who is not active is
// not valid.
func (s *Service) DeactivateUser(ctx context.Context, actor identity.Actor, userID uuid.UUID) (identity.User, error) {
	if actor.UserID == userID {
		return identity.User{}, ErrCannotDeactivateYourself
	}

	return s.administerUser(ctx, actor, userID, func(ctx context.Context, admin, user UserRecord) (UserRecord, error) {
		if user.IsBootstrapAdmin {
			return UserRecord{}, ErrBootstrapAdminProtected
		}

		if user.Role == identity.RoleAdmin && !admin.IsBootstrapAdmin {
			return UserRecord{}, ErrOnlyBootstrapAdminManagesAdmins
		}

		err := s.repository.updateStatus(ctx, user.ID, identity.StatusInactive)
		if err != nil {
			return UserRecord{}, err
		}

		user.Status = identity.StatusInactive

		return user, s.recordAuditEvent(ctx, admin, "User deactivated: "+user.Email)
	})
}

func (s *Service) ChangeUserRole(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	newRole identity.Role,
) (identity.User, error) {
	if !newRole.IsValid() {
		return identity.User{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{
			{Field: "role", Detail: "is not a role"},
		})
	}

	if actor.UserID == userID {
		return identity.User{}, ErrCannotChangeOwnRole
	}

	return s.administerUser(ctx, actor, userID, func(ctx context.Context, admin, user UserRecord) (UserRecord, error) {
		if user.IsBootstrapAdmin {
			return UserRecord{}, ErrBootstrapAdminProtected
		}

		if (newRole == identity.RoleAdmin || user.Role == identity.RoleAdmin) && !admin.IsBootstrapAdmin {
			return UserRecord{}, ErrOnlyBootstrapAdminManagesAdmins
		}

		err := s.repository.updateRole(ctx, user.ID, newRole)
		if err != nil {
			return UserRecord{}, err
		}

		previousRole := user.Role
		user.Role = newRole

		return user, s.recordAuditEvent(ctx, admin,
			fmt.Sprintf("User role changed: %s from %s to %s", user.Email, previousRole, newRole))
	})
}

// administerUser loads the administrator and the user in one transaction and applies the change.
// The bootstrap administrator is read from the record, so the powers survive a change of address.
func (s *Service) administerUser(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	change func(ctx context.Context, admin, user UserRecord) (UserRecord, error),
) (identity.User, error) {
	if !actor.IsAdmin() {
		return identity.User{}, ErrNotAdmin
	}

	var changedUser UserRecord

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		admin, isFound, err := s.repository.getByID(ctx, actor.UserID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrNotAdmin
		}

		user, isFound, err := s.repository.getByID(ctx, userID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrUserNotFound
		}

		changedUser, err = change(ctx, admin, user)

		return err
	})
	if err != nil {
		return identity.User{}, err
	}

	return changedUser.ToUser(), nil
}
