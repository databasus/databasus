package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/settings"
)

func (h *Handler) GetInstanceSettings(
	ctx context.Context,
	_ GetInstanceSettingsRequestObject,
) (GetInstanceSettingsResponseObject, error) {
	instanceSettings, err := h.settings.GetInstanceSettings(ctx)
	if err != nil {
		return nil, err
	}

	return GetInstanceSettings200JSONResponse(toInstanceSettingsBody(instanceSettings)), nil
}

func (h *Handler) UpdateInstanceSettings(
	ctx context.Context,
	request UpdateInstanceSettingsRequestObject,
) (UpdateInstanceSettingsResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	instanceSettings, err := h.settings.UpdateInstanceSettings(ctx, actor, settings.Switches{
		IsAllowExternalRegistrations:      request.Body.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          request.Body.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: request.Body.IsMemberAllowedToCreateWorkspaces,
		IsTwoFactorAuthRequired:           request.Body.IsTwoFactorAuthRequired,
	})
	if err != nil {
		return nil, err
	}

	return UpdateInstanceSettings200JSONResponse(toInstanceSettingsBody(instanceSettings)), nil
}

func (h *Handler) SendTestEmail(
	ctx context.Context,
	_ SendTestEmailRequestObject,
) (SendTestEmailResponseObject, error) {
	actor, err := identity.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.settings.SendTestEmail(ctx, actor)
	if err != nil {
		return nil, err
	}

	body := SendTestEmail201JSONResponse{
		RecipientEmail: result.RecipientEmail,
		IsDelivered:    result.IsDelivered,
		FailureDetail:  nil,
	}

	if !result.IsDelivered {
		body.FailureDetail = &result.FailureDetail
	}

	return body, nil
}

func toInstanceSettingsBody(instanceSettings settings.InstanceSettings) InstanceSettings {
	return InstanceSettings{
		IsAllowExternalRegistrations:      instanceSettings.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          instanceSettings.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: instanceSettings.IsMemberAllowedToCreateWorkspaces,
		IsTwoFactorAuthRequired:           instanceSettings.IsTwoFactorAuthRequired,
		IsEmailConfigured:                 instanceSettings.IsEmailConfigured,
	}
}
