package userdatabase

import "github.com/google/uuid"

const (
	defaultDatabasePageLimit = 20
	maxDatabasePageLimit     = 100
)

type ListQuery struct {
	WorkspaceID uuid.UUID
	// NotifierID is nil for every database of the workspace.
	NotifierID *uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type Page struct {
	Databases []WithSettings
	// NextCursor is nil on the last page.
	NextCursor *string
}
