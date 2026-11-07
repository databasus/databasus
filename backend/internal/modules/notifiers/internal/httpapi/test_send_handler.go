package httpapi

import (
	"context"

	"databasus-backend/internal/modules/notifiers/internal/notifier"
	"databasus-backend/internal/modules/users"
)

func (h *Handler) TestNotifier(
	ctx context.Context,
	request TestNotifierRequestObject,
) (TestNotifierResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	testResult, err := h.testSends.TestNotifier(ctx, actor, request.NotifierId)
	if err != nil {
		return nil, err
	}

	return TestNotifier200JSONResponse(toTestResultBody(testResult)), nil
}

func (h *Handler) TestNotifierSettings(
	ctx context.Context,
	request TestNotifierSettingsRequestObject,
) (TestNotifierSettingsResponseObject, error) {
	actor, err := users.RequireActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	testResult, err := h.testSends.TestSettings(ctx, actor, notifier.SettingsTest{
		WorkspaceID: request.Body.WorkspaceId,
		NotifierID:  request.Body.NotifierId,
		Settings:    settings,
	})
	if err != nil {
		return nil, err
	}

	return TestNotifierSettings200JSONResponse(toTestResultBody(testResult)), nil
}

func toTestResultBody(testResult notifier.TestResult) NotifierTestResult {
	return NotifierTestResult{IsDelivered: testResult.IsDelivered, Error: toOptionalString(testResult.Error)}
}
