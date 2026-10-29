package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
)

func (h *Handler) GetVerificationConfig(
	ctx context.Context,
	request GetVerificationConfigRequestObject,
) (GetVerificationConfigResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}
	config, err := h.configs.GetConfig(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}
	return GetVerificationConfig200JSONResponse(fromConfig(config)), nil
}

func (h *Handler) UpdateVerificationConfig(
	ctx context.Context,
	request UpdateVerificationConfigRequestObject,
) (UpdateVerificationConfigResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}
	config, err := h.configs.SaveConfig(ctx, actor, toConfig(request.DatabaseId, *request.Body))
	if err != nil {
		return nil, err
	}
	return UpdateVerificationConfig200JSONResponse(fromConfig(config)), nil
}
