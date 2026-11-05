package files

import (
	"context"

	"github.com/google/uuid"
)

type DiscardedFiles struct {
	DatabaseID uuid.UUID
	References []StoredFileReference
}

func (s *Store) RequestDiscardedFileDeletions(ctx context.Context, request DiscardedFiles) error {
	if !s.repository.IsInTransaction(ctx) {
		return ErrTransactionRequired
	}

	scopes, err := s.repository.GetOriginalCleanupScopes(ctx, request)
	if err != nil {
		return err
	}

	for _, reference := range request.References {
		var scope *CleanupScope
		if originalScope, hasScope := scopes[reference.StorageID]; hasScope {
			scope = &originalScope
		}

		if err := s.requestFileDeletions(
			ctx,
			FileDeletionRequest{References: []StoredFileReference{reference}, Scope: scope},
		); err != nil {
			return err
		}
	}

	return nil
}
