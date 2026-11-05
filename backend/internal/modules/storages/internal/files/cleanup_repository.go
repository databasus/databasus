package files

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
)

func (r *PendingDeletionRepository) GetCleanupSummaries(
	ctx context.Context,
	scopes []CleanupScope,
) (map[CleanupScope]CleanupSummary, error) {
	summaries := make(map[CleanupScope]CleanupSummary, len(scopes))

	if len(scopes) == 0 {
		return summaries, nil
	}

	predicates := make([]string, 0, len(scopes))
	arguments := make([]any, 0, len(scopes)*2)
	for _, scope := range scopes {
		predicates = append(predicates, "(cleanup_database_id = ? AND cleanup_workspace_id = ?)")
		arguments = append(arguments, scope.DatabaseID, scope.WorkspaceID)
	}

	var counts []struct {
		CleanupScope
		CleanupSummary
	}
	err := database.Conn(ctx, r.db).Raw(`
        SELECT cleanup_database_id AS database_id, cleanup_workspace_id AS workspace_id,
               count(*) AS pending_file_count,
               count(*) FILTER (WHERE last_error IS NOT NULL) AS failed_file_count
        FROM storages_pending_file_deletions
        WHERE `+strings.Join(predicates, " OR ")+`
        GROUP BY cleanup_database_id, cleanup_workspace_id`, arguments...).Scan(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("read current cleanup summaries: %w", err)
	}

	for _, count := range counts {
		summaries[count.CleanupScope] = count.CleanupSummary
	}

	return summaries, nil
}

func (r *PendingDeletionRepository) GetOriginalCleanupScopes(
	ctx context.Context,
	request DiscardedFiles,
) (map[uuid.UUID]CleanupScope, error) {
	scopes := make(map[uuid.UUID]CleanupScope)
	storageIDs := make([]uuid.UUID, 0, len(request.References))
	for _, reference := range request.References {
		storageIDs = append(storageIDs, reference.StorageID)
	}

	if len(storageIDs) == 0 {
		return scopes, nil
	}

	var attributedStorages []struct {
		StorageID uuid.UUID
		CleanupScope
	}
	err := database.Conn(ctx, r.db).Raw(`
        SELECT storage_id, cleanup_database_id AS database_id,
               (array_agg(DISTINCT cleanup_workspace_id))[1] AS workspace_id
        FROM storages_pending_file_deletions
        WHERE cleanup_database_id=? AND storage_id IN ?
        GROUP BY storage_id, cleanup_database_id
        HAVING count(DISTINCT cleanup_workspace_id)=1`, request.DatabaseID, storageIDs).Scan(&attributedStorages).Error
	if err != nil {
		return nil, fmt.Errorf("read discarded file cleanup attribution: %w", err)
	}

	for _, storage := range attributedStorages {
		scopes[storage.StorageID] = storage.CleanupScope
	}

	return scopes, nil
}
