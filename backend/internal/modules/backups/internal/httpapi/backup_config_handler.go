package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) ListBackupConfigs(
	ctx context.Context,
	request ListBackupConfigsRequestObject,
) (ListBackupConfigsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.configs.ListBackupConfigs(ctx, actor, config.ListQuery{
		WorkspaceID: request.Params.WorkspaceId,
		StorageID:   request.Params.StorageId,
		Limit:       request.Params.Limit,
		After:       request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]BackupConfig, 0, len(page.Configs))
	for _, listedConfig := range page.Configs {
		item, err := toBackupConfigBody(listedConfig)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return ListBackupConfigs200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) GetBackupConfig(
	ctx context.Context,
	request GetBackupConfigRequestObject,
) (GetBackupConfigResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	readConfig, err := h.configs.GetBackupConfig(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	body, err := toBackupConfigBody(readConfig)
	if err != nil {
		return nil, err
	}

	return GetBackupConfig200JSONResponse(body), nil
}

func (h *Handler) UpdateBackupConfig(
	ctx context.Context,
	request UpdateBackupConfigRequestObject,
) (UpdateBackupConfigResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	updatedConfig, err := toBackupConfig(request.DatabaseId, *request.Body)
	if err != nil {
		return nil, err
	}

	savedConfig, err := h.configSaves.SaveBackupConfig(ctx, actor, updatedConfig)
	if err != nil {
		return nil, err
	}

	body, err := toBackupConfigBody(savedConfig)
	if err != nil {
		return nil, err
	}

	return UpdateBackupConfig200JSONResponse(body), nil
}
