// Package workspace creates, lists, reads, renames and deletes workspaces, and publishes
// WorkspaceDeleting inside a deletion.
package workspace

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/workspaces/internal/access"
)

type Workspace struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// ListedWorkspace is a workspace as the actor sees it. CurrentUserRole is nil for an instance admin
// who is not a member of it.
type ListedWorkspace struct {
	Workspace
	CurrentUserRole *access.Role
}
