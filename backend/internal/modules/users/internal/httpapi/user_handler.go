package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/users/internal/accounts"
)

func (h *Handler) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.GetUser(ctx, actor, request.UserId)
	if err != nil {
		return nil, err
	}

	return GetUser200JSONResponse(toUserBody(user)), nil
}

func (h *Handler) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.UpdateProfile(ctx, actor, request.UserId, accounts.ProfileUpdate{
		Name:  request.Body.Name,
		Email: request.Body.Email,
	})
	if err != nil {
		return nil, err
	}

	return UpdateUser200JSONResponse(toUserBody(user)), nil
}

func (h *Handler) ChangeUserPassword(
	ctx context.Context,
	request ChangeUserPasswordRequestObject,
) (ChangeUserPasswordResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	err = h.accounts.ChangePassword(ctx, actor, request.UserId, getSecret(request.Body.NewPassword))
	if err != nil {
		return nil, err
	}

	return ChangeUserPassword204Response{}, nil
}
