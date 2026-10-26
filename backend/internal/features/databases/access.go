package databases_bridge

import (
	"context"

	"github.com/google/uuid"

	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// Access is what the old features need to read a database: the lookup by ID and the access check of
// their own operations. They keep their own errors until each moves into its module.
type Access struct {
	// getModule is read per call: each test application installs its own module.
	getModule       func() databases.Module
	workspaceAccess *workspaces_bridge.Access
}

func GetAccess() *Access {
	return &Access{getModule: GetModule, workspaceAccess: workspaces_bridge.GetAccess()}
}

// GetDatabaseByID returns ErrDatabaseNotFound for an ID that matches no database.
func (a *Access) GetDatabaseByID(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	foundDatabases, err := a.getModule().GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	return foundDatabases[0], nil
}

// GetAuthorizedDatabase returns ErrInsufficientPermissions when the actor may not take the action in
// the database's workspace.
func (a *Access) GetAuthorizedDatabase(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	action workspaces.Action,
) (databases.Database, error) {
	database, err := a.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return databases.Database{}, err
	}

	isAllowed, err := a.workspaceAccess.IsActorAllowed(ctx, actor, database.WorkspaceID, action)
	if err != nil {
		return databases.Database{}, err
	}

	if !isAllowed {
		return databases.Database{}, ErrInsufficientPermissions
	}

	return database, nil
}
