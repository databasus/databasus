package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users/internal/signin"
)

func (h *Handler) SignIn(ctx context.Context, request SignInRequestObject) (SignInResponseObject, error) {
	outcome, err := h.signIn.SignIn(ctx, signin.SignInRequest{
		Email:           request.Body.Email,
		Password:        getSecret(request.Body.Password),
		TurnstileAnswer: request.Body.CloudflareTurnstileToken,
	})
	if err != nil {
		return nil, err
	}

	body, err := toSignInOutcomeBody(outcome)
	if err != nil {
		return nil, err
	}

	return SignIn200JSONResponse(body), nil
}

func (h *Handler) GetSignInOptions(
	ctx context.Context,
	_ GetSignInOptionsRequestObject,
) (GetSignInOptionsResponseObject, error) {
	options, err := h.signIn.GetSignInOptions(ctx)
	if err != nil {
		return nil, err
	}

	return GetSignInOptions200JSONResponse{HasAnyUser: options.HasAnyUser}, nil
}

func toSignInOutcomeBody(outcome signin.SignInOutcome) (SignInOutcome, error) {
	var body SignInOutcome

	if outcome.PendingSignIn != nil {
		err := body.FromSignInCodeRequired(SignInCodeRequired{
			Outcome:         SignInCodeRequiredOutcomeCODEREQUIRED,
			PendingSignInId: outcome.PendingSignIn.PendingSignInID,
			Email:           outcome.PendingSignIn.Email,
		})

		return body, err
	}

	err := body.FromSignInCompleted(SignInCompleted{
		Outcome: SignInCompletedOutcomeCOMPLETED,
		UserId:  outcome.Session.UserID,
		Email:   outcome.Session.Email,
		Token:   outcome.Session.Token,
	})

	return body, err
}

func (h *Handler) VerifySignInCode(
	ctx context.Context,
	request VerifySignInCodeRequestObject,
) (VerifySignInCodeResponseObject, error) {
	session, err := h.signIn.VerifySignInCode(
		ctx, request.PendingSignInId, getSecret(request.Body.Code), request.Body.CloudflareTurnstileToken,
	)
	if err != nil {
		return nil, err
	}

	return VerifySignInCode200JSONResponse{UserId: session.UserID, Email: session.Email, Token: session.Token}, nil
}

func (h *Handler) ResendSignInCode(
	ctx context.Context,
	request ResendSignInCodeRequestObject,
) (ResendSignInCodeResponseObject, error) {
	pendingSignIn, err := h.signIn.ResendSignInCode(ctx, request.PendingSignInId, request.Body.CloudflareTurnstileToken)
	if err != nil {
		return nil, err
	}

	return ResendSignInCode200JSONResponse{
		PendingSignInId: pendingSignIn.PendingSignInID,
		Email:           pendingSignIn.Email,
	}, nil
}

func (h *Handler) SignInWithGitHub(
	ctx context.Context,
	request SignInWithGitHubRequestObject,
) (SignInWithGitHubResponseObject, error) {
	session, err := h.signIn.SignInWithGitHub(ctx, toOAuthSignInRequest(*request.Body))
	if err != nil {
		return nil, err
	}

	return SignInWithGitHub200JSONResponse{UserId: session.UserID, Email: session.Email, Token: session.Token}, nil
}

func (h *Handler) SignInWithGoogle(
	ctx context.Context,
	request SignInWithGoogleRequestObject,
) (SignInWithGoogleResponseObject, error) {
	session, err := h.signIn.SignInWithGoogle(ctx, toOAuthSignInRequest(*request.Body))
	if err != nil {
		return nil, err
	}

	return SignInWithGoogle200JSONResponse{UserId: session.UserID, Email: session.Email, Token: session.Token}, nil
}

func toOAuthSignInRequest(body OAuthSignInRequest) signin.OAuthSignInRequest {
	return signin.OAuthSignInRequest{Code: body.Code, RedirectURI: body.RedirectUri}
}
