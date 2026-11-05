package storages_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestListStoragesPages(t *testing.T) {
	t.Parallel()

	t.Run("pages the storages by name and points to the next page while more remain", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		for _, name := range []string{"b", "a", "c"} {
			tm.createLocalStorage(t, name)
		}

		firstPage, err := tm.owner.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       new(2),
			After:       nil,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, firstPage.StatusCode(), string(firstPage.Body))
		require.NotNil(t, firstPage.JSON200.NextCursor)

		secondPage, err := tm.owner.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       new(2),
			After:       firstPage.JSON200.NextCursor,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, secondPage.StatusCode(), string(secondPage.Body))

		assert.Equal(t, []string{"a", "b"}, getStorageNames(firstPage.JSON200.Items))
		assert.Equal(t, []string{"c"}, getStorageNames(secondPage.JSON200.Items))
		assert.Nil(t, secondPage.JSON200.NextCursor)
	})

	t.Run("returns 400 invalid_request when the limit is above 100", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       new(101),
			After:       nil,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func getStorageNames(storages []apiclient.Storage) []string {
	names := make([]string, 0, len(storages))
	for _, listedStorage := range storages {
		names = append(names, listedStorage.Name)
	}

	return names
}
