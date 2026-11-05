package notifier

import (
	"context"

	"github.com/google/uuid"
)

// Transfer names the notifier to move and the workspace it moves to.
type Transfer struct {
	NotifierID        uuid.UUID
	TargetWorkspaceID uuid.UUID
}

// Transferring is published inside the transfer's transaction, before the notifier moves.
type Transferring struct {
	Transfer
}

// TransferringHandler runs in the transfer's transaction, which ctx carries; an error refuses the
// transfer.
type TransferringHandler func(ctx context.Context, event Transferring) error
