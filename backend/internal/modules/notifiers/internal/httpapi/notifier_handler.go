package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/notifiers/internal/notifier"
)

func (h *Handler) ListNotifiers(
	ctx context.Context,
	request ListNotifiersRequestObject,
) (ListNotifiersResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := h.notifiers.ListNotifiers(ctx, actor, notifier.ListQuery{
		WorkspaceID: request.Params.WorkspaceId,
		Limit:       request.Params.Limit,
		After:       request.Params.After,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Notifier, 0, len(page.Notifiers))
	for _, listedNotifier := range page.Notifiers {
		item, err := toNotifierBody(listedNotifier)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return ListNotifiers200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) CreateNotifier(
	ctx context.Context,
	request CreateNotifierRequestObject,
) (CreateNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	createdNotifier, err := h.notifiers.CreateNotifier(ctx, actor, notifier.Creation{
		WorkspaceID: request.Body.WorkspaceId,
		Name:        request.Body.Name,
		Settings:    settings,
	})
	if err != nil {
		return nil, err
	}

	body, err := toNotifierBody(createdNotifier)
	if err != nil {
		return nil, err
	}

	return CreateNotifier201JSONResponse(body), nil
}

func (h *Handler) GetNotifier(
	ctx context.Context,
	request GetNotifierRequestObject,
) (GetNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	foundNotifier, err := h.notifiers.GetNotifier(ctx, actor, request.NotifierId)
	if err != nil {
		return nil, err
	}

	body, err := toNotifierBody(foundNotifier)
	if err != nil {
		return nil, err
	}

	return GetNotifier200JSONResponse(body), nil
}

func (h *Handler) UpdateNotifier(
	ctx context.Context,
	request UpdateNotifierRequestObject,
) (UpdateNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	updatedNotifier, err := h.notifiers.UpdateNotifier(ctx, actor, notifier.Change{
		NotifierID: request.NotifierId,
		Name:       request.Body.Name,
		Settings:   settings,
	})
	if err != nil {
		return nil, err
	}

	body, err := toNotifierBody(updatedNotifier)
	if err != nil {
		return nil, err
	}

	return UpdateNotifier200JSONResponse(body), nil
}

func (h *Handler) DeleteNotifier(
	ctx context.Context,
	request DeleteNotifierRequestObject,
) (DeleteNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.notifiers.DeleteNotifier(ctx, actor, request.NotifierId)
	if err != nil {
		return nil, err
	}

	return DeleteNotifier204Response{}, nil
}

func (h *Handler) TransferNotifier(
	ctx context.Context,
	request TransferNotifierRequestObject,
) (TransferNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.transfers.TransferNotifier(ctx, actor, notifier.Transfer{
		NotifierID:        request.NotifierId,
		TargetWorkspaceID: request.Body.TargetWorkspaceId,
	})
	if err != nil {
		return nil, err
	}

	return TransferNotifier204Response{}, nil
}

func toNotifierBody(notifierWithSettings notifier.WithSettings) (Notifier, error) {
	settings, err := fromProviderSettings(notifierWithSettings.Settings)
	if err != nil {
		return Notifier{}, err
	}

	return Notifier{
		Id:            notifierWithSettings.ID,
		WorkspaceId:   notifierWithSettings.WorkspaceID,
		Name:          notifierWithSettings.Name,
		LastSendError: notifierWithSettings.LastSendError,
		Settings:      settings,
	}, nil
}
