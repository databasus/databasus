package accounts

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users/internal/identity"
)

// Directory reads users without any rule of their own. The settings read the administrators
// through it, which is why it exists apart from Service: Service asks the settings who may invite.
type Directory struct {
	repository *userRepository
}

func NewDirectory(repositories Repositories) *Directory {
	return &Directory{repository: repositories.users}
}

func (d *Directory) GetUser(ctx context.Context, userID uuid.UUID) (identity.User, bool, error) {
	record, isFound, err := d.repository.getByID(ctx, userID)
	if err != nil || !isFound {
		return identity.User{}, isFound, err
	}

	return record.ToUser(), true, nil
}

// GetUsers leaves out the IDs that match no user.
func (d *Directory) GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]identity.User, error) {
	if len(userIDs) == 0 {
		return []identity.User{}, nil
	}

	records, err := d.repository.listByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	foundUsers := make([]identity.User, 0, len(records))
	for _, record := range records {
		foundUsers = append(foundUsers, record.ToUser())
	}

	return foundUsers, nil
}

func (d *Directory) GetUserByEmail(ctx context.Context, email string) (identity.User, bool, error) {
	record, isFound, err := d.repository.getByEmail(ctx, email)
	if err != nil || !isFound {
		return identity.User{}, isFound, err
	}

	return record.ToUser(), true, nil
}

func (d *Directory) ListAdmins(ctx context.Context) ([]identity.User, error) {
	records, err := d.repository.listAdmins(ctx)
	if err != nil {
		return nil, err
	}

	admins := make([]identity.User, 0, len(records))
	for _, record := range records {
		admins = append(admins, record.ToUser())
	}

	return admins, nil
}

func (d *Directory) GetUserCount(ctx context.Context) (int64, error) {
	return d.repository.countAll(ctx)
}

// RequireRootAdmin returns the actor's user when it is the active root admin and answers
// permission_denied otherwise. The persisted bootstrap designation decides, never the global role.
func (d *Directory) RequireRootAdmin(ctx context.Context, actor identity.Actor) (identity.User, error) {
	record, isFound, err := d.repository.getByID(ctx, actor.UserID)
	if err != nil {
		return identity.User{}, err
	}

	if !isFound || !record.IsBootstrapAdmin || !record.IsActive() || record.Role != identity.RoleAdmin {
		return identity.User{}, ErrNotRootAdmin
	}

	return record.ToUser(), nil
}

// GetRootAdmin reports false when the instance has no root admin yet.
func (d *Directory) GetRootAdmin(ctx context.Context) (identity.User, bool, error) {
	record, isFound, err := d.repository.getBootstrapAdmin(ctx)
	if err != nil || !isFound {
		return identity.User{}, isFound, err
	}

	return record.ToUser(), true, nil
}
