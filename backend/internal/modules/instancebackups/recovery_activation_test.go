package instancebackups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("shows root admin the paused recovery in the protection status", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())

		status := getStatus(t, owner.RootAdmin)

		assert.True(t, status.IsRecoveryPaused)
		require.NotNil(t, status.Recovery)
		assert.Equal(t, owner.App.GetPausedRecoveryID(), status.Recovery.Id)
		assert.Equal(t, apiclient.InstanceRecoveryStatusPAUSED, status.Recovery.Status)
		assert.Equal(t, apiclient.InstanceRecoveryOriginSTARTUP, status.Recovery.Origin)
	})

	t.Run("refuses Databasus backup settings changes while paused", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())
		settings := owner.getSettings(t)
		settings.RetentionDays = 7

		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		requireProblemCode(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeRecoveryPaused,
		)
		assert.Equal(t, 30, owner.getSettings(t).RetentionDays)
	})

	t.Run("refuses activation by an instance admin who is not root admin and stays paused", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())

		response, err := owner.OtherAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
			owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
		assert.True(t, getStatus(t, owner.RootAdmin).IsRecoveryPaused)
	})

	t.Run("refuses activation without the confirmation that the old instance is stopped", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())

		response, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
			owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: false})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.True(t, getStatus(t, owner.RootAdmin).IsRecoveryPaused)
	})

	t.Run("answers 404 for a recovery the journal does not hold", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())

		response, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
			uuid.New(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("activates on root admin's confirmation and keeps a recovered local storage unavailable",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t, apptest.WithRecoveryPause())
			storageID := owner.createLocalStorage(t, "Recovered")
			owner.App.MarkStorageRecovered(t, storageID)

			response, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
				owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})

			require.NoError(t, err)
			require.NotNil(t, response.JSON200, string(response.Body))
			assert.Equal(t, apiclient.InstanceRecoveryStatusACTIVE, response.JSON200.Status)
			assert.False(t, getStatus(t, owner.RootAdmin).IsRecoveryPaused)
			testResponse, err := owner.RootAdmin.Client.TestStorageWithResponse(t.Context(), storageID)
			require.NoError(t, err)
			require.NotNil(t, testResponse.JSON200, string(testResponse.Body))
			storage, err := owner.RootAdmin.Client.GetStorageWithResponse(t.Context(), storageID)
			require.NoError(t, err)
			require.NotNil(t, storage.JSON200)
			assert.Equal(
				t,
				apiclient.StorageRecoveryValidationStatusAVAILABLE,
				storage.JSON200.RecoveryValidationStatus,
			)
			repeated, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
				owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})
			require.NoError(t, err)
			requireProblemCode(t, repeated.StatusCode(), repeated.Body,
				http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
		})

	t.Run("keeps a recovered local storage unavailable right after activation", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryPause())
		storageID := owner.createLocalStorage(t, "Recovered")
		owner.App.MarkStorageRecovered(t, storageID)

		activateRecovery(t, owner)

		storage, err := owner.RootAdmin.Client.GetStorageWithResponse(t.Context(), storageID)
		require.NoError(t, err)
		require.NotNil(t, storage.JSON200)
		assert.Equal(t, apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
			storage.JSON200.RecoveryValidationStatus)
	})
}

func activateRecovery(t *testing.T, owner instanceOwner) {
	t.Helper()

	response, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
		owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
}
