package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
)

func (h *Handler) GetHealthcheckConfig(
	ctx context.Context,
	request GetHealthcheckConfigRequestObject,
) (GetHealthcheckConfigResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	config, err := h.configs.GetConfig(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	return GetHealthcheckConfig200JSONResponse(fromConfig(config)), nil
}

func (h *Handler) UpdateHealthcheckConfig(
	ctx context.Context,
	request UpdateHealthcheckConfigRequestObject,
) (UpdateHealthcheckConfigResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)
	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	config, err := h.configs.SaveConfig(ctx, actor, toConfig(request.DatabaseId, *request.Body))
	if err != nil {
		return nil, err
	}

	return UpdateHealthcheckConfig200JSONResponse(fromConfig(config)), nil
}
