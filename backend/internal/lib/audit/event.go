package audit

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	// Author is nil for an action that ran without a signed-in user.
	Author      *Author
	WorkspaceID *uuid.UUID
	Message     string
}

// Author is the user an event names as the one who acted, as they were at the time of the action.
type Author struct {
	UserID uuid.UUID
	Email  string
	Name   string
}

// RecordedEvent is an event as it was written. Author and WorkspaceID are nil when the event has
// none.
type RecordedEvent struct {
	ID          uuid.UUID
	Author      *Author
	WorkspaceID *uuid.UUID
	Message     string
	CreatedAt   time.Time
}
