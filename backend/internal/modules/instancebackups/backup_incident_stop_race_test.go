package instancebackups_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestStopInstanceBackupProtectionDuringIncidentPass(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stopped incident open when a running pass assessed protection before the removal",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			storageID := owner.createLocalStorage(t, "Archive")
			settings := owner.selectDestinations(t, storageID)
			owner.backUp(t)
			workspaceBarrier := owner.App.BeginSQLTransaction(t)
			workspaceBarrier.ExecuteSQL(t, "LOCK TABLE workspaces IN ACCESS EXCLUSIVE MODE")

			passResult := make(chan error, 1)
			go func() { passResult <- owner.App.RunJobPassWithContext(t.Context(), incidentJob) }()
			owner.waitForLockWaiter(t, "the incident pass reads the workspaces of its destinations",
				"locktype = 'relation' AND relation = 'workspaces'::regclass")

			settings.StorageIds = []uuid.UUID{}
			removalStatus := make(chan int, 1)
			go func() { removalStatus <- requestSettingsUpdate(t, owner, settings) }()
			owner.waitForRemovalOrLockWaiter(t, removalStatus)
			workspaceBarrier.Rollback(t)

			require.NoError(t, <-passResult)
			assert.Equal(t, http.StatusOK, <-removalStatus)
			status := getStatus(t, owner.RootAdmin)
			assert.Equal(t, apiclient.InstanceBackupHealthSTOPPED, status.Health)
			require.NotNil(t, status.Incident)
			assert.Equal(t, apiclient.InstanceBackupIncidentKindSTOPPED, status.Incident.Kind)
		})
}

// waitForRemovalOrLockWaiter returns once the removal either finished or waits for the incident
// lock, so the pass resumes only after the removal went as far as it can. The status a finished
// removal sends is put back for the test to read.
func (o instanceOwner) waitForRemovalOrLockWaiter(t *testing.T, removalStatus chan int) {
	t.Helper()

	testwait.Until(t, "the removal finishes or waits for the incident lock", func(context.Context) error {
		select {
		case status := <-removalStatus:
			removalStatus <- status

			return nil
		default:
		}

		if o.countLockWaiters(t, "locktype = 'advisory'") == 0 {
			return errors.New("the removal neither finished nor waits for the incident lock")
		}

		return nil
	})
}

func (o instanceOwner) waitForLockWaiter(t *testing.T, description, lockCondition string) {
	t.Helper()

	testwait.Until(t, description, func(context.Context) error {
		if o.countLockWaiters(t, lockCondition) == 0 {
			return errors.New("no session waits for the lock")
		}

		return nil
	})
}

func (o instanceOwner) countLockWaiters(t *testing.T, lockCondition string) int {
	t.Helper()

	var waiterCount int
	o.App.QuerySQL(t, &waiterCount, `SELECT count(*) FROM pg_locks WHERE NOT granted
		AND database = (SELECT oid FROM pg_database WHERE datname = current_database()) AND `+lockCondition)

	return waiterCount
}

func requestSettingsUpdate(t *testing.T, owner instanceOwner, settings apiclient.InstanceBackupSettings) int {
	t.Helper()

	response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	if err != nil {
		return 0
	}

	return response.StatusCode()
}
