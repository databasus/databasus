package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification/internal/verifications"
)

func (h *Handler) StartVerification(
	ctx context.Context,
	request StartVerificationRequestObject,
) (StartVerificationResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	verification, err := h.verifications.StartVerification(ctx, actor, request.Body.BackupId)
	if err != nil {
		return nil, err
	}
	return StartVerification202JSONResponse(fromVerification(verification)), nil
}

func (h *Handler) CancelVerification(
	ctx context.Context,
	request CancelVerificationRequestObject,
) (CancelVerificationResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	verification, err := h.verifications.CancelVerification(ctx, actor, request.VerificationId)
	if err != nil {
		return nil, err
	}
	return CancelVerification200JSONResponse(fromVerification(verification)), nil
}

func (h *Handler) GetVerification(
	ctx context.Context,
	request GetVerificationRequestObject,
) (GetVerificationResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	verification, err := h.verifications.GetVerification(ctx, actor, request.VerificationId)
	if err != nil {
		return nil, err
	}
	return GetVerification200JSONResponse(fromDetails(verification)), nil
}

func (h *Handler) ListVerifications(
	ctx context.Context,
	request ListVerificationsRequestObject,
) (ListVerificationsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}
	page, err := h.verifications.ListVerifications(
		ctx,
		actor,
		verifications.ListQuery{
			DatabaseID: request.Params.DatabaseId,
			Limit:      request.Params.Limit,
			After:      request.Params.After,
		},
	)
	if err != nil {
		return nil, err
	}
	return ListVerifications200JSONResponse{
		Items:      fromVerifications(page.Verifications),
		NextCursor: page.NextCursor,
	}, nil
}
