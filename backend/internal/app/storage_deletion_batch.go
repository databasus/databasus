package app

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

func (a *App) RequestStorageDeletionBatch(
	ctx context.Context,
	request storages.DeletionBatchRequest,
) (storages.DeletionBatch, error) {
	var batch storages.DeletionBatch
	err := a.transactor.Transaction(ctx, func(ctx context.Context) error {
		var err error
		batch, err = a.storagesModule.RequestDeletionBatch(ctx, request)
		return err
	})
	return batch, err
}

func (a *App) GetStorageDeletionBatches(ctx context.Context, ids []uuid.UUID) ([]storages.DeletionBatch, error) {
	return a.storagesModule.GetDeletionBatches(ctx, ids)
}
