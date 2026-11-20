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
func TestInstanceBackupsAfterRecovery(t *testing.T) {
	t.Parallel()

	t.Run("stores a backup in a recovered local storage and sends incident mail from the first start after a recovery",
		func(t *testing.T) {
			owner := newInstanceOwner(t, apptest.WithPublishedRecovery())
			storageID := owner.createLocalStorage(t, "Recovered")
			owner.App.NormalizeRecoveredStorages(t)
			// A recovered snapshot can hold a selected destination whose copy is overdue, a state the
			// API cannot create.
			owner.App.ExecuteSQL(t, `INSERT INTO instancebackups_destinations
				(storage_id, selection_generation, selected_at, outstanding_due_at)
				VALUES (?, 1, ?, ?)`, storageID, owner.App.Clock.Now(), owner.App.Clock.Now())
			owner.App.ExecuteSQL(
				t,
				"UPDATE instancebackups_configs SET protection_state = 'PROTECTED', next_selection_generation = 2",
			)
			owner.App.Clock.Advance(time.Hour)

			owner.App.RunJobPass(t, incidentJob)
			owner.App.RunJobPass(t, productionJob)
			owner.App.RunJobPass(t, deliveryJob)

			assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
			backups := owner.listBackups(t)
			require.Len(t, backups, 1)
			assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, backups[0].Status)
			assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, getCopy(t, backups[0], storageID).Status)
		})
}
