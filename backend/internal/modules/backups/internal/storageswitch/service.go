package storageswitch

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Storages     storages.Module
	Workspaces   workspaces.Module
}

type Service struct {
	repository *storageSwitchRepository
	storages   storages.Module
	workspaces workspaces.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.switches,
		storages:   deps.Storages,
		workspaces: deps.Workspaces,
	}
}

func (s *Service) Accept(ctx context.Context, acceptance Acceptance) (*Outcome, error) {
	operation := StorageSwitch{
		ID:                uuid.New(),
		DatabaseID:        acceptance.DatabaseID,
		WorkspaceID:       acceptance.WorkspaceID,
		PreviousStorageID: acceptance.PreviousStorageID,
		TargetStorageID:   acceptance.TargetStorageID,
		DeletionBatchID:   uuid.New(),
	}

	cleanup, err := s.storages.RequestDeletionBatch(
		ctx,
		storages.DeletionBatchRequest{ID: operation.DeletionBatchID, References: acceptance.References},
	)
	if err != nil {
		return nil, err
	}

	operation.CreatedAt = cleanup.CreatedAt
	if err := s.repository.insert(ctx, operation); err != nil {
		return nil, err
	}

	return &Outcome{Operation: operation, Cleanup: cleanup}, nil
}
