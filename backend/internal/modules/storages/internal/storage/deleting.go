package storage

import (
	"context"

	"github.com/google/uuid"
)

// Deleting is published inside the transaction of a storage deletion or a workspace deletion, with
// the rows of every storage it names locked and before any of them is deleted, so a handler's
// refusal takes precedence over the references and cleanup the deletion checks afterwards.
type Deleting struct {
	StorageIDs []uuid.UUID
}

// DeletingHandler runs in the deletion's transaction, which ctx carries; an error refuses the
// deletion and rolls back what the handler changed.
type DeletingHandler func(ctx context.Context, event Deleting) error
