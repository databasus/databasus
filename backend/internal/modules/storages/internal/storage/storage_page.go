package storage

import "github.com/google/uuid"

const (
	defaultStoragePageLimit = 20
	maxStoragePageLimit     = 100
)

type ListQuery struct {
	WorkspaceID uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type Page struct {
	Storages []WithSettings
	// NextCursor is nil on the last page.
	NextCursor *string
}
