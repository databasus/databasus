package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/users/internal/passwordreset"
)

func (h *Handler) RequestPasswordReset(
	ctx context.Context,
	request RequestPasswordResetRequestObject,
) (RequestPasswordResetResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	err := h.passwordReset.RequestReset(ctx, passwordreset.ResetRequest{
		Email:           request.Body.Email,
		TurnstileAnswer: request.Body.CloudflareTurnstileToken,
	})
	if err != nil {
		return nil, err
	}

	return RequestPasswordReset204Response{}, nil
}

func (h *Handler) CompletePasswordReset(
	ctx context.Context,
	request CompletePasswordResetRequestObject,
) (CompletePasswordResetResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	err := h.passwordReset.CompleteReset(ctx, passwordreset.ResetCompletion{
		Email:       request.Body.Email,
		Code:        request.Body.Code,
		NewPassword: getSecret(request.Body.NewPassword),
	})
	if err != nil {
		return nil, err
	}

	return CompletePasswordReset204Response{}, nil
}
