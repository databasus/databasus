package workspace

import (
	"context"

	"github.com/google/uuid"
)

// Deleting is published inside the deletion's transaction, before the workspace row goes.
type Deleting struct {
	WorkspaceID uuid.UUID
}

// DeletingHandler runs in the deletion's transaction, which ctx carries; an error rolls the whole
// deletion back.
type DeletingHandler func(ctx context.Context, event Deleting) error
