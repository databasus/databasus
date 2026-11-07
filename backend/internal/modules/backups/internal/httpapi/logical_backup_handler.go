package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) ListLogicalBackups(
	ctx context.Context,
	request ListLogicalBackupsRequestObject,
) (ListLogicalBackupsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	query := logical.ListQuery{
		DatabaseID:    request.Params.DatabaseId,
		Statuses:      nil,
		CreatedBefore: request.Params.CreatedBefore,
		Limit:         request.Params.Limit,
		After:         request.Params.After,
	}

	if request.Params.Status != nil {
		for _, status := range *request.Params.Status {
			query.Statuses = append(query.Statuses, logical.Status(status))
		}
	}

	page, err := h.logicalBackups.ListBackups(ctx, actor, query)
	if err != nil {
		return nil, err
	}

	items := make([]LogicalBackup, 0, len(page.Backups))
	for _, listedBackup := range page.Backups {
		items = append(items, toLogicalBackupBody(listedBackup))
	}

	return ListLogicalBackups200JSONResponse{
		Items:        items,
		NextCursor:   page.NextCursor,
		TotalUsageMb: page.TotalUsageMb,
	}, nil
}

func (h *Handler) StartLogicalBackup(
	ctx context.Context,
	request StartLogicalBackupRequestObject,
) (StartLogicalBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	startedBackup, err := h.logicalBackups.StartBackup(ctx, actor, request.Body.DatabaseId)
	if err != nil {
		return nil, err
	}

	return StartLogicalBackup202JSONResponse(toLogicalBackupBody(startedBackup)), nil
}

func (h *Handler) GetLogicalBackup(
	ctx context.Context,
	request GetLogicalBackupRequestObject,
) (GetLogicalBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	readBackup, err := h.logicalBackups.GetBackup(ctx, actor, request.BackupId)
	if err != nil {
		return nil, err
	}

	return GetLogicalBackup200JSONResponse(toLogicalBackupBody(readBackup)), nil
}

func (h *Handler) DeleteLogicalBackup(
	ctx context.Context,
	request DeleteLogicalBackupRequestObject,
) (DeleteLogicalBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.logicalBackups.DeleteBackup(ctx, actor, request.BackupId)
	if err != nil {
		return nil, err
	}

	return DeleteLogicalBackup204Response{}, nil
}

func (h *Handler) CancelLogicalBackup(
	ctx context.Context,
	request CancelLogicalBackupRequestObject,
) (CancelLogicalBackupResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	cancelledBackup, err := h.logicalBackups.CancelBackup(ctx, actor, request.BackupId)
	if err != nil {
		return nil, err
	}

	return CancelLogicalBackup202JSONResponse(toLogicalBackupBody(cancelledBackup)), nil
}
