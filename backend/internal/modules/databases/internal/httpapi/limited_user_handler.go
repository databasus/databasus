package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
)

func (h *Handler) CheckLimitedUser(
	ctx context.Context,
	request CheckLimitedUserRequestObject,
) (CheckLimitedUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	target, err := toSettingsTarget(*request.Body)
	if err != nil {
		return nil, err
	}

	limitedUserCheck, err := h.limitedUsers.CheckLimitedUser(ctx, actor, target)
	if err != nil {
		return nil, err
	}

	return CheckLimitedUser200JSONResponse{
		IsAdvisable: limitedUserCheck.IsAdvisable,
		Privileges:  toNonNilStrings(limitedUserCheck.Privileges),
	}, nil
}

func (h *Handler) CreateLimitedUser(
	ctx context.Context,
	request CreateLimitedUserRequestObject,
) (CreateLimitedUserResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	target, err := toSettingsTarget(*request.Body)
	if err != nil {
		return nil, err
	}

	createdUser, err := h.limitedUsers.CreateLimitedUser(ctx, actor, target)
	if err != nil {
		return nil, err
	}

	return CreateLimitedUser201JSONResponse{
		Username:                     createdUser.Username,
		Password:                     createdUser.Password,
		IsForcedWalRotationAvailable: createdUser.IsForcedWalRotationAvailable,
	}, nil
}
