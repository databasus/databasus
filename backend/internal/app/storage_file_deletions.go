package app

import (
	"context"

	"databasus-backend/internal/modules/storages"
)

func (a *App) RequestStorageFileDeletions(ctx context.Context, request storages.FileDeletionRequest) error {
	return a.transactor.Transaction(ctx, func(ctx context.Context) error {
		return a.storagesModule.RequestScopedFileDeletions(ctx, request)
	})
}
