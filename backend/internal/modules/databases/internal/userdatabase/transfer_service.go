package userdatabase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
)

// TransferDatabaseToWorkspace serves the database transfer of the backups module, which checks access
// in both workspaces itself. The audit event names the actor as its author.
func (s *Service) TransferDatabaseToWorkspace(
	ctx context.Context,
	actor users.Actor,
	databaseID, workspaceID uuid.UUID,
) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		transferredDatabase, isFound, err := s.repository.get(ctx, databaseID, forUpdateRowLock)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrDatabaseNotFound
		}

		err = s.repository.updateWorkspace(ctx, databaseID, workspaceID)
		if err != nil {
			return err
		}

		transferWorkspaces, err := s.workspaces.GetWorkspaces(
			ctx,
			[]uuid.UUID{transferredDatabase.WorkspaceID, workspaceID},
		)
		if err != nil {
			return err
		}

		workspaceNamesByID := make(map[uuid.UUID]string, len(transferWorkspaces))
		for _, transferWorkspace := range transferWorkspaces {
			workspaceNamesByID[transferWorkspace.ID] = transferWorkspace.Name
		}

		return s.audit.record(ctx, actor, workspaceID, fmt.Sprintf(
			"Database transferred: %s from workspace '%s' to workspace '%s'",
			transferredDatabase.Name,
			workspaceNamesByID[transferredDatabase.WorkspaceID],
			workspaceNamesByID[workspaceID],
		))
	})
}

// ReplaceDatabaseNotifiers runs the notifier check of a save, so a transfer of the database can only
// bring notifiers of its new workspace.
func (s *Service) ReplaceDatabaseNotifiers(ctx context.Context, databaseID uuid.UUID, notifierIDs []uuid.UUID) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		attachedDatabase, isFound, err := s.repository.get(ctx, databaseID, forUpdateRowLock)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrDatabaseNotFound
		}

		return s.attachNotifiers(ctx, attachedDatabase, notifierIDs)
	})
}

// OnNotifierTransferring refuses to move a notifier that a database outside the target workspace
// uses. A database transfer moves the database first, so its own notifiers pass.
func (s *Service) OnNotifierTransferring(ctx context.Context, event notifiers.NotifierTransferring) error {
	count, err := s.repository.countUsingNotifierOutsideWorkspace(ctx, event.NotifierID, event.TargetWorkspaceID)
	if err != nil {
		return err
	}

	if count > 0 {
		return ErrNotifierUsedOutsideTargetWorkspace
	}

	return nil
}
