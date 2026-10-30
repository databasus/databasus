package apptest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/storages"
)

func (a *App) RequestStorageDeletionBatch(
	t *testing.T,
	references []storages.StoredFileReference,
) storages.DeletionBatch {
	t.Helper()
	batch, err := a.app.RequestStorageDeletionBatch(
		t.Context(),
		storages.DeletionBatchRequest{ID: uuid.New(), References: references},
	)
	require.NoError(t, err)
	return batch
}

func (a *App) GetStorageDeletionBatch(t *testing.T, id uuid.UUID) storages.DeletionBatch {
	t.Helper()
	batches, err := a.app.GetStorageDeletionBatches(t.Context(), []uuid.UUID{id})
	require.NoError(t, err)
	require.Len(t, batches, 1)
	return batches[0]
}

func (a *App) RefuseStorageDeletionBatches(t *testing.T) {
	t.Helper()
	a.ExecuteSQL(
		t,
		`CREATE FUNCTION refuse_deletion_batch() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'cleanup refused'; END $$ LANGUAGE plpgsql`,
	)
	a.ExecuteSQL(
		t,
		`CREATE TRIGGER refuse_deletion_batch BEFORE INSERT ON storages_deletion_batch_items FOR EACH ROW EXECUTE FUNCTION refuse_deletion_batch()`,
	)
}
