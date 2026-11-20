package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// DeleteDatabase removes the database, its settings and its attachments in one transaction, after
// every subscriber of Deleting removed its own rows in it. A subscriber that refuses rolls the whole
// deletion back. Effects outside the database, such as cancelling a running backup or dropping a
// replication slot, are registered by the subscribers as after-commit hooks, so they run only when the
// deletion is kept.
func (s *Service) DeleteDatabase(ctx context.Context, actor users.Actor, databaseID uuid.UUID) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		deletedDatabase, err := s.getAuthorized(ctx, actor, databaseID, workspaces.ActionEditContent, forUpdateRowLock)
		if err != nil {
			return err
		}

		for _, handler := range s.deletingHandlers {
			err := handler(ctx, Deleting{DatabaseID: deletedDatabase.ID})
			if err != nil {
				return err
			}
		}

		err = s.repository.delete(ctx, deletedDatabase.ID)
		if err != nil {
			return err
		}

		return s.audit.record(ctx, actor, deletedDatabase.WorkspaceID, "Database deleted: "+deletedDatabase.Name)
	})
}

// OnWorkspaceDeleting refuses the deletion of a workspace that still has databases, before the later
// handlers delete its notifiers and storages.
func (s *Service) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	hasDatabases, err := s.repository.existsInWorkspace(ctx, event.WorkspaceID)
	if err != nil {
		return err
	}

	if hasDatabases {
		return ErrWorkspaceHasDatabases
	}

	return nil
}
