package files

import (
	"context"

	"github.com/google/uuid"
)

type CleanupScope struct {
	DatabaseID  uuid.UUID
	WorkspaceID uuid.UUID
}

type FileDeletionRequest struct {
	References []StoredFileReference
	Scope      *CleanupScope
}

type CleanupSummary struct {
	PendingFileCount int64
	FailedFileCount  int64
}

func (s *Store) RequestScopedFileDeletions(ctx context.Context, request FileDeletionRequest) error {
	if !s.repository.IsInTransaction(ctx) {
		return ErrTransactionRequired
	}

	return s.repository.InsertOrTake(ctx, request, s.clock.Now())
}

func (s *Store) GetCleanupSummaries(
	ctx context.Context,
	scopes []CleanupScope,
) (map[CleanupScope]CleanupSummary, error) {
	return s.repository.GetCleanupSummaries(ctx, scopes)
}

func (s *Store) GetStoragePendingDeletions(ctx context.Context, storageID uuid.UUID) ([]PendingDeletion, error) {
	return s.repository.ListByStorage(ctx, storageID)
}
