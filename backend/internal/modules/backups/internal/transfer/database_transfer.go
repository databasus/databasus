// Package transfer moves a database to another workspace together with its storage or onto a storage
// of the target workspace, and with its notifiers, in one transaction.
package transfer

import "github.com/google/uuid"

// DatabaseTransfer names the database, the workspace it moves to and what happens to its storage and
// notifiers.
type DatabaseTransfer struct {
	DatabaseID        uuid.UUID
	TargetWorkspaceID uuid.UUID
	// TargetStorageID switches the database to this storage of the target workspace; nil moves the
	// database's storage along with it.
	TargetStorageID *uuid.UUID
	// IsMovingNotifiers moves every notifier of the database that no other database uses.
	IsMovingNotifiers bool
	// TargetNotifierIDs replace the database's notifiers when set; they must be of the target workspace.
	TargetNotifierIDs []uuid.UUID
}
