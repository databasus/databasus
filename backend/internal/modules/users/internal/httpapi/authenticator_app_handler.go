package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users/internal/identity"
)

func (h *Handler) StartAuthenticatorAppSetup(
	ctx context.Context,
	request StartAuthenticatorAppSetupRequestObject,
) (StartAuthenticatorAppSetupResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	startedSetup, err := h.authenticator.StartProfileSetup(ctx, actor, request.UserId, getSecret(request.Body.Code))
	if err != nil {
		return nil, err
	}

	return StartAuthenticatorAppSetup201JSONResponse{
		AuthenticatorAppSetupId: startedSetup.SetupID,
		Secret:                  startedSetup.Secret,
		OtpauthUri:              startedSetup.Address,
		ExpiresAt:               startedSetup.ExpiresAt,
	}, nil
}

func (h *Handler) ConfirmAuthenticatorAppSetup(
	ctx context.Context,
	request ConfirmAuthenticatorAppSetupRequestObject,
) (ConfirmAuthenticatorAppSetupResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	recoveryCodes, err := h.authenticator.ConfirmProfileSetup(
		ctx, actor, request.UserId, request.AuthenticatorAppSetupId, getSecret(request.Body.Code),
	)
	if err != nil {
		return nil, err
	}

	return ConfirmAuthenticatorAppSetup200JSONResponse{RecoveryCodes: recoveryCodes}, nil
}

func (h *Handler) DisconnectAuthenticatorApp(
	ctx context.Context,
	request DisconnectAuthenticatorAppRequestObject,
) (DisconnectAuthenticatorAppResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.accounts.DisconnectAuthenticatorApp(ctx, actor, request.UserId, getSecret(request.Body.Code))
	if err != nil {
		return nil, err
	}

	return DisconnectAuthenticatorApp200JSONResponse(toUserBody(user)), nil
}

func (h *Handler) RegenerateRecoveryCodes(
	ctx context.Context,
	request RegenerateRecoveryCodesRequestObject,
) (RegenerateRecoveryCodesResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	recoveryCodes, err := h.authenticator.RegenerateRecoveryCodes(
		ctx, actor, request.UserId, getSecret(request.Body.Code),
	)
	if err != nil {
		return nil, err
	}

	return RegenerateRecoveryCodes201JSONResponse{RecoveryCodes: recoveryCodes}, nil
}
