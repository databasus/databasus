package access

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
)

// Authorizer is the one place where a workspace role meets an action.
type Authorizer struct {
	repository *memberRoleRepository
}

func NewAuthorizer(repositories Repositories) *Authorizer {
	return &Authorizer{repository: repositories.memberRoles}
}

// Authorize lets an instance admin through. Anyone else needs a member in the workspace whose role
// allows the action: without one the answer is ErrWorkspaceNotFound, whether the workspace exists
// or not, and with a role too low it is ErrActionNotAllowed.
func (a *Authorizer) Authorize(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, action Action) error {
	if actor.IsAdmin() {
		return nil
	}

	role, isMember, err := a.repository.getRole(ctx, workspaceID, actor.UserID)
	if err != nil {
		return err
	}

	if !isMember {
		return ErrWorkspaceNotFound
	}

	if !role.IsAllowed(action) {
		return ErrActionNotAllowed
	}

	return nil
}
