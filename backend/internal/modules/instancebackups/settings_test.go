package instancebackups_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestGetInstanceBackupSettings(t *testing.T) {
	t.Parallel()

	t.Run("returns hourly schedule, 30-day retention and no destinations on a new installation", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		settings := owner.getSettings(t)

		assert.Equal(t, apiclient.ScheduleTypeHOURLY, settings.Schedule.Type)
		assert.Equal(t, 30, settings.RetentionDays)
		assert.Empty(t, settings.StorageIds)
	})

	t.Run("returns 403 when an instance admin who is not root admin reads them", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.OtherAdmin.Client.GetInstanceBackupSettingsWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied,
		)
	})

	t.Run("returns 403 when a member who is not an instance admin reads them", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		rootAdmin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, rootAdmin)

		response, err := member.Client.GetInstanceBackupSettingsWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied,
		)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)

		response, err := app.NewClient(t).GetInstanceBackupSettingsWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated,
		)
	})
}

func TestUpdateInstanceBackupSettings(t *testing.T) {
	t.Parallel()

	t.Run("selects storages of several workspaces and keeps the schedule and retention", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		firstStorageID := owner.createLocalStorage(t, "Production")
		secondStorageID := owner.createLocalStorage(t, "Archive")
		timeOfDay := "03:30"

		settings := owner.getSettings(t)
		settings.Schedule = apiclient.Schedule{Type: apiclient.ScheduleTypeDAILY, TimeOfDay: &timeOfDay}
		settings.RetentionDays = 14
		settings.StorageIds = []uuid.UUID{firstStorageID, secondStorageID}
		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		persistedSettings := owner.getSettings(t)
		assert.Equal(t, apiclient.ScheduleTypeDAILY, persistedSettings.Schedule.Type)
		assert.Equal(t, &timeOfDay, persistedSettings.Schedule.TimeOfDay)
		assert.Equal(t, 14, persistedSettings.RetentionDays)
		assert.ElementsMatch(t, []uuid.UUID{firstStorageID, secondStorageID}, persistedSettings.StorageIds)
	})

	t.Run("refuses an unknown storage without changing the selection", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		selectedStorageID := owner.createLocalStorage(t, "Production")
		owner.selectDestinations(t, selectedStorageID)

		settings := owner.getSettings(t)
		settings.StorageIds = []uuid.UUID{selectedStorageID, uuid.New()}
		settings.RetentionDays = 7
		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest,
		)
		persistedSettings := owner.getSettings(t)
		assert.Equal(t, []uuid.UUID{selectedStorageID}, persistedSettings.StorageIds)
		assert.Equal(t, 30, persistedSettings.RetentionDays)
	})

	t.Run("refuses a storage selected twice", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Production")

		settings := owner.getSettings(t)
		settings.StorageIds = []uuid.UUID{storageID, storageID}
		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest,
		)
		assert.Empty(t, owner.getSettings(t).StorageIds)
	})

	t.Run("refuses a retention outside its bounds", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		settings := owner.getSettings(t)
		settings.RetentionDays = 0
		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest,
		)
	})

	t.Run("returns 403 without changes when an instance admin who is not root admin saves", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Production")

		settings := owner.getSettings(t)
		settings.StorageIds = []uuid.UUID{storageID}
		response, err := owner.OtherAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied,
		)
		assert.Empty(t, owner.getSettings(t).StorageIds)
	})
}

func TestListInstanceBackupDestinations(t *testing.T) {
	t.Parallel()

	t.Run(
		"pages storages of every workspace by workspace and storage name and marks selected ones",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			archiveStorageID := owner.createLocalStorage(t, "Archive")
			productionStorageID := owner.createLocalStorage(t, "Production")
			owner.selectDestinations(t, productionStorageID)

			limit := 1
			firstPage, err := owner.RootAdmin.Client.ListInstanceBackupDestinationsWithResponse(
				t.Context(), &apiclient.ListInstanceBackupDestinationsParams{Limit: &limit},
			)
			require.NoError(t, err)
			require.NotNil(t, firstPage.JSON200, string(firstPage.Body))
			require.NotNil(t, firstPage.JSON200.NextCursor)
			secondPage, err := owner.RootAdmin.Client.ListInstanceBackupDestinationsWithResponse(
				t.Context(),
				&apiclient.ListInstanceBackupDestinationsParams{Limit: &limit, After: firstPage.JSON200.NextCursor},
			)
			require.NoError(t, err)
			require.NotNil(t, secondPage.JSON200, string(secondPage.Body))

			listedChoices := slices.Concat(firstPage.JSON200.Items, secondPage.JSON200.Items)
			require.Len(t, listedChoices, 2)
			assert.Equal(t, archiveStorageID, listedChoices[0].StorageId)
			assert.Equal(t, "Archive", listedChoices[0].WorkspaceName)
			assert.False(t, listedChoices[0].IsSelected)
			assert.Equal(t, productionStorageID, listedChoices[1].StorageId)
			assert.True(t, listedChoices[1].IsSelected)
			assert.Nil(t, secondPage.JSON200.NextCursor)
		},
	)

	t.Run("keeps only choices whose storage or workspace name matches the query", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.createLocalStorage(t, "Archive")
		productionStorageID := owner.createLocalStorage(t, "Production")

		query := "prod"
		response, err := owner.RootAdmin.Client.ListInstanceBackupDestinationsWithResponse(
			t.Context(), &apiclient.ListInstanceBackupDestinationsParams{Query: &query},
		)

		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		require.Len(t, response.JSON200.Items, 1)
		assert.Equal(t, productionStorageID, response.JSON200.Items[0].StorageId)
	})

	t.Run("returns 403 when an instance admin who is not root admin lists them", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.OtherAdmin.Client.ListInstanceBackupDestinationsWithResponse(
			t.Context(), &apiclient.ListInstanceBackupDestinationsParams{},
		)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied,
		)
	})

	t.Run("returns 400 with a malformed cursor", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		cursor := "not-a-cursor"
		response, err := owner.RootAdmin.Client.ListInstanceBackupDestinationsWithResponse(
			t.Context(), &apiclient.ListInstanceBackupDestinationsParams{After: &cursor},
		)

		require.NoError(t, err)
		requireProblemCode(
			t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest,
		)
	})
}
