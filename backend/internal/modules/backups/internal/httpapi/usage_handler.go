package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetStorageBackupUsage(
	ctx context.Context,
	request GetStorageBackupUsageRequestObject,
) (GetStorageBackupUsageResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	totalUsageMb, err := h.usage.GetStorageUsage(ctx, actor, request.StorageId)
	if err != nil {
		return nil, err
	}

	return GetStorageBackupUsage200JSONResponse{TotalUsageMb: totalUsageMb}, nil
}
