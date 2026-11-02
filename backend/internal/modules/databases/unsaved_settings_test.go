package databases_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestTestDatabaseSettings(t *testing.T) {
	t.Parallel()

	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("reports the outcome when a member tests a database that is not saved", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		testResult := testUnsavedSettings(t, member, newSettingsTarget(tm.workspace.Id, nil,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer))))

		assert.True(t, testResult.IsConnected, "%v", testResult.Error)
		assert.Empty(t, tm.listDatabaseIDs(t), "a test saves nothing")
	})

	t.Run("connects to the other database with the stored password when the request leaves the password out",
		func(t *testing.T) {
			tm := newTeam(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			otherDatabase := newUniqueName("other_")
			_, err := connectAsAdmin(t, postgresServer, containers.PostgresDatabase).
				Exec(t.Context(), "CREATE DATABASE "+otherDatabase)
			require.NoError(t, err)

			changedTarget := newPostgresqlContainerTarget(postgresServer)
			changedTarget.database = otherDatabase
			changedTarget.password = nil

			testResult := testUnsavedSettings(t, tm.owner, newSettingsTarget(tm.workspace.Id, &savedDatabase.Id,
				newPostgresqlLogicalSettings(t, changedTarget)))

			assert.True(t, testResult.IsConnected, "%v", testResult.Error)
		})

	t.Run("returns 404 not_found without connecting when the named database belongs to another workspace",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other "+uuid.NewString()[:8])
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response := testUnsavedSettingsResponse(t, tm.owner, newSettingsTarget(otherWorkspace.Id,
				&savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 404 not_found without connecting when the named database belongs to another team",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			outsiderWorkspace := apptest.CreateWorkspace(t, outsider, "Outsider "+uuid.NewString()[:8])
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response := testUnsavedSettingsResponse(t, outsider, newSettingsTarget(outsiderWorkspace.Id,
				&savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("answers an unknown database with the body of another team's database apart from the ID",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			outsiderWorkspace := apptest.CreateWorkspace(t, outsider, "Outsider "+uuid.NewString()[:8])
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			unknownDatabaseID := uuid.New()
			settings := newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer))

			otherTeamResponse := testUnsavedSettingsResponse(t, outsider,
				newSettingsTarget(outsiderWorkspace.Id, &savedDatabase.Id, settings))
			unknownResponse := testUnsavedSettingsResponse(t, outsider,
				newSettingsTarget(outsiderWorkspace.Id, &unknownDatabaseID, settings))

			require.Equal(t, http.StatusNotFound, otherTeamResponse.StatusCode())
			require.Equal(t, http.StatusNotFound, unknownResponse.StatusCode())
			assert.Equal(
				t,
				strings.ReplaceAll(
					string(otherTeamResponse.Body),
					savedDatabase.Id.String(),
					unknownDatabaseID.String(),
				),
				string(unknownResponse.Body),
			)
		})

	t.Run("returns 400 invalid_request without connecting when the settings are of another type than the database",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			mysqlDatabaseID := tm.insertMysqlDatabase(t)

			response := testUnsavedSettingsResponse(t, tm.owner, newSettingsTarget(tm.workspace.Id,
				&mysqlDatabaseID, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 400 invalid_request without connecting when the request names no workspace", func(t *testing.T) {
		tm := newTeam(t)
		recorder := startConnectionRecorder(t)
		requestBody, err := json.Marshal(map[string]any{
			"settings": newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget()),
		})
		require.NoError(t, err)

		response, err := tm.owner.Client.TestDatabaseSettingsWithBodyWithResponse(t.Context(),
			"application/json", bytes.NewReader(requestBody))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("returns 403 permission_denied without connecting when a viewer tests unsaved settings", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		recorder := startConnectionRecorder(t)

		response := testUnsavedSettingsResponse(t, viewer, newSettingsTarget(tm.workspace.Id, nil,
			newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("returns 403 permission_denied without connecting when a viewer points a saved database at another host",
		func(t *testing.T) {
			tm := newTeam(t)
			viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response := testUnsavedSettingsResponse(t, viewer, newSettingsTarget(tm.workspace.Id,
				&savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 404 not_found without connecting when an outsider tests settings for another team's workspace",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			recorder := startConnectionRecorder(t)

			response := testUnsavedSettingsResponse(t, outsider, newSettingsTarget(tm.workspace.Id, nil,
				newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 500 internal without connecting or disclosing the error when the saved database lookup fails",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			tm.app.ExecuteSQL(t, "ALTER TABLE databases RENAME TO databases_broken_for_test")

			response := testUnsavedSettingsResponse(t, tm.owner, newSettingsTarget(tm.workspace.Id,
				&savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			requireInternalProblemWithoutDetails(t, response.StatusCode(), response.Body, "databases")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 500 internal without connecting or disclosing the error when the permission lookup fails",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			tm.app.ExecuteSQL(t, "ALTER TABLE workspaces_members RENAME TO workspaces_members_broken_for_test")
			settings := newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())

			unsavedResponse := testUnsavedSettingsResponse(t, tm.owner,
				newSettingsTarget(tm.workspace.Id, nil, settings))
			savedResponse := testUnsavedSettingsResponse(t, tm.owner,
				newSettingsTarget(tm.workspace.Id, &savedDatabase.Id, settings))

			requireInternalProblemWithoutDetails(t, unsavedResponse.StatusCode(), unsavedResponse.Body,
				"workspaces_members")
			requireInternalProblemWithoutDetails(t, savedResponse.StatusCode(), savedResponse.Body,
				"workspaces_members")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		tm := newTeam(t)

		response, err := tm.app.NewClient(t).TestDatabaseSettingsWithResponse(t.Context(),
			newSettingsTarget(tm.workspace.Id, nil,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer))))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

func testUnsavedSettings(
	t *testing.T,
	session apptest.Session,
	target apiclient.DatabaseSettingsTarget,
) apiclient.DatabaseTestResult {
	t.Helper()

	response := testUnsavedSettingsResponse(t, session, target)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func testUnsavedSettingsResponse(
	t *testing.T,
	session apptest.Session,
	target apiclient.DatabaseSettingsTarget,
) *apiclient.TestDatabaseSettingsResponse {
	t.Helper()

	response, err := session.Client.TestDatabaseSettingsWithResponse(t.Context(), target)
	require.NoError(t, err)

	return response
}

// requireInternalProblemWithoutDetails checks that a failure inside the server reaches the client as a
// bare internal error, without the failed query or the table it named.
func requireInternalProblemWithoutDetails(t *testing.T, status int, body []byte, brokenTable string) {
	t.Helper()

	requireProblem(t, status, body, http.StatusInternalServerError, apiclient.ProblemCodeInternal)
	assert.NotContains(t, string(body), brokenTable)
	assert.NotContains(t, string(body), "SQLSTATE")
}
