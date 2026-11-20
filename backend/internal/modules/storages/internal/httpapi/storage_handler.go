package httpapi

import (
	"context"

	"databasus-backend/internal/modules/storages/internal/storage"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) ListStorages(
	ctx context.Context,
	request ListStoragesRequestObject,
) (ListStoragesResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.storages.ListStorages(ctx, actor, storage.ListQuery{
		WorkspaceID: request.Params.WorkspaceId,
		Limit:       request.Params.Limit,
		After:       request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Storage, 0, len(page.Storages))
	for _, listedStorage := range page.Storages {
		item, err := toStorageBody(listedStorage)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return ListStorages200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) CreateStorage(
	ctx context.Context,
	request CreateStorageRequestObject,
) (CreateStorageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	createdStorage, err := h.storages.CreateStorage(ctx, actor, storage.Creation{
		WorkspaceID: request.Body.WorkspaceId,
		Name:        request.Body.Name,
		Settings:    settings,
	})
	if err != nil {
		return nil, err
	}

	body, err := toStorageBody(createdStorage)
	if err != nil {
		return nil, err
	}

	return CreateStorage201JSONResponse(body), nil
}

func (h *Handler) GetStorage(
	ctx context.Context,
	request GetStorageRequestObject,
) (GetStorageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	foundStorage, err := h.storages.GetStorage(ctx, actor, request.StorageId)
	if err != nil {
		return nil, err
	}

	body, err := toStorageBody(foundStorage)
	if err != nil {
		return nil, err
	}

	return GetStorage200JSONResponse(body), nil
}

func (h *Handler) UpdateStorage(
	ctx context.Context,
	request UpdateStorageRequestObject,
) (UpdateStorageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	updatedStorage, err := h.storages.UpdateStorage(ctx, actor, storage.Change{
		StorageID: request.StorageId,
		Name:      request.Body.Name,
		Settings:  settings,
	})
	if err != nil {
		return nil, err
	}

	body, err := toStorageBody(updatedStorage)
	if err != nil {
		return nil, err
	}

	return UpdateStorage200JSONResponse(body), nil
}

func (h *Handler) DeleteStorage(
	ctx context.Context,
	request DeleteStorageRequestObject,
) (DeleteStorageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.storages.DeleteStorage(
		ctx,
		actor,
		storage.Deletion{
			StorageID:            request.StorageId,
			ShouldAbandonCleanup: request.Params.ShouldAbandonCleanup != nil && *request.Params.ShouldAbandonCleanup,
		},
	)
	if err != nil {
		return nil, err
	}

	return DeleteStorage204Response{}, nil
}

func (h *Handler) TransferStorage(
	ctx context.Context,
	request TransferStorageRequestObject,
) (TransferStorageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.transfers.TransferStorage(ctx, actor, storage.Transfer{
		StorageID:         request.StorageId,
		TargetWorkspaceID: request.Body.TargetWorkspaceId,
	})
	if err != nil {
		return nil, err
	}

	return TransferStorage204Response{}, nil
}

func toStorageBody(storageWithSettings storage.WithSettings) (Storage, error) {
	settings, err := fromProviderSettings(storageWithSettings.Settings)
	if err != nil {
		return Storage{}, err
	}

	return Storage{
		Id:          storageWithSettings.ID,
		WorkspaceId: storageWithSettings.WorkspaceID,
		Name:        storageWithSettings.Name,
		Settings:    settings,
	}, nil
}
