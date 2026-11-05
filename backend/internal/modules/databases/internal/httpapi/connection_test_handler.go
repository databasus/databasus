package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/databases/internal/connection"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
)

func (h *Handler) TestDatabase(
	ctx context.Context,
	request TestDatabaseRequestObject,
) (TestDatabaseResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	testResult, err := h.connections.TestDatabase(ctx, actor, request.DatabaseId)
	if err != nil {
		return nil, err
	}

	return TestDatabase200JSONResponse(toTestResultBody(testResult)), nil
}

func (h *Handler) TestDatabaseSettings(
	ctx context.Context,
	request TestDatabaseSettingsRequestObject,
) (TestDatabaseSettingsResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	target, err := toSettingsTarget(*request.Body)
	if err != nil {
		return nil, err
	}

	testResult, err := h.connections.TestSettings(ctx, actor, target)
	if err != nil {
		return nil, err
	}

	return TestDatabaseSettings200JSONResponse(toTestResultBody(testResult)), nil
}

func toSettingsTarget(body DatabaseSettingsTarget) (userdatabase.SettingsTarget, error) {
	settings, err := toProviderSettings(body.Settings)
	if err != nil {
		return userdatabase.SettingsTarget{}, err
	}

	return userdatabase.SettingsTarget{
		WorkspaceID: body.WorkspaceId,
		DatabaseID:  body.DatabaseId,
		Settings:    settings,
	}, nil
}

func toTestResultBody(testResult connection.TestResult) DatabaseTestResult {
	var reason *DatabaseConnectionFailure
	if testResult.Reason != nil {
		reason = new(DatabaseConnectionFailure(*testResult.Reason))
	}

	return DatabaseTestResult{
		IsConnected: testResult.IsConnected,
		Reason:      reason,
		Error:       toOptionalValue(testResult.Error),
	}
}
