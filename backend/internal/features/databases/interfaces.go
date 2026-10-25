package databases

import (
	"context"

	"github.com/google/uuid"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	users_enums "databasus-backend/internal/features/users/enums"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	"databasus-backend/internal/modules/users"
)

type databaseStore interface {
	Save(database *Database) (*Database, error)
	FindByID(id uuid.UUID) (*Database, error)
	UpdateColumns(id uuid.UUID, columns map[string]any) error
	FindByWorkspaceID(workspaceID uuid.UUID) ([]*Database, error)
	Delete(id uuid.UUID) error
	IsNotifierUsing(notifierID uuid.UUID) (bool, error)
	GetAllDatabases() ([]*Database, error)
	GetDatabasesIDsByNotifierID(notifierID uuid.UUID) ([]uuid.UUID, error)
}

type workspaceService interface {
	CanUserAccessWorkspace(
		ctx context.Context,
		workspaceID uuid.UUID,
		actor users.Actor,
	) (bool, *users_enums.WorkspaceRole, error)
	CanUserManageDBs(ctx context.Context, workspaceID uuid.UUID, actor users.Actor) (bool, error)
	GetWorkspaceByID(workspaceID uuid.UUID) (*workspaces_models.Workspace, error)
}

type DatabaseCreationListener interface {
	OnDatabaseCreated(ctx context.Context, databaseID uuid.UUID)
}

type DatabaseRemoveListener interface {
	OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error
}

type DatabaseCopyListener interface {
	OnDatabaseCopied(ctx context.Context, originalDatabaseID, newDatabaseID uuid.UUID)
}

type BackupTypeChange struct {
	DatabaseID    uuid.UUID
	OldBackupType postgresql_physical.BackupType
	NewBackupType postgresql_physical.BackupType
}

type DatabaseBackupTypeChangeListener interface {
	OnBackupTypeChanged(ctx context.Context, change BackupTypeChange)
}
