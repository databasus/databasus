package databases_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func createDatabase(t *testing.T, session apptest.Session, creation apiclient.DatabaseCreate) apiclient.Database {
	t.Helper()

	response, err := session.Client.CreateDatabaseWithResponse(t.Context(), creation)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

func getDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) apiclient.Database {
	t.Helper()

	response, err := session.Client.GetDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func listDatabases(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []apiclient.Database {
	t.Helper()

	response, err := session.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
		WorkspaceId: workspaceID,
		NotifierId:  nil,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.Items
}

func getListedDatabaseNames(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []string {
	t.Helper()

	listedDatabases := listDatabases(t, session, workspaceID)

	names := make([]string, 0, len(listedDatabases))
	for _, listedDatabase := range listedDatabases {
		names = append(names, listedDatabase.Name)
	}

	return names
}

func updateDatabase(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	update apiclient.DatabaseUpdate,
) *apiclient.UpdateDatabaseResponse {
	t.Helper()

	response, err := session.Client.UpdateDatabaseWithResponse(t.Context(), databaseID, update)
	require.NoError(t, err)

	return response
}

func copyDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) apiclient.Database {
	t.Helper()

	response, err := session.Client.CopyDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

// toUpdate sends a read database back unchanged, the way the edit form starts.
func toUpdate(database apiclient.Database) apiclient.DatabaseUpdate {
	return apiclient.DatabaseUpdate{
		Name:        database.Name,
		NotifierIds: database.NotifierIds,
		Settings:    database.Settings,
	}
}

func newSettingsTarget(
	workspaceID uuid.UUID,
	databaseID *uuid.UUID,
	settings apiclient.DatabaseSettings,
) apiclient.DatabaseSettingsTarget {
	return apiclient.DatabaseSettingsTarget{
		WorkspaceId: workspaceID,
		DatabaseId:  databaseID,
		Settings:    settings,
	}
}
