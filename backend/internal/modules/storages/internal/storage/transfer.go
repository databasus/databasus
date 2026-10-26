package storage

import (
	"context"

	"github.com/google/uuid"
)

// Transfer names the storage to move and the workspace it moves to.
type Transfer struct {
	StorageID         uuid.UUID
	TargetWorkspaceID uuid.UUID
}

// Transferring is published inside the transfer's transaction, before the storage moves.
type Transferring struct {
	Transfer
}

// TransferringHandler runs in the transfer's transaction, which ctx carries; an error refuses the
// transfer.
type TransferringHandler func(ctx context.Context, event Transferring) error
