package databases_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestListDatabases(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("returns the databases in name order a page at a time with a cursor to the next page",
		func(t *testing.T) {
			tm := newTeam(t)
			for _, name := range []string{"c", "a", "b"} {
				tm.createDatabaseNamed(t, name, fromLogicalSettings(t, newLogicalSettings(server)))
			}

			firstPage, err := tm.owner.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
				WorkspaceId: tm.workspace.Id,
				NotifierId:  nil,
				Limit:       new(2),
				After:       nil,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, firstPage.StatusCode(), string(firstPage.Body))
			require.NotNil(t, firstPage.JSON200.NextCursor)

			nextPage, err := tm.owner.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
				WorkspaceId: tm.workspace.Id,
				NotifierId:  nil,
				Limit:       new(2),
				After:       firstPage.JSON200.NextCursor,
			})

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, nextPage.StatusCode(), string(nextPage.Body))
			assert.Equal(t, []string{"a", "b"}, getDatabaseNames(firstPage.JSON200.Items))
			assert.Equal(t, []string{"c"}, getDatabaseNames(nextPage.JSON200.Items))
			assert.Nil(t, nextPage.JSON200.NextCursor)
		})

	t.Run("returns only the databases of the workspace that use the notifier when filtered by it",
		func(t *testing.T) {
			tm := newTeam(t)
			usedNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
			otherNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
			notifiedDatabase := createDatabase(t, tm.owner, apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Notified",
				NotifierIds: []uuid.UUID{usedNotifier.Id},
				Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
			})
			createDatabase(t, tm.owner, apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Notified elsewhere",
				NotifierIds: []uuid.UUID{otherNotifier.Id},
				Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
			})
			tm.createDatabaseNamed(t, "Silent", fromLogicalSettings(t, newLogicalSettings(server)))

			response, err := tm.owner.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
				WorkspaceId: tm.workspace.Id,
				NotifierId:  &usedNotifier.Id,
				Limit:       nil,
				After:       nil,
			})

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			require.Len(t, response.JSON200.Items, 1)
			assert.Equal(t, notifiedDatabase.Id, response.JSON200.Items[0].Id)
			assert.Equal(t, []uuid.UUID{usedNotifier.Id}, response.JSON200.Items[0].NotifierIds)
		})

	t.Run("returns only the databases of the requested workspace", func(t *testing.T) {
		tm := newTeam(t)
		otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other "+uuid.NewString()[:8])
		tm.createDatabaseNamed(t, "Mine", fromLogicalSettings(t, newLogicalSettings(server)))
		createDatabase(t, tm.owner, apiclient.DatabaseCreate{
			WorkspaceId: otherWorkspace.Id,
			Name:        "Elsewhere",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
		})

		assert.Equal(t, []string{"Mine"}, getListedDatabaseNames(t, tm.owner, tm.workspace.Id))
		assert.Equal(t, []string{"Elsewhere"}, getListedDatabaseNames(t, tm.owner, otherWorkspace.Id))
	})

	t.Run("returns 400 invalid_request when the limit is above 100", func(t *testing.T) {
		tm := newTeam(t)

		response, err := tm.owner.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Limit:       new(101),
			After:       nil,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func getDatabaseNames(databases []apiclient.Database) []string {
	names := make([]string, 0, len(databases))
	for _, database := range databases {
		names = append(names, database.Name)
	}

	return names
}
