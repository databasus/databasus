package files

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (s *Store) RequestDeletionBatch(ctx context.Context, request DeletionBatchRequest) (DeletionBatch, error) {
	if !s.repository.IsInTransaction(ctx) {
		return DeletionBatch{}, ErrTransactionRequired
	}

	references := []StoredFileReference{}
	for _, reference := range deduplicateReferences(request.References) {
		if reference.FileName != "" {
			references = append(references, reference)
		}
	}
	request.References = references

	now := s.clock.Now().UTC().Truncate(time.Microsecond)
	batch := DeletionBatch{
		ID:              request.ID,
		Status:          CleanupPending,
		CreatedAt:       now,
		CleanupDeadline: now.Add(24 * time.Hour),
		Total:           int64(len(references)),
		Pending:         int64(len(references)),
	}
	if len(references) == 0 {
		batch.Status = CleanupCompleted
		batch.CompletedAt = &now
	}

	isInserted, err := s.batches.Insert(ctx, batch)
	if err != nil {
		return DeletionBatch{}, err
	}
	if !isInserted {
		batches, readErr := s.batches.GetBatches(ctx, []uuid.UUID{request.ID})
		if readErr != nil {
			return DeletionBatch{}, readErr
		}
		return batches[0], nil
	}

	if err := s.batches.InsertItems(ctx, request, now); err != nil {
		return DeletionBatch{}, err
	}

	return batch, nil
}

func (s *Store) GetDeletionBatches(ctx context.Context, ids []uuid.UUID) ([]DeletionBatch, error) {
	return s.batches.GetBatches(ctx, ids)
}
