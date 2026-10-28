package catalog_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

type catalogFixture struct {
	App          *apptest.App
	Owner        apptest.Session
	Admin        apptest.Session
	DatabaseName string
	WorkspaceID  uuid.UUID
	DatabaseID   uuid.UUID
	StorageID    uuid.UUID
}

func newCatalog(t *testing.T) *catalogFixture {
	t.Helper()
	testApp := apptest.New(t)
	admin := testApp.CreateAdmin(t)
	owner := testApp.CreateNonAdminUser(t, admin)
	workspace := apptest.CreateWorkspace(t, owner, "Physical catalog")
	databaseID := testApp.InsertPhysicalDatabase(t, workspace.Id)
	database, err := owner.Client.GetDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, database.StatusCode())
	return &catalogFixture{
		App: testApp, Owner: owner, Admin: admin, DatabaseName: database.JSON200.Name, WorkspaceID: workspace.Id,
		DatabaseID: databaseID,
		StorageID:  testApp.CreateLocalStorage(t, owner, workspace.Id),
	}
}

type catalogResponse struct{ Body []byte }

type catalogCall struct {
	Method        string
	Path          string
	Authorization string
	Body          any
	StatusCode    int
	Destination   any
}

func (catalog *catalogFixture) call(t *testing.T, call catalogCall) catalogResponse {
	t.Helper()
	method := call.Method
	if method == "" {
		method = http.MethodGet
	}
	status, body := catalog.App.CallOldAPI(t, apptest.Session{Token: strings.TrimPrefix(call.Authorization, "Bearer ")},
		apptest.OldAPICall{Method: method, Path: strings.TrimPrefix(call.Path, "/api/v1"), Body: call.Body})
	require.Equal(t, call.StatusCode, status, "%s", body)
	if call.Destination != nil {
		require.NoError(t, json.Unmarshal(body, call.Destination))
	}
	return catalogResponse{Body: body}
}

func createSecondPhysicalDatabase(t *testing.T, catalog *catalogFixture) struct{ ID uuid.UUID } {
	t.Helper()
	return struct{ ID uuid.UUID }{ID: catalog.App.InsertPhysicalDatabase(t, catalog.WorkspaceID)}
}
