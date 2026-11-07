package usage

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Storages     storages.Module
	Workspaces   workspaces.Module
}

type Service struct {
	repository *usageRepository
	transactor *database.Transactor
	storages   storages.Module
	workspaces workspaces.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.usage,
		transactor: deps.Transactor,
		storages:   deps.Storages,
		workspaces: deps.Workspaces,
	}
}

func (s *Service) GetDatabaseUsage(ctx context.Context, databaseID uuid.UUID) (float64, error) {
	return s.repository.getDatabaseUsage(ctx, databaseID)
}

func (s *Service) GetStorageUsage(ctx context.Context, actor users.Actor, storageID uuid.UUID) (float64, error) {
	var totalUsageMb float64
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		storage, isFound, err := s.storages.GetStorageBlockingTransfer(ctx, storageID)
		if err != nil {
			return err
		}

		if !isFound {
			return errStorageNotFound
		}

		if err := s.workspaces.Authorize(ctx, actor, storage.WorkspaceID, workspaces.ActionView); err != nil {
			var problem *apierror.Problem
			if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
				return errStorageNotFound
			}

			return err
		}

		totalUsageMb, err = s.repository.getStorageUsage(ctx, storageID)
		return err
	})
	if err != nil {
		return 0, err
	}

	return totalUsageMb, nil
}
