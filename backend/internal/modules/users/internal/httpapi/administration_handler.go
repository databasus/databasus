package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
)

func (h *Handler) ListUsers(ctx context.Context, request ListUsersRequestObject) (ListUsersResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	search := ""
	if request.Params.Query != nil {
		search = *request.Params.Query
	}

	page, err := h.accounts.ListUsers(ctx, actor, accounts.UserListQuery{
		Limit:  request.Params.Limit,
		After:  request.Params.After,
		Search: search,
	})
	if err != nil {
		return nil, err
	}

	items := make([]User, 0, len(page.Users))
	for _, user := range page.Users {
		items = append(items, toUserBody(user))
	}

	return ListUsers200JSONResponse{Items: items, NextCursor: page.NextCursor}, nil
}

func (h *Handler) ActivateUser(
	ctx context.Context,
	request ActivateUserRequestObject,
) (ActivateUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.ActivateUser(ctx, actor, request.UserId)
	if err != nil {
		return nil, err
	}

	return ActivateUser200JSONResponse(toUserBody(user)), nil
}

func (h *Handler) DeactivateUser(
	ctx context.Context,
	request DeactivateUserRequestObject,
) (DeactivateUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.DeactivateUser(ctx, actor, request.UserId)
	if err != nil {
		return nil, err
	}

	return DeactivateUser200JSONResponse(toUserBody(user)), nil
}

func (h *Handler) ChangeUserRole(
	ctx context.Context,
	request ChangeUserRoleRequestObject,
) (ChangeUserRoleResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.ChangeUserRole(ctx, actor, request.UserId, identity.Role(request.Body.Role))
	if err != nil {
		return nil, err
	}

	return ChangeUserRole200JSONResponse(toUserBody(user)), nil
}
