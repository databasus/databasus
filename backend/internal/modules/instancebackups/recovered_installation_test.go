package instancebackups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRecoveredInstallation(t *testing.T) {
	t.Parallel()

	t.Run("shows root admin the completed recovery in the protection status", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithPublishedRecovery())

		status := getStatus(t, owner.RootAdmin)

		require.NotNil(t, status.Recovery)
		assert.Equal(t, apiclient.InstanceRecoveryStatusCOMPLETED, status.Recovery.Status)
		assert.Equal(t, apiclient.InstanceRecoveryOriginSTARTUP, status.Recovery.Origin)
	})

	t.Run("accepts Databasus backup settings changes right after a recovery", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithPublishedRecovery())
		settings := owner.getSettings(t)
		settings.RetentionDays = 7

		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, 7, owner.getSettings(t).RetentionDays)
	})
}
