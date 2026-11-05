package files

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletionWakeSignal(t *testing.T) {
	t.Parallel()

	requests := []struct {
		name    string
		request func(context.Context, *Store, StoredFileReference) error
	}{
		{
			name: "ordinary",
			request: func(ctx context.Context, store *Store, reference StoredFileReference) error {
				return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
			},
		},
		{
			name: "scoped",
			request: func(ctx context.Context, store *Store, reference StoredFileReference) error {
				return store.RequestScopedFileDeletions(ctx, FileDeletionRequest{
					References: []StoredFileReference{reference},
					Scope:      &CleanupScope{DatabaseID: uuid.New(), WorkspaceID: uuid.New()},
				})
			},
		},
		{
			name: "discarded",
			request: func(ctx context.Context, store *Store, reference StoredFileReference) error {
				return store.RequestDiscardedFileDeletions(ctx, DiscardedFiles{
					DatabaseID: uuid.New(),
					References: []StoredFileReference{reference},
				})
			},
		},
	}

	for _, deletionRequest := range requests {
		for _, shouldCommit := range []bool{false, true} {
			scenario := "keeps the deletion worker asleep when a " + deletionRequest.name + " request rolls back"
			if shouldCommit {
				scenario = "wakes the deletion worker only after a " + deletionRequest.name + " request commits"
			}

			t.Run(scenario, func(t *testing.T) {
				t.Parallel()

				files := newTestFiles(t)
				reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "removed.dump"}
				rollbackError := errors.New("rollback deletion")

				err := files.inTransaction(t, func(ctx context.Context) error {
					if requestErr := deletionRequest.request(ctx, files.store, reference); requestErr != nil {
						return requestErr
					}

					assert.Empty(t, files.store.deletionRequests)

					if !shouldCommit {
						return rollbackError
					}

					return nil
				})

				if shouldCommit {
					require.NoError(t, err)
					assert.Len(t, files.store.deletionRequests, 1)
				} else {
					require.ErrorIs(t, err, rollbackError)
					assert.Empty(t, files.store.deletionRequests)
				}
			})
		}
	}
}
