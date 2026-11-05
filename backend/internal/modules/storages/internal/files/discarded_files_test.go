package files

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestDiscardedFileDeletions(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name       string
		scopeCount int
	}{
		{name: "keeps discarded files unscoped when no attribution remains", scopeCount: 0},
		{name: "inherits attribution when one original workspace remains", scopeCount: 1},
		{name: "keeps discarded files unscoped when original attribution is ambiguous", scopeCount: 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			files := newTestFiles(t)
			storageID, databaseID := files.createStorage(t), uuid.New()
			var originalScope *CleanupScope
			for range scenario.scopeCount {
				originalScope = &CleanupScope{DatabaseID: databaseID, WorkspaceID: uuid.New()}
				require.NoError(t, files.repository.InsertOrTake(t.Context(), FileDeletionRequest{
					References: []StoredFileReference{
						{StorageID: storageID, FileName: originalScope.WorkspaceID.String()},
					},
					Scope: originalScope,
				}, files.clock.Now()))
			}
			discarded := StoredFileReference{StorageID: storageID, FileName: "late-WAL"}

			require.NoError(t, files.inTransaction(t, func(ctx context.Context) error {
				return files.store.RequestDiscardedFileDeletions(
					ctx,
					DiscardedFiles{DatabaseID: databaseID, References: []StoredFileReference{discarded}},
				)
			}))

			pending, isPending := files.findPending(t, discarded)
			require.True(t, isPending)

			if scenario.scopeCount == 1 {
				assert.Equal(t, &originalScope.DatabaseID, pending.CleanupDatabaseID)
				assert.Equal(t, &originalScope.WorkspaceID, pending.CleanupWorkspaceID)
			} else {
				assert.Nil(t, pending.CleanupDatabaseID)
				assert.Nil(t, pending.CleanupWorkspaceID)
			}
		})
	}
}
