package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/transfer"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) TransferDatabase(
	ctx context.Context,
	request TransferDatabaseRequestObject,
) (TransferDatabaseResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	databaseTransfer := transfer.DatabaseTransfer{
		DatabaseID:        request.DatabaseId,
		TargetWorkspaceID: request.Body.TargetWorkspaceId,
		TargetStorageID:   nil,
		IsMovingNotifiers: request.Body.IsMovingNotifiers,
		TargetNotifierIDs: nil,
	}

	if request.Body.TargetNotifierIds != nil {
		databaseTransfer.TargetNotifierIDs = *request.Body.TargetNotifierIds
	}

	storageOption, err := request.Body.Storage.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownStorageOption
	}

	switch typedOption := storageOption.(type) {
	case StorageMovedWithDatabase:
	case StorageSwitchedOnTransfer:
		databaseTransfer.TargetStorageID = &typedOption.StorageId
	default:
		return nil, errUnknownStorageOption
	}

	err = h.transfers.TransferDatabase(ctx, actor, databaseTransfer)
	if err != nil {
		return nil, err
	}

	return TransferDatabase204Response{}, nil
}
