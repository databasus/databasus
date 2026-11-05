package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type TransferServiceDeps struct {
	Storages     *Service
	Repositories Repositories
	Transactor   *database.Transactor
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
}

// TransferService moves a storage to another workspace after every subscriber of Transferring agreed.
type TransferService struct {
	storages             *Service
	repository           *storageRepository
	transactor           *database.Transactor
	audit                auditRecorder
	workspaces           workspaces.Module
	transferringHandlers []TransferringHandler
}

func NewTransferService(deps TransferServiceDeps) *TransferService {
	return &TransferService{
		storages:             deps.Storages,
		repository:           deps.Repositories.storages,
		transactor:           deps.Transactor,
		audit:                auditRecorder{auditLog: deps.AuditLog, users: deps.Users},
		workspaces:           deps.Workspaces,
		transferringHandlers: nil,
	}
}

// SubscribeToTransferring adds a handler after the ones subscribed before it. internal/app subscribes
// while it builds the application, before any request runs.
func (s *TransferService) SubscribeToTransferring(handler TransferringHandler) {
	s.transferringHandlers = append(s.transferringHandlers, handler)
}

// TransferStorage needs the right to change content in both workspaces. A user outside the source
// workspace learns nothing about the storage, and a viewer of the target is refused. The row lock
// makes a backup config save that read the storage through GetStorageBlockingTransfer finish first,
// so the Transferring handlers see its reference, or wait and then see the storage in its new
// workspace.
func (s *TransferService) TransferStorage(ctx context.Context, actor users.Actor, transfer Transfer) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		transferredStorage, err := s.storages.getAuthorized(
			ctx, actor, transfer.StorageID, workspaces.ActionEditContent, forUpdateRowLock,
		)
		if err != nil {
			return err
		}

		err = s.workspaces.Authorize(ctx, actor, transfer.TargetWorkspaceID, workspaces.ActionEditContent)
		if err != nil {
			return err
		}

		if transferredStorage.WorkspaceID == transfer.TargetWorkspaceID {
			return nil
		}

		for _, handler := range s.transferringHandlers {
			err := handler(ctx, Transferring{Transfer: transfer})
			if err != nil {
				return err
			}
		}

		err = s.repository.updateWorkspace(ctx, transfer.StorageID, transfer.TargetWorkspaceID)
		if err != nil {
			return err
		}

		return s.recordTransfer(ctx, actor, transferredStorage, transfer.TargetWorkspaceID)
	})
}

func (s *TransferService) recordTransfer(
	ctx context.Context,
	actor users.Actor,
	transferredStorage Storage,
	targetWorkspaceID uuid.UUID,
) error {
	transferWorkspaces, err := s.workspaces.GetWorkspaces(
		ctx, []uuid.UUID{transferredStorage.WorkspaceID, targetWorkspaceID},
	)
	if err != nil {
		return err
	}

	workspaceNamesByID := make(map[uuid.UUID]string, len(transferWorkspaces))
	for _, transferWorkspace := range transferWorkspaces {
		workspaceNamesByID[transferWorkspace.ID] = transferWorkspace.Name
	}

	author, err := s.audit.getAuthor(ctx, actor)
	if err != nil {
		return err
	}

	err = s.audit.recordAs(ctx, author, transferredStorage.WorkspaceID, fmt.Sprintf(
		"Storage transferred out: %s to workspace '%s'",
		transferredStorage.Name, workspaceNamesByID[targetWorkspaceID],
	))
	if err != nil {
		return err
	}

	return s.audit.recordAs(ctx, author, targetWorkspaceID, fmt.Sprintf(
		"Storage transferred in: %s from workspace '%s'",
		transferredStorage.Name, workspaceNamesByID[transferredStorage.WorkspaceID],
	))
}
