package instancebackups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/diskusage"
)

func TestStartInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("returns 202 at once and stores the backup in every destination after the jobs run", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		firstStorageID := owner.createLocalStorage(t, "First")
		secondStorageID := owner.createLocalStorage(t, "Second")
		owner.selectDestinations(t, firstStorageID, secondStorageID)

		startedBackup := owner.startBackup(t)
		owner.App.RunJobPass(t, productionJob)
		owner.App.RunJobPass(t, deliveryJob)

		assert.Equal(t, apiclient.InstanceBackupStatusValueINPROGRESS, startedBackup.Status)
		assert.Equal(t, apiclient.InstanceBackupTriggerMANUAL, startedBackup.Trigger)
		assert.Empty(t, startedBackup.Copies)
		storedBackup := owner.getBackup(t, startedBackup.Id)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, storedBackup.Status)
		require.NotNil(t, storedBackup.SnapshotAt)
		require.NotNil(t, storedBackup.SizeBytes)
		assert.Positive(t, *storedBackup.SizeBytes)
		require.Len(t, storedBackup.Copies, 2)
		for _, storedCopy := range storedBackup.Copies {
			assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, storedCopy.Status)
			assert.True(t, storedCopy.IsDownloadable)
			assert.Equal(t, 1, storedCopy.AttemptCount)
		}
	})

	t.Run("removes the local file and dumps again after planning the copies fails", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		owner.App.ExecuteSQL(t, `CREATE FUNCTION reject_instance_backup_copy() RETURNS trigger
			LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'copy rejected by the test'; END $$`)
		owner.App.ExecuteSQL(t, `CREATE TRIGGER reject_instance_backup_copy
			BEFORE INSERT ON instancebackups_copies
			FOR EACH ROW EXECUTE FUNCTION reject_instance_backup_copy()`)

		failedPassErr := owner.App.RunJobPassWithContext(t.Context(), productionJob)
		spoolFileNamesAfterFailure := owner.getSpoolFileNames(t)
		runningBackup := owner.getLatestBackup(t)
		owner.App.ExecuteSQL(t, `DROP TRIGGER reject_instance_backup_copy ON instancebackups_copies`)
		completedBackup := owner.backUp(t)

		require.ErrorContains(t, failedPassErr, "copy rejected by the test")
		assert.Empty(t, spoolFileNamesAfterFailure)
		assert.Equal(t, apiclient.InstanceBackupStatusValueINPROGRESS, runningBackup.Status)
		assert.Empty(t, runningBackup.Copies)
		assert.Equal(t, runningBackup.Id, completedBackup.Id)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, completedBackup.Status)
		assert.Empty(t, owner.getSpoolFileNames(t))
	})

	t.Run("fails the run without copies or a local file when the staging disk is full", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithDiskUsage(func(string) (diskusage.Usage, error) {
			return diskusage.Usage{TotalBytes: 1 << 30, UsedBytes: 1 << 30, FreeBytes: 0}, nil
		}))
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))

		startedBackup := owner.startBackup(t)
		owner.App.RunJobPass(t, productionJob)
		owner.App.RunJobPass(t, deliveryJob)

		failedBackup := owner.getBackup(t, startedBackup.Id)
		assert.Equal(t, apiclient.InstanceBackupStatusValueFAILED, failedBackup.Status)
		require.NotNil(t, failedBackup.FailureReason)
		assert.NotEmpty(t, *failedBackup.FailureReason)
		assert.Empty(t, failedBackup.Copies)
		var dumpStatus string
		owner.App.QuerySQL(t, &dumpStatus,
			`SELECT dump_status FROM instancebackups_backups WHERE id = ?`, startedBackup.Id)
		assert.Equal(t, "FAILED", dumpStatus)
		var plannedCopyCount int64
		owner.App.QuerySQL(t, &plannedCopyCount, `SELECT count(*) FROM instancebackups_copies`)
		assert.Zero(t, plannedCopyCount)
		assert.Empty(t, owner.getSpoolFileNames(t))
	})

	t.Run("returns 409 already_running while another dump runs", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		owner.startBackup(t)

		response, err := owner.RootAdmin.Client.StartInstanceBackupWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
		assert.Len(t, owner.listBackups(t), 1)
	})

	t.Run("returns 400 invalid_request without destinations", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.RootAdmin.Client.StartInstanceBackupWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Empty(t, owner.listBackups(t))
	})

	t.Run("returns 403 to an instance admin who is not root admin", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))

		startResponse, err := owner.OtherAdmin.Client.StartInstanceBackupWithResponse(t.Context())
		require.NoError(t, err)
		listResponse, err := owner.OtherAdmin.Client.ListInstanceBackupsWithResponse(t.Context(), nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusForbidden, startResponse.StatusCode(), string(startResponse.Body))
		assert.Equal(t, http.StatusForbidden, listResponse.StatusCode(), string(listResponse.Body))
		assert.Empty(t, owner.listBackups(t))
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.App.NewClient(t).StartInstanceBackupWithResponse(t.Context())

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode(), string(response.Body))
	})
}

//nolint:tparallel // Each scenario moves its own application's clock between passes.
func TestScheduleInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("backs up at once after the first destination is added", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))

		backup := owner.backUp(t)

		assert.Equal(t, apiclient.InstanceBackupTriggerDESTINATIONADDED, backup.Trigger)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, backup.Status)
	})

	t.Run("backs up again only when the hourly schedule is due", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		firstBackup := owner.backUp(t)

		owner.App.Clock.Advance(59 * time.Minute)
		owner.App.RunJobPass(t, productionJob)
		backupsBeforeDue := owner.listBackups(t)
		owner.App.Clock.Advance(time.Minute)
		scheduledBackup := owner.backUp(t)

		assert.Len(t, backupsBeforeDue, 1)
		assert.NotEqual(t, firstBackup.Id, scheduledBackup.Id)
		assert.Equal(t, apiclient.InstanceBackupTriggerSCHEDULE, scheduledBackup.Trigger)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, scheduledBackup.Status)
	})

	t.Run("does not back up without destinations even when the schedule is due", func(t *testing.T) {
		owner := newInstanceOwner(t)

		owner.App.Clock.Advance(2 * time.Hour)
		owner.App.RunJobPass(t, productionJob)

		assert.Empty(t, owner.listBackups(t))
	})

	t.Run("backs up at once again when a destination is added after all were removed", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		owner.backUp(t)
		owner.selectDestinations(t)

		owner.selectDestinations(t, storageID)
		resumedBackup := owner.backUp(t)

		assert.Len(t, owner.listBackups(t), 2)
		assert.Equal(t, apiclient.InstanceBackupTriggerDESTINATIONADDED, resumedBackup.Trigger)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, resumedBackup.Status)
	})
}
