package storages_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestStorageAccess(t *testing.T) {
	t.Parallel()

	t.Run("lets a viewer list and read the workspace's storages", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdStorage := tm.createLocalStorage(t, "Local")

		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, viewer, tm.workspace.Id))
		assert.Equal(t, "Local", getStorage(t, viewer, createdStorage.Id).Name)
	})

	t.Run("lets a workspace member create and change a storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		createdStorage := createStorage(t, member, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Local",
			Settings:    newLocalSettings(t),
		})
		response := updateStorage(t, member, createdStorage.Id, apiclient.StorageUpdate{
			Name:     "Renamed",
			Settings: newLocalSettings(t),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, "Renamed", getStorage(t, tm.owner, createdStorage.Id).Name)
	})

	t.Run("returns 403 permission_denied and keeps the name when a viewer renames a storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := updateStorage(t, viewer, createdStorage.Id, apiclient.StorageUpdate{
			Name:     "Renamed",
			Settings: newLocalSettings(t),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, "Local", getStorage(t, tm.owner, createdStorage.Id).Name)
	})

	t.Run("returns 403 permission_denied and creates nothing when a viewer creates a storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response, err := viewer.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Local",
			Settings:    newLocalSettings(t),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("answers another team's storage exactly as a storage that does not exist", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		missingStorageID := uuid.New()

		existingResponse, err := stranger.Client.GetStorageWithResponse(t.Context(), createdStorage.Id)
		require.NoError(t, err)
		missingResponse, err := stranger.Client.GetStorageWithResponse(t.Context(), missingStorageID)
		require.NoError(t, err)

		requireProblem(t, existingResponse.StatusCode(), existingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.Equal(t,
			normalizeProblem(t, missingResponse.Body, missingStorageID),
			normalizeProblem(t, existingResponse.Body, createdStorage.Id))
	})

	t.Run("returns 404 not_found and keeps the name when a stranger changes another team's storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			stranger := tm.stranger(t)
			createdStorage := tm.createLocalStorage(t, "Local")

			response := updateStorage(t, stranger, createdStorage.Id, apiclient.StorageUpdate{
				Name:     "Taken over",
				Settings: newLocalSettings(t),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, "Local", getStorage(t, tm.owner, createdStorage.Id).Name)
		})

	t.Run("returns 404 not_found when a stranger lists another team's storages", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		tm.createLocalStorage(t, "Local")

		response, err := stranger.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       nil,
			After:       nil,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

// normalizeProblem drops the ID from a problem body, so two answers can be compared apart from it.
func normalizeProblem(t *testing.T, body []byte, storageID uuid.UUID) map[string]any {
	t.Helper()

	var problem map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.ReplaceAll(string(body), storageID.String(), "")), &problem))

	return problem
}
