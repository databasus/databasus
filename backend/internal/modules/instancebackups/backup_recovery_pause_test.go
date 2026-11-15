package instancebackups_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The scenario moves its application's clock between passes.
func TestInstanceBackupsDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("makes no backup and sends no incident mail while paused and resumes both after activation",
		func(t *testing.T) {
			owner := newInstanceOwner(t, apptest.WithRecoveryPause())
			storageID := owner.createLocalStorage(t, "Recovered")
			// A recovered snapshot can hold a selected destination whose copy is overdue, a state the
			// API refuses to create while the instance is paused.
			owner.App.ExecuteSQL(t, `INSERT INTO instancebackups_destinations
				(storage_id, selection_generation, selected_at, outstanding_due_at)
				VALUES (?, 1, ?, ?)`, storageID, owner.App.Clock.Now(), owner.App.Clock.Now())
			owner.App.ExecuteSQL(
				t,
				"UPDATE instancebackups_configs SET protection_state = 'PROTECTED', next_selection_generation = 2",
			)
			owner.App.Clock.Advance(time.Hour)

			owner.App.RunJobPass(t, productionJob)
			owner.App.RunJobPass(t, incidentJob)
			backupsWhilePaused := owner.listBackups(t)
			messagesWhilePaused := owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email)
			activation, err := owner.RootAdmin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
				owner.App.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})
			require.NoError(t, err)
			require.NotNil(t, activation.JSON200, string(activation.Body))
			owner.App.RunJobPass(t, incidentJob)
			owner.App.RunJobPass(t, productionJob)

			assert.Empty(t, backupsWhilePaused)
			assert.Empty(t, messagesWhilePaused)
			assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
			assert.Len(t, owner.listBackups(t), 1)
		})
}
