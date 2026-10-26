package notifier

import "github.com/google/uuid"

const (
	defaultNotifierPageLimit = 20
	maxNotifierPageLimit     = 100
)

type ListQuery struct {
	WorkspaceID uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type Page struct {
	Notifiers []WithSettings
	// NextCursor is nil on the last page.
	NextCursor *string
}
