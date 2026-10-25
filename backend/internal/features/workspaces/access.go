package workspaces_bridge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// Access is what the old features need from the workspaces module: the access check and the names
// of the workspaces their access-checked operations show. They keep their own errors and statuses
// until each moves into its module and calls Authorize itself.
type Access struct {
	// getModule is read per call: each test application installs its own module.
	getModule func() workspaces.Module
}

func GetAccess() *Access {
	return &Access{getModule: GetModule}
}

// IsActorAllowed answers false where Authorize answers not_found or permission_denied.
func (a *Access) IsActorAllowed(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	action workspaces.Action,
) (bool, error) {
	err := a.getModule().Authorize(ctx, actor, workspaceID, action)
	if err == nil {
		return true, nil
	}

	var problem *apierror.Problem
	if errors.As(err, &problem) &&
		(problem.Code == apierror.CodeNotFound || problem.Code == apierror.CodePermissionDenied) {
		return false, nil
	}

	return false, err
}

func (a *Access) GetWorkspaceByID(ctx context.Context, workspaceID uuid.UUID) (workspaces.Workspace, error) {
	foundWorkspaces, err := a.getModule().GetWorkspaces(ctx, []uuid.UUID{workspaceID})
	if err != nil {
		return workspaces.Workspace{}, err
	}

	if len(foundWorkspaces) == 0 {
		return workspaces.Workspace{}, fmt.Errorf("%w: %s", errWorkspaceNotFound, workspaceID)
	}

	return foundWorkspaces[0], nil
}
