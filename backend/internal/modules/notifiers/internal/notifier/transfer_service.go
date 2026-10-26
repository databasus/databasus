package notifier

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
	Notifiers    *Service
	Repositories Repositories
	Transactor   *database.Transactor
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
}

// TransferService moves a notifier to another workspace after every subscriber of Transferring
// agreed.
type TransferService struct {
	notifiers            *Service
	repository           *notifierRepository
	transactor           *database.Transactor
	audit                auditRecorder
	workspaces           workspaces.Module
	transferringHandlers []TransferringHandler
}

func NewTransferService(deps TransferServiceDeps) *TransferService {
	return &TransferService{
		notifiers:            deps.Notifiers,
		repository:           deps.Repositories.notifiers,
		transactor:           deps.Transactor,
		audit:                auditRecorder{auditLog: deps.AuditLog, users: deps.Users},
		workspaces:           deps.Workspaces,
		transferringHandlers: nil,
	}
}

// SubscribeToTransferring adds a handler after the ones subscribed before it. internal/app
// subscribes while it builds the application, before any request runs.
func (s *TransferService) SubscribeToTransferring(handler TransferringHandler) {
	s.transferringHandlers = append(s.transferringHandlers, handler)
}

// TransferNotifier needs the right to change content in both workspaces. A user outside the source
// workspace learns nothing about the notifier, and a viewer of the target is refused. The notifier
// stays locked from before the Transferring handlers count its databases until the move commits.
func (s *TransferService) TransferNotifier(ctx context.Context, actor users.Actor, transfer Transfer) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.lockForUpdate(ctx, transfer.NotifierID)
		if err != nil {
			return err
		}

		transferredNotifier, err := s.notifiers.getAuthorized(
			ctx, actor, transfer.NotifierID, workspaces.ActionEditContent,
		)
		if err != nil {
			return err
		}

		err = s.workspaces.Authorize(ctx, actor, transfer.TargetWorkspaceID, workspaces.ActionEditContent)
		if err != nil {
			return err
		}

		if transferredNotifier.WorkspaceID == transfer.TargetWorkspaceID {
			return nil
		}

		for _, handler := range s.transferringHandlers {
			err := handler(ctx, Transferring{Transfer: transfer})
			if err != nil {
				return err
			}
		}

		err = s.repository.updateWorkspace(ctx, transfer.NotifierID, transfer.TargetWorkspaceID)
		if err != nil {
			return err
		}

		message, err := s.describeTransfer(ctx, transferredNotifier, transfer.TargetWorkspaceID)
		if err != nil {
			return err
		}

		return s.audit.record(ctx, actor, transfer.TargetWorkspaceID, message)
	})
}

func (s *TransferService) describeTransfer(
	ctx context.Context,
	transferredNotifier Notifier,
	targetWorkspaceID uuid.UUID,
) (string, error) {
	transferWorkspaces, err := s.workspaces.GetWorkspaces(
		ctx, []uuid.UUID{transferredNotifier.WorkspaceID, targetWorkspaceID},
	)
	if err != nil {
		return "", err
	}

	workspaceNamesByID := make(map[uuid.UUID]string, len(transferWorkspaces))
	for _, transferWorkspace := range transferWorkspaces {
		workspaceNamesByID[transferWorkspace.ID] = transferWorkspace.Name
	}

	return fmt.Sprintf(
		"Notifier transferred: %s from workspace '%s' to workspace '%s'",
		transferredNotifier.Name,
		workspaceNamesByID[transferredNotifier.WorkspaceID],
		workspaceNamesByID[targetWorkspaceID],
	), nil
}
