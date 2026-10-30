package storageswitch

import (
	"github.com/google/uuid"
)

type ListQuery struct {
	WorkspaceID uuid.UUID
	DatabaseID  *uuid.UUID
	Limit       *int
	After       *string
}

type Page struct {
	Items      []Outcome
	NextCursor *string
}
