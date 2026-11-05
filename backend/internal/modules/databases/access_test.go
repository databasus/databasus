package databases_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func getUnreachableServer() containers.Endpoint {
	return containers.Endpoint{Host: "db.example.com", Port: 5432}
}

//nolint:tparallel // The subtests share one source server.
func TestDatabaseAccess(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("lets a viewer list and read the databases of the workspace", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)

		listedDatabases := listDatabases(t, viewer, tm.workspace.Id)

		require.Len(t, listedDatabases, 1)
		assert.Equal(t, databaseID, listedDatabases[0].Id)
		assert.Equal(t, databaseID, getDatabase(t, viewer, databaseID).Id)
	})

	t.Run("lets a member create, change and copy a database of the workspace", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		createdDatabase := createDatabase(t, member, apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Orders",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
		})
		update := toUpdate(createdDatabase)
		update.Name = "Orders renamed"
		updateResponse := updateDatabase(t, member, createdDatabase.Id, update)
		copiedDatabase := copyDatabase(t, member, createdDatabase.Id)

		require.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))
		assert.Equal(t, "Orders renamed (Copy)", copiedDatabase.Name)
		assert.ElementsMatch(t, []string{"Orders renamed", "Orders renamed (Copy)"},
			getListedDatabaseNames(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 403 permission_denied to a viewer who creates a database and creates nothing", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response, err := viewer.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Orders",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, newLogicalSettings(getUnreachableServer())),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 403 permission_denied to a viewer who renames or copies a database and changes nothing",
		func(t *testing.T) {
			tm := newTeam(t)
			viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
			readDatabase := getDatabase(t, viewer, databaseID)
			update := toUpdate(readDatabase)
			update.Name = "Renamed by a viewer"

			updateResponse := updateDatabase(t, viewer, databaseID, update)
			copyResponse, err := viewer.Client.CopyDatabaseWithResponse(t.Context(), databaseID)

			require.NoError(t, err)
			requireProblem(t, updateResponse.StatusCode(), updateResponse.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			requireProblem(t, copyResponse.StatusCode(), copyResponse.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Equal(t, []string{readDatabase.Name}, getListedDatabaseNames(t, tm.owner, tm.workspace.Id))
		})

	t.Run("returns the same 404 not_found to a user outside the workspace for its database and a missing one",
		func(t *testing.T) {
			tm := newTeam(t)
			stranger := tm.signUpNonMember(t)
			databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
			missingDatabaseID := uuid.New()

			existingResponse, err := stranger.Client.GetDatabaseWithResponse(t.Context(), databaseID)
			require.NoError(t, err)
			missingResponse, err := stranger.Client.GetDatabaseWithResponse(t.Context(), missingDatabaseID)
			require.NoError(t, err)

			requireProblem(t, existingResponse.StatusCode(), existingResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t,
				decodeProblemWithoutID(t, missingResponse.Body, missingDatabaseID),
				decodeProblemWithoutID(t, existingResponse.Body, databaseID))
		})

	t.Run("returns 404 not_found to a user outside the workspace who changes or copies its database",
		func(t *testing.T) {
			tm := newTeam(t)
			stranger := tm.signUpNonMember(t)
			databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
			readDatabase := getDatabase(t, tm.owner, databaseID)

			updateResponse := updateDatabase(t, stranger, databaseID, apiclient.DatabaseUpdate{
				Name:        "Taken over",
				NotifierIds: []uuid.UUID{},
				Settings:    fromLogicalSettings(t, newLogicalSettings(getUnreachableServer())),
			})
			copyResponse, err := stranger.Client.CopyDatabaseWithResponse(t.Context(), databaseID)

			require.NoError(t, err)
			requireProblem(t, updateResponse.StatusCode(), updateResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			requireProblem(t, copyResponse.StatusCode(), copyResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, []string{readDatabase.Name}, getListedDatabaseNames(t, tm.owner, tm.workspace.Id))
			assert.Equal(t, getLogicalSettings(t, readDatabase),
				getLogicalSettings(t, getDatabase(t, tm.owner, databaseID)))
		})

	t.Run("returns 404 not_found to a user outside the workspace who lists its databases", func(t *testing.T) {
		tm := newTeam(t)
		stranger := tm.signUpNonMember(t)
		tm.app.InsertDatabase(t, tm.workspace.Id)

		response, err := stranger.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Limit:       nil,
			After:       nil,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 404 not_found to a user outside the workspace who creates a database in it", func(t *testing.T) {
		tm := newTeam(t)
		stranger := tm.signUpNonMember(t)

		response, err := stranger.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Planted",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, newLogicalSettings(getUnreachableServer())),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})
}

// decodeProblemWithoutID blanks the ID out of a problem, so two answers can be compared apart from it.
func decodeProblemWithoutID(t *testing.T, body []byte, databaseID uuid.UUID) apiclient.Problem {
	t.Helper()

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal([]byte(strings.ReplaceAll(string(body), databaseID.String(), "")), &problem))

	return problem
}
