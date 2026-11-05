package transfer

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Configs    *config.Service
	Transactor *database.Transactor
	Workspaces workspaces.Module
	Storages   storages.Module
	Databases  databases.Module
	Notifiers  notifiers.Module
}

// Service runs the database transfer. The modules it calls join its transaction through the context,
// so a failure in any step leaves the database, its config, its storage and its notifiers as they
// were.
type Service struct {
	configs    *config.Service
	transactor *database.Transactor
	workspaces workspaces.Module
	storages   storages.Module
	databases  databases.Module
	notifiers  notifiers.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		configs:    deps.Configs,
		transactor: deps.Transactor,
		workspaces: deps.Workspaces,
		storages:   deps.Storages,
		databases:  deps.Databases,
		notifiers:  deps.Notifiers,
	}
}

// TransferDatabase needs the right to edit content in both workspaces, because the move gives the
// target workspace's members the database's stored secrets.
func (s *Service) TransferDatabase(
	ctx context.Context,
	actor users.Actor,
	transfer DatabaseTransfer,
) error {
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		transferredDatabase, err := s.getDatabase(ctx, transfer.DatabaseID)
		if err != nil {
			return err
		}

		err = s.workspaces.Authorize(ctx, actor, transferredDatabase.WorkspaceID, workspaces.ActionEditContent)

		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
			return ErrDatabaseNotFound
		}

		if err != nil {
			return err
		}

		err = s.workspaces.Authorize(ctx, actor, transfer.TargetWorkspaceID, workspaces.ActionEditContent)
		if err != nil {
			return err
		}

		transferredConfig, err := s.configs.LockBackupConfig(ctx, transfer.DatabaseID)
		if err != nil {
			return err
		}

		// The database moves first, so the storage's transfer check finds it in the target workspace
		// already and lets a storage only this database uses through.
		err = s.databases.TransferDatabaseToWorkspace(ctx, actor, transfer.DatabaseID, transfer.TargetWorkspaceID)
		if err != nil {
			return err
		}

		if transfer.TargetStorageID == nil {
			err = s.moveStorage(ctx, actor, transferredConfig.GetBackupConfig().StorageID, transfer.TargetWorkspaceID)
		} else {
			err = s.switchStorage(ctx, transfer)
		}

		if err != nil {
			return err
		}

		if transfer.IsMovingNotifiers {
			err = s.moveNotifiers(ctx, actor, transferredDatabase.NotifierIDs, transfer.TargetWorkspaceID)
			if err != nil {
				return err
			}
		}

		if len(transfer.TargetNotifierIDs) > 0 {
			return s.databases.ReplaceDatabaseNotifiers(ctx, transfer.DatabaseID, transfer.TargetNotifierIDs)
		}

		return nil
	})
	return err
}

// moveStorage is refused with in_use by the configs' transfer check when another database backs up
// into the storage.
func (s *Service) moveStorage(
	ctx context.Context,
	actor users.Actor,
	storageID *uuid.UUID,
	targetWorkspaceID uuid.UUID,
) error {
	if storageID == nil {
		return ErrDatabaseHasNoStorage
	}

	return s.storages.TransferStorage(ctx, actor, storages.StorageTransfer{
		StorageID:         *storageID,
		TargetWorkspaceID: targetWorkspaceID,
	})
}

func (s *Service) switchStorage(ctx context.Context, transfer DatabaseTransfer) error {
	targetStorage, isFound, err := s.storages.GetStorageBlockingTransfer(ctx, *transfer.TargetStorageID)
	if err != nil {
		return err
	}

	if !isFound || targetStorage.WorkspaceID != transfer.TargetWorkspaceID {
		return ErrTargetStorageNotInTargetWorkspace
	}

	return s.configs.SwitchStorage(
		ctx,
		config.StorageSwitchAssignment{
			DatabaseID:  transfer.DatabaseID,
			StorageID:   targetStorage.ID,
			WorkspaceID: transfer.TargetWorkspaceID,
		},
	)
}

// moveNotifiers moves each notifier under its own savepoint. A notifier another database uses is
// refused with in_use, its savepoint rolls back, and it stays in its workspace.
func (s *Service) moveNotifiers(
	ctx context.Context,
	actor users.Actor,
	notifierIDs []uuid.UUID,
	targetWorkspaceID uuid.UUID,
) error {
	for _, notifierID := range notifierIDs {
		err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
			return s.notifiers.TransferNotifier(ctx, actor, notifiers.NotifierTransfer{
				NotifierID:        notifierID,
				TargetWorkspaceID: targetWorkspaceID,
			})
		})

		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeInUse {
			continue
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	return foundDatabases[0], nil
}
