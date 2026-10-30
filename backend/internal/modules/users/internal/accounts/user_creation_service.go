package accounts

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users/internal/identity"
)

// bootstrapAdminIndexName is the partial unique index that admits one bootstrap administrator.
// Naming it tells a lost race for the empty instance apart from a duplicate address.
const bootstrapAdminIndexName = "idx_users_is_root_admin"

// NewUser describes a user to create. OrdinaryRole is the role the user gets unless it claims
// the empty instance, and CheckOrdinaryUserAllowed is the rule an ordinary user must pass, such
// as the registration setting.
type NewUser struct {
	Email                    string
	Name                     string
	HashedPassword           *string
	GitHubOAuthID            *string
	GoogleOAuthID            *string
	OrdinaryRole             identity.Role
	CheckOrdinaryUserAllowed func(ctx context.Context) error
}

// CreateUserClaimingEmptyInstance creates a user. On an instance without users the user becomes
// its administrator and is recorded as the bootstrap administrator, and the ordinary rule is not
// asked, because an empty instance whose settings forbid registration would otherwise never get
// an administrator.
//
// Losing the race for the empty instance is no failure: the partial unique index admits one
// bootstrap administrator, so the loser is created again as an ordinary user, now under the rule
// it skipped. Run it inside the caller's transaction; the insert runs under a savepoint, so a lost
// race leaves that transaction usable.
func (s *Service) CreateUserClaimingEmptyInstance(ctx context.Context, newUser NewUser) (UserRecord, error) {
	userCount, err := s.repository.countAll(ctx)
	if err != nil {
		return UserRecord{}, err
	}

	isClaimingInstance := userCount == 0

	if !isClaimingInstance {
		err = newUser.CheckOrdinaryUserAllowed(ctx)
		if err != nil {
			return UserRecord{}, err
		}
	}

	record := s.buildUserRecord(newUser, isClaimingInstance)

	err = s.insertUnderSavepoint(ctx, record)
	if err != nil {
		if !isClaimingInstance || !isBootstrapAdminTaken(err) {
			return UserRecord{}, err
		}

		err = newUser.CheckOrdinaryUserAllowed(ctx)
		if err != nil {
			return UserRecord{}, err
		}

		record = s.buildUserRecord(newUser, false)

		err = s.repository.insert(ctx, record)
		if err != nil {
			return UserRecord{}, err
		}
	}

	if record.IsBootstrapAdmin {
		err = s.auditLog.Record(ctx, audit.Event{
			Author:      record.ToUser().GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "First account on the instance now administers it: " + record.Email,
		})
		if err != nil {
			return UserRecord{}, err
		}
	}

	return record, nil
}

// ActivateInvitedUser finishes the sign-up of an invited user with the name and the password
// they chose, and answers the active user.
func (s *Service) ActivateInvitedUser(
	ctx context.Context,
	invitedUser UserRecord,
	name string,
	hashedPassword *string,
) (UserRecord, error) {
	now := s.clock.Now()
	passwordVersion := uuid.New()

	err := s.repository.activateInvited(ctx, invitedUserActivation{
		userID:               invitedUser.ID,
		name:                 name,
		hashedPassword:       hashedPassword,
		passwordCreationTime: now,
		passwordVersion:      passwordVersion,
	})
	if err != nil {
		return UserRecord{}, err
	}

	invitedUser.Status = identity.StatusActive
	invitedUser.Name = name
	invitedUser.HashedPassword = hashedPassword
	invitedUser.PasswordCreationTime = now
	invitedUser.PasswordVersion = passwordVersion

	return invitedUser, nil
}

func (s *Service) insertUnderSavepoint(ctx context.Context, record UserRecord) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		return s.repository.insert(ctx, record)
	})
}

func (s *Service) buildUserRecord(newUser NewUser, isClaimingInstance bool) UserRecord {
	now := s.clock.Now()

	role := newUser.OrdinaryRole
	if isClaimingInstance {
		role = identity.RoleAdmin
	}

	return UserRecord{
		ID:                   uuid.New(),
		Email:                newUser.Email,
		Name:                 newUser.Name,
		HashedPassword:       newUser.HashedPassword,
		PasswordCreationTime: now,
		PasswordVersion:      uuid.New(),
		Role:                 role,
		Status:               identity.StatusActive,
		GitHubOAuthID:        newUser.GitHubOAuthID,
		GoogleOAuthID:        newUser.GoogleOAuthID,
		IsBootstrapAdmin:     isClaimingInstance,
		CreatedAt:            now,
	}
}

func isBootstrapAdminTaken(err error) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.ConstraintName == bootstrapAdminIndexName
}
