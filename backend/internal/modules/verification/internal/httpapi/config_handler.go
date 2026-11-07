package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetVerificationConfig(
	ctx context.Context,
	request GetVerificationConfigRequestObject,
) (GetVerificationConfigResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
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
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	config, err := h.configs.SaveConfig(ctx, actor, toConfig(request.DatabaseId, *request.Body))
	if err != nil {
		return nil, err
	}
	return UpdateVerificationConfig200JSONResponse(fromConfig(config)), nil
}
