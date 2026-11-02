package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// ResolveSettingsTarget needs the right to change the workspace's content, because the backend connects
// wherever the settings point, and the named database fills in its stored secrets while the resource
// address stays the stored one. A named database that does not exist or
// belongs to another workspace is refused first, then the actor's role, then settings of another type
// or settings a save would refuse, so no stored secret reaches anything before all of them pass.
func (s *Service) ResolveSettingsTarget(
	ctx context.Context,
	actor users.Actor,
	target SettingsTarget,
) (ResolvedTarget, error) {
	if target.WorkspaceID == uuid.Nil {
		return ResolvedTarget{}, ErrWorkspaceMissing
	}

	var storedDatabase *Database

	if target.DatabaseID != nil {
		foundDatabase, isFound, err := s.repository.get(ctx, *target.DatabaseID, noRowLock)
		if err != nil {
			return ResolvedTarget{}, err
		}

		if !isFound || foundDatabase.WorkspaceID != target.WorkspaceID {
			return ResolvedTarget{}, ErrDatabaseNotFound
		}

		storedDatabase = &foundDatabase
	}

	err := s.workspaces.Authorize(ctx, actor, target.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return ResolvedTarget{}, err
	}

	if storedDatabase == nil {
		provider, err := s.registry.GetProvider(target.Settings.GetType())
		if err != nil {
			return ResolvedTarget{}, err
		}

		err = provider.Validate(target.Settings)
		if err != nil {
			return ResolvedTarget{}, err
		}

		return ResolvedTarget{Database: nil, Settings: target.Settings}, nil
	}

	storedWithSettings, err := s.loadOneWithSettings(ctx, *storedDatabase)
	if err != nil {
		return ResolvedTarget{}, err
	}

	mergedSettings, err := s.mergeAndValidate(storedWithSettings, target.Settings)
	if err != nil {
		return ResolvedTarget{}, err
	}

	return ResolvedTarget{Database: storedDatabase, Settings: mergedSettings}, nil
}

// GetAuthorizedWithSettings answers a database of a workspace the actor is not a member of exactly as
// a missing one.
func (s *Service) GetAuthorizedWithSettings(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	action workspaces.Action,
) (WithSettings, error) {
	foundDatabase, err := s.getAuthorized(ctx, actor, databaseID, action, noRowLock)
	if err != nil {
		return WithSettings{}, err
	}

	return s.loadOneWithSettings(ctx, foundDatabase)
}

// RecordAudit writes an audit event of the actor in the workspace, for the services of the module
// that change something outside its tables.
func (s *Service) RecordAudit(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, message string) error {
	return s.audit.record(ctx, actor, workspaceID, message)
}
