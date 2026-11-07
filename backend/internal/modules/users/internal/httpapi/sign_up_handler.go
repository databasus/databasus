package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users/internal/signin"
)

func (h *Handler) SignUp(ctx context.Context, request SignUpRequestObject) (SignUpResponseObject, error) {
	session, err := h.signIn.SignUp(ctx, signin.SignUpRequest{
		Email:           string(request.Body.Email),
		Password:        getSecret(request.Body.Password),
		Name:            request.Body.Name,
		TurnstileAnswer: request.Body.CloudflareTurnstileToken,
	})
	if err != nil {
		return nil, err
	}

	return SignUp201JSONResponse{UserId: session.UserID, Email: session.Email, Token: session.Token}, nil
}
