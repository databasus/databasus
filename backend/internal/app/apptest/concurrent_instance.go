package apptest

import (
	"net/http/httptest"
	"testing"

	"databasus-backend/internal/app"
)

func (a *App) StartConcurrentInstance(t *testing.T) *App {
	t.Helper()
	createdApp, err := app.New(t.Context(), a.environment)
	if err != nil {
		t.Fatalf("build concurrent application: %v", err)
	}
	t.Cleanup(createdApp.Close)

	handler, err := createdApp.NewHandler()
	if err != nil {
		t.Fatalf("build concurrent handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	instance := *a
	instance.URL = server.URL
	instance.app = createdApp
	instance.stopModules = startModules(t, createdApp)
	instance.cancelPhysicalLoops = nil
	instance.physicalRunner = nil
	t.Cleanup(func() { instance.stop(t) })
	return &instance
}
