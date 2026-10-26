//go:build testhelpers

package databases_bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/databases"
)

// CreateTestDatabaseViaAPI creates a logical PostgreSQL database on the shared PostgreSQL 16 test
// server through the contract route, which a router built with GetTestRoutes serves.
func CreateTestDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	token string,
	workspaceID uuid.UUID,
	name string,
) databases.Database {
	t.Helper()

	env := config.GetEnv()

	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	require.NoError(t, err)

	databaseName := "testdb"
	password := "testpassword"

	var settings apiclient.DatabaseSettings

	require.NoError(t, settings.FromPostgresqlLogicalSettings(apiclient.PostgresqlLogicalSettings{
		Type:             apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:             env.TestLocalhost,
		Port:             port,
		Username:         "testuser",
		Password:         &password,
		Database:         &databaseName,
		SslMode:          apiclient.PostgresqlSslModeDisable,
		SshTunnel:        apiclient.SshTunnelSettings{IsEnabled: false},
		IncludeSchemas:   []string{},
		ExcludeTables:    []string{},
		ExcludeTableData: []string{},
		CpuCount:         1,
	}))

	response := workspaces_testing.MakeAPIRequest(router, http.MethodPost, "/api/v1/databases", "Bearer "+token,
		apiclient.DatabaseCreate{
			Name:        name,
			NotifierIds: []uuid.UUID{},
			Settings:    settings,
			WorkspaceId: workspaceID,
		})
	require.Equalf(t, http.StatusCreated, response.Code, "create database: %s", response.Body.String())

	var createdDatabase apiclient.Database
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &createdDatabase))

	database, err := GetAccess().GetDatabaseByID(t.Context(), createdDatabase.Id)
	require.NoError(t, err)

	return database
}

// GetTestDatabaseViaAPI reads a database through the contract route.
func GetTestDatabaseViaAPI(t *testing.T, router *gin.Engine, token string, databaseID uuid.UUID) apiclient.Database {
	t.Helper()

	response := workspaces_testing.MakeAPIRequest(
		router, http.MethodGet, "/api/v1/databases/"+databaseID.String(), "Bearer "+token, nil,
	)
	require.Equalf(t, http.StatusOK, response.Code, "get database: %s", response.Body.String())

	var foundDatabase apiclient.Database
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &foundDatabase))

	return foundDatabase
}

// UpdateTestDatabaseViaAPI reads a database through the contract route, lets change edit the update
// built from it and sends the update. Secrets stay out of the update, so the stored ones are kept.
func UpdateTestDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	token string,
	databaseID uuid.UUID,
	change func(update *apiclient.DatabaseUpdate),
) *httptest.ResponseRecorder {
	t.Helper()

	foundDatabase := GetTestDatabaseViaAPI(t, router, token, databaseID)

	update := apiclient.DatabaseUpdate{
		Name:        foundDatabase.Name,
		NotifierIds: foundDatabase.NotifierIds,
		Settings:    foundDatabase.Settings,
	}
	change(&update)

	return workspaces_testing.MakeAPIRequest(
		router, http.MethodPut, "/api/v1/databases/"+databaseID.String(), "Bearer "+token, update,
	)
}
