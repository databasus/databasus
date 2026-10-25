package audit

import "github.com/google/uuid"

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
