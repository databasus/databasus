package httpapi

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/backups/internal/storageswitch"
)

func (h *Handler) GetStorageSwitch(
	ctx context.Context,
	request GetStorageSwitchRequestObject,
) (GetStorageSwitchResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}
	outcome, err := h.switches.GetStorageSwitch(ctx, actor, request.StorageSwitchId)
	if err != nil {
		return nil, err
	}
	return GetStorageSwitch200JSONResponse(toStorageSwitch(outcome)), nil
}

func (h *Handler) ListStorageSwitches(
	ctx context.Context,
	request ListStorageSwitchesRequestObject,
) (ListStorageSwitchesResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}
	page, err := h.switches.ListStorageSwitches(
		ctx,
		actor,
		storageswitch.ListQuery{
			WorkspaceID: request.Params.WorkspaceId,
			DatabaseID:  request.Params.DatabaseId,
			Limit:       request.Params.Limit,
			After:       request.Params.After,
		},
	)
	if err != nil {
		return nil, err
	}
	items := make([]StorageSwitch, 0, len(page.Items))
	for _, outcome := range page.Items {
		items = append(items, toStorageSwitch(outcome))
	}
	return ListStorageSwitches200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func toStorageSwitch(outcome storageswitch.Outcome) StorageSwitch {
	operation, cleanup := outcome.Operation, outcome.Cleanup
	return StorageSwitch{
		Id:                operation.ID,
		DatabaseId:        operation.DatabaseID,
		WorkspaceId:       operation.WorkspaceID,
		PreviousStorageId: operation.PreviousStorageID,
		TargetStorageId:   operation.TargetStorageID,
		Status:            StorageSwitchStatus(cleanup.Status),
		CreatedAt:         operation.CreatedAt.UTC(),
		CompletedAt:       cleanup.CompletedAt,
		CleanupDeadline:   cleanup.CleanupDeadline.UTC(),
		Total:             cleanup.Total,
		Pending:           cleanup.Pending,
		Deleted:           cleanup.Deleted,
		Remaining:         cleanup.Remaining,
	}
}

func storageSwitchLocation(id uuid.UUID) *string {
	location := "/api/v1/storage-switches/" + id.String()
	return &location
}
