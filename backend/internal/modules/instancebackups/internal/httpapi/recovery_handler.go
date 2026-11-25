package httpapi

import (
	"context"

	"databasus-backend/internal/modules/instancebackups/internal/recovery"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) GetInstanceRecovery(
	ctx context.Context,
	request GetInstanceRecoveryRequestObject,
) (GetInstanceRecoveryResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	instanceRecovery, err := h.recoveries.GetRecovery(ctx, actor, request.InstanceRecoveryId)
	if err != nil {
		return nil, err
	}

	return GetInstanceRecovery200JSONResponse(fromRecovery(instanceRecovery)), nil
}

func (h *Handler) CreateInstanceRecovery(
	ctx context.Context,
	request CreateInstanceRecoveryRequestObject,
) (CreateInstanceRecoveryResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	createdRecovery, err := h.recoveryUploads.Create(ctx, actor, request.Body)
	if err != nil {
		return nil, err
	}

	return CreateInstanceRecovery202JSONResponse(fromRecovery(createdRecovery)), nil
}

func (h *Handler) CancelInstanceRecovery(
	ctx context.Context,
	request CancelInstanceRecoveryRequestObject,
) (CancelInstanceRecoveryResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	cancelledRecovery, err := h.recoveryUploads.Cancel(ctx, actor, request.InstanceRecoveryId)
	if err != nil {
		return nil, err
	}

	return CancelInstanceRecovery200JSONResponse(fromRecovery(cancelledRecovery)), nil
}

func (h *Handler) RequestInstanceRecoveryConfirmation(
	ctx context.Context,
	request RequestInstanceRecoveryConfirmationRequestObject,
) (RequestInstanceRecoveryConfirmationResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	confirmation, err := h.recoveries.RequestConfirmation(ctx, actor, request.InstanceRecoveryId)
	if err != nil {
		return nil, err
	}

	return RequestInstanceRecoveryConfirmation201JSONResponse(toIdentityConfirmationBody(confirmation)), nil
}

func (h *Handler) ApplyInstanceRecovery(
	ctx context.Context,
	request ApplyInstanceRecoveryRequestObject,
) (ApplyInstanceRecoveryResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, errNoActor
	}

	appliedRecovery, err := h.recoveries.Apply(ctx, actor, request.InstanceRecoveryId, recovery.ApplyRequest{
		IsReplacementConfirmed: request.Body.IsReplacementConfirmed,
		Code:                   request.Body.Code,
	})
	if err != nil {
		return nil, err
	}

	return ApplyInstanceRecovery202JSONResponse(fromRecovery(appliedRecovery)), nil
}

func fromRecovery(instanceRecovery recovery.Recovery) InstanceRecovery {
	var failure *InstanceRecoveryFailure
	if instanceRecovery.Failure != nil {
		value := InstanceRecoveryFailure(*instanceRecovery.Failure)
		failure = &value
	}

	return InstanceRecovery{
		Id:                    instanceRecovery.ID,
		Status:                InstanceRecoveryStatus(instanceRecovery.Status),
		Origin:                InstanceRecoveryOrigin(instanceRecovery.Origin),
		ArchiveSha256:         instanceRecovery.ArchiveSha256,
		RunningRelease:        instanceRecovery.RunningRelease,
		SourceRelease:         instanceRecovery.SourceRelease,
		SnapshotAt:            instanceRecovery.SnapshotAt,
		CreatedAt:             instanceRecovery.CreatedAt,
		ExpiresAt:             instanceRecovery.ExpiresAt,
		CompletedAt:           instanceRecovery.CompletedAt,
		Failure:               failure,
		DiscardedCleanupCount: instanceRecovery.DiscardedCleanupCount,
	}
}
