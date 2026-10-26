package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/notifiers/internal/notifier"
)

func (h *Handler) TestNotifier(
	ctx context.Context,
	request TestNotifierRequestObject,
) (TestNotifierResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
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
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
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
