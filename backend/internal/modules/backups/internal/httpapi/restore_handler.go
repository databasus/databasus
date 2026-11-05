package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/backups/internal/restore"
)

func (h *Handler) ListRestores(
	ctx context.Context,
	request ListRestoresRequestObject,
) (ListRestoresResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.restores.ListRestores(ctx, actor, restore.ListQuery{
		BackupID: request.Params.BackupId,
		Limit:    request.Params.Limit,
		After:    request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Restore, 0, len(page.Restores))
	for _, listedRestore := range page.Restores {
		items = append(items, toRestoreBody(listedRestore))
	}

	return ListRestores200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) StartRestore(
	ctx context.Context,
	request StartRestoreRequestObject,
) (StartRestoreResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	target, err := toRestoreTarget(request.Body.Target)
	if err != nil {
		return nil, err
	}

	startedRestore, err := h.restores.StartRestore(ctx, actor, request.Body.BackupId, target)
	if err != nil {
		return nil, err
	}

	return StartRestore202JSONResponse(toRestoreBody(startedRestore)), nil
}

func (h *Handler) GetRestore(
	ctx context.Context,
	request GetRestoreRequestObject,
) (GetRestoreResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	readRestore, err := h.restores.GetRestore(ctx, actor, request.RestoreId)
	if err != nil {
		return nil, err
	}

	return GetRestore200JSONResponse(toRestoreBody(readRestore)), nil
}

func (h *Handler) CancelRestore(
	ctx context.Context,
	request CancelRestoreRequestObject,
) (CancelRestoreResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	cancelledRestore, err := h.restores.CancelRestore(ctx, actor, request.RestoreId)
	if err != nil {
		return nil, err
	}

	return CancelRestore202JSONResponse(toRestoreBody(cancelledRestore)), nil
}
