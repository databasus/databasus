package auditlog

import (
	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
)

const (
	defaultEventPageLimit = 50
	maxEventPageLimit     = 100
)

// EventListQuery leaves a filter nil to not filter by it. The filters choose the view, and the view
// chooses who may read it.
type EventListQuery struct {
	WorkspaceID  *uuid.UUID
	AuthorUserID *uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type EventPage struct {
	Events []ListedEvent
	// NextCursor is nil on the last page.
	NextCursor *string
}

// ListedEvent is an event with its workspace's current name. Workspace is nil for an event outside
// any workspace.
type ListedEvent struct {
	audit.RecordedEvent
	Workspace *EventWorkspace
}

// EventWorkspace has a nil Name when the workspace has been deleted since the event.
type EventWorkspace struct {
	ID   uuid.UUID
	Name *string
}
