package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
)

func (h *Handler) ListDatabases(
	ctx context.Context,
	request ListDatabasesRequestObject,
) (ListDatabasesResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.databases.ListDatabases(ctx, actor, userdatabase.ListQuery{
		WorkspaceID: request.Params.WorkspaceId,
		NotifierID:  request.Params.NotifierId,
		Limit:       request.Params.Limit,
		After:       request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Database, 0, len(page.Databases))
	for _, listedDatabase := range page.Databases {
		item, err := toDatabaseBody(listedDatabase)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return ListDatabases200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) CreateDatabase(
	ctx context.Context,
	request CreateDatabaseRequestObject,
) (CreateDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	createdDatabase, err := h.databases.CreateDatabase(ctx, actor, userdatabase.Creation{
		WorkspaceID: request.Body.WorkspaceId,
		Name:        request.Body.Name,
		Settings:    settings,
		NotifierIDs: request.Body.NotifierIds,
	})
	if err != nil {
		return nil, err
	}

	body, err := toDatabaseBody(createdDatabase)
	if err != nil {
		return nil, err
	}

	return CreateDatabase201JSONResponse(body), nil
}

func (h *Handler) GetDatabase(
	ctx context.Context,
	request GetDatabaseRequestObject,
) (GetDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	foundDatabase, err := h.databases.GetDatabase(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	body, err := toDatabaseBody(foundDatabase)
	if err != nil {
		return nil, err
	}

	return GetDatabase200JSONResponse(body), nil
}

func (h *Handler) UpdateDatabase(
	ctx context.Context,
	request UpdateDatabaseRequestObject,
) (UpdateDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	updatedDatabase, err := h.databases.UpdateDatabase(ctx, actor, userdatabase.Change{
		DatabaseID:  request.DatabaseId,
		Name:        request.Body.Name,
		Settings:    settings,
		NotifierIDs: request.Body.NotifierIds,
	})
	if err != nil {
		return nil, err
	}

	body, err := toDatabaseBody(updatedDatabase)
	if err != nil {
		return nil, err
	}

	return UpdateDatabase200JSONResponse(body), nil
}

func (h *Handler) DeleteDatabase(
	ctx context.Context,
	request DeleteDatabaseRequestObject,
) (DeleteDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.databases.DeleteDatabase(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	return DeleteDatabase204Response{}, nil
}

func (h *Handler) CopyDatabase(
	ctx context.Context,
	request CopyDatabaseRequestObject,
) (CopyDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	copiedDatabase, err := h.databases.CopyDatabase(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	body, err := toDatabaseBody(copiedDatabase)
	if err != nil {
		return nil, err
	}

	return CopyDatabase201JSONResponse(body), nil
}

func toDatabaseBody(databaseWithSettings userdatabase.WithSettings) (Database, error) {
	settings, err := fromProviderSettings(databaseWithSettings.Settings)
	if err != nil {
		return Database{}, err
	}

	var healthStatus *DatabaseHealthStatus
	if databaseWithSettings.HealthStatus != nil {
		healthStatus = new(DatabaseHealthStatus(*databaseWithSettings.HealthStatus))
	}

	return Database{
		Id:                     databaseWithSettings.ID,
		WorkspaceId:            databaseWithSettings.WorkspaceID,
		Name:                   databaseWithSettings.Name,
		Settings:               settings,
		NotifierIds:            databaseWithSettings.NotifierIDs,
		LastBackupTime:         databaseWithSettings.LastBackupTime,
		LastBackupErrorMessage: databaseWithSettings.LastBackupErrorMessage,
		HealthStatus:           healthStatus,
	}, nil
}
