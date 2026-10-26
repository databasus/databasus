package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/storages/internal/storage"
)

func (h *Handler) TestStorage(
	ctx context.Context,
	request TestStorageRequestObject,
) (TestStorageResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	testResult, err := h.connectionTests.TestStorage(ctx, actor, request.StorageId)
	if err != nil {
		return nil, err
	}

	return TestStorage200JSONResponse(toTestResultBody(testResult)), nil
}

func (h *Handler) TestStorageSettings(
	ctx context.Context,
	request TestStorageSettingsRequestObject,
) (TestStorageSettingsResponseObject, error) {
	ctx = httpserver.GetRequestContext(ctx)

	actor, err := getActor(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := toProviderSettings(request.Body.Settings)
	if err != nil {
		return nil, err
	}

	testResult, err := h.connectionTests.TestSettings(ctx, actor, storage.SettingsTest{
		WorkspaceID: request.Body.WorkspaceId,
		StorageID:   request.Body.StorageId,
		Settings:    settings,
	})
	if err != nil {
		return nil, err
	}

	return TestStorageSettings200JSONResponse(toTestResultBody(testResult)), nil
}

func toTestResultBody(testResult storage.TestResult) StorageTestResult {
	return StorageTestResult{IsConnected: testResult.IsConnected, Error: toOptionalString(testResult.Error)}
}
