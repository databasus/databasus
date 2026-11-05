package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func (h *Handler) ListPhysicalBackups(
	ctx context.Context,
	request ListPhysicalBackupsRequestObject,
) (ListPhysicalBackupsResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	query := physical.GetPhysicalBackupsRequest{Limit: request.Params.Limit}

	if request.Params.After != nil {
		query.Cursor = *request.Params.After
	}

	query.BeforeDate = request.Params.CreatedBefore

	if request.Params.Type != nil {
		for _, backupType := range *request.Params.Type {
			query.Types = append(query.Types, catalog.BackupType(backupType))
		}
	}

	if request.Params.Status != nil {
		for _, status := range *request.Params.Status {
			query.Statuses = append(query.Statuses, string(status))
		}
	}

	page, err := h.physicalBackups.GetBackups(ctx, actor, request.Params.DatabaseId, &query)
	if err != nil {
		return nil, err
	}

	items := make([]PhysicalBackupListRow, 0, len(page.Backups))
	for _, backup := range page.Backups {
		item, err := toPhysicalBackupListRow(backup)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	response := PhysicalBackupPage{Items: items, TotalUsageMb: page.TotalUsageMb}
	if page.NextCursor != "" {
		response.NextCursor = &page.NextCursor
	}

	return ListPhysicalBackups200JSONResponse(response), nil
}

func (h *Handler) StartPhysicalBackup(
	ctx context.Context,
	request StartPhysicalBackupRequestObject,
) (StartPhysicalBackupResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.physicalBackups.StartBackup(
		ctx,
		actor,
		physical.BackupStart{DatabaseID: request.Body.DatabaseId, Kind: physical.BackupStartKind(request.Body.Kind)},
	)
	if err != nil {
		return nil, err
	}

	return StartPhysicalBackup202Response{}, nil
}

func (h *Handler) CancelPhysicalBackup(
	ctx context.Context,
	request CancelPhysicalBackupRequestObject,
) (CancelPhysicalBackupResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	cancelledBackup, err := h.physicalBackups.CancelBackup(ctx, actor, request.BackupId)
	if err != nil {
		return nil, err
	}

	body, err := toPhysicalBackupListRow(cancelledBackup)
	if err != nil {
		return nil, err
	}

	return CancelPhysicalBackup200JSONResponse(body), nil
}

func (h *Handler) DeletePhysicalBackup(
	ctx context.Context,
	request DeletePhysicalBackupRequestObject,
) (DeletePhysicalBackupResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.physicalBackups.DeleteBackup(ctx, actor, request.BackupId); err != nil {
		return nil, err
	}

	return DeletePhysicalBackup204Response{}, nil
}
