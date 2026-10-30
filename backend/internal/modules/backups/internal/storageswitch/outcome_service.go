package storageswitch

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) GetStorageSwitch(ctx context.Context, actor users.Actor, id uuid.UUID) (Outcome, error) {
	operation, isFound, err := s.repository.get(ctx, id)
	if err != nil {
		return Outcome{}, err
	}
	if !isFound {
		return Outcome{}, ErrNotFound
	}
	if err := s.workspaces.Authorize(ctx, actor, operation.WorkspaceID, workspaces.ActionView); err != nil {
		return Outcome{}, err
	}
	return s.GetOutcome(ctx, operation)
}

func (s *Service) GetOutcome(ctx context.Context, operation StorageSwitch) (Outcome, error) {
	outcomes, err := s.getOutcomes(ctx, []StorageSwitch{operation})
	if err != nil {
		return Outcome{}, err
	}
	return outcomes[0], nil
}

func (s *Service) ListStorageSwitches(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	if err := s.workspaces.Authorize(ctx, actor, query.WorkspaceID, workspaces.ActionView); err != nil {
		return Page{}, err
	}
	limit := 20
	if query.Limit != nil {
		limit = *query.Limit
	}
	if limit < 1 || limit > 100 {
		return Page{}, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "limit", Detail: "must be between 1 and 100"}},
		)
	}
	operations, err := s.repository.list(ctx, query, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{}
	if len(operations) > limit {
		operations = operations[:limit]
		last := operations[len(operations)-1]
		cursor := (database.TimeCursor{Time: last.CreatedAt, ID: last.ID}).Encode()
		page.NextCursor = &cursor
	}
	page.Items, err = s.getOutcomes(ctx, operations)
	return page, err
}

func (s *Service) getOutcomes(ctx context.Context, operations []StorageSwitch) ([]Outcome, error) {
	batchIDs := make([]uuid.UUID, 0, len(operations))
	for _, operation := range operations {
		batchIDs = append(batchIDs, operation.DeletionBatchID)
	}
	batches, err := s.storages.GetDeletionBatches(ctx, batchIDs)
	if err != nil {
		return nil, err
	}
	outcomes := make([]Outcome, 0, len(operations))
	for _, operation := range operations {
		isFound := false
		for _, batch := range batches {
			if batch.ID == operation.DeletionBatchID {
				outcomes = append(outcomes, Outcome{Operation: operation, Cleanup: batch})
				isFound = true
				break
			}
		}
		if !isFound {
			return nil, errMissingBatch
		}
	}
	return outcomes, nil
}
