package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetHealthcheckConfig(
	ctx context.Context,
	request GetHealthcheckConfigRequestObject,
) (GetHealthcheckConfigResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
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
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	config, err := h.configs.SaveConfig(ctx, actor, toConfig(request.DatabaseId, *request.Body))
	if err != nil {
		return nil, err
	}

	return UpdateHealthcheckConfig200JSONResponse(fromConfig(config)), nil
}
