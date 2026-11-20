package instancebackups_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const backupFilePrefix = "databasus-backups-"

//nolint:tparallel // Each scenario moves its own application's clock between passes.
func TestDeliverInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("stores the encrypted file in a remote storage under a name of this instance", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		owner.selectDestinations(t, namespace.StorageID)

		backup := owner.backUp(t)
		storedFileName := owner.getStoredFileName(t, backup, namespace.StorageID)

		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, backup.Status)
		assert.True(t, strings.HasPrefix(storedFileName, backupFilePrefix), storedFileName)
		assert.Contains(t, storedFileName, backup.Id.String())
		namespace.RequireFile(t, storedFileName)
		assert.Empty(t, owner.getSpoolFileNames(t))
	})

	t.Run("keeps the stored copy and retries the failed one with the storage's changed settings", func(t *testing.T) {
		owner := newInstanceOwner(t)
		localStorageID := owner.createLocalStorage(t, "Local")
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.selectDestinations(t, localStorageID, namespace.StorageID)

		partialBackup := owner.backUp(t)
		namespace.MakeAvailable(t, owner.RootAdmin)
		owner.App.RunJobPass(t, deliveryJob)
		backupBeforeRetry := owner.getBackup(t, partialBackup.Id)
		owner.App.Clock.Advance(time.Minute)
		owner.App.RunJobPass(t, deliveryJob)
		completedBackup := owner.getBackup(t, partialBackup.Id)

		assert.Equal(t, apiclient.InstanceBackupStatusValueINPROGRESS, partialBackup.Status)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, getCopy(t, partialBackup, localStorageID).Status)
		failedCopy := getCopy(t, partialBackup, namespace.StorageID)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusRETRYING, failedCopy.Status)
		require.NotNil(t, failedCopy.LastError)
		assert.NotEmpty(t, *failedCopy.LastError)
		assert.False(t, failedCopy.IsDownloadable)
		assert.Equal(t, 1, getCopy(t, backupBeforeRetry, namespace.StorageID).AttemptCount)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, completedBackup.Status)
		retriedCopy := getCopy(t, completedBackup, namespace.StorageID)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, retriedCopy.Status)
		assert.Equal(t, 2, retriedCopy.AttemptCount)
		assert.Nil(t, retriedCopy.LastError)
	})

	t.Run("fails the copy without an upload when its destination was removed before delivery", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		owner.App.RunJobPass(t, productionJob)
		owner.selectDestinations(t)

		owner.App.RunJobPass(t, deliveryJob)
		backup := owner.getLatestBackup(t)

		assert.Equal(t, apiclient.InstanceBackupStatusValueFAILED, backup.Status)
		require.Len(t, backup.Copies, 1)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusFAILED, backup.Copies[0].Status)
		assert.Equal(t, 0, backup.Copies[0].AttemptCount)
		assert.Empty(t, owner.getSpoolFileNames(t))
	})

	t.Run("rejects a late upload after its destination was removed and added again", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		owner.selectDestinations(t, namespace.StorageID)
		owner.App.RunJobPass(t, productionJob)
		heldUpload := namespace.HoldNextUpload(t, owner.RootAdmin, backupFilePrefix)

		passResult := owner.runDeliveryInBackground(t.Context(), t)
		<-heldUpload.Started
		owner.selectDestinations(t)
		owner.selectDestinations(t, namespace.StorageID)
		heldUpload.Release()
		require.NoError(t, <-passResult)
		backup := owner.getLatestBackup(t)
		status := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupStatusValueFAILED, backup.Status)
		rejectedCopy := getCopy(t, backup, namespace.StorageID)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusFAILED, rejectedCopy.Status)
		assert.False(t, rejectedCopy.IsDownloadable)
		assert.NotEqual(t, apiclient.InstanceBackupHealthHEALTHY, status.Health)
	})

	t.Run("fails the waiting copies once the local file is older than 24 hours", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.selectDestinations(t, namespace.StorageID)
		retryingBackup := owner.backUp(t)

		owner.App.Clock.Advance(24 * time.Hour)
		owner.App.RunJobPass(t, deliveryJob)
		expiredBackup := owner.getBackup(t, retryingBackup.Id)

		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusRETRYING,
			getCopy(t, retryingBackup, namespace.StorageID).Status,
		)
		assert.Equal(t, apiclient.InstanceBackupStatusValueFAILED, expiredBackup.Status)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusFAILED, getCopy(t, expiredBackup, namespace.StorageID).Status)
		assert.Empty(t, owner.getSpoolFileNames(t))
	})
}

//nolint:tparallel // Each scenario restarts its own application's modules between steps.
func TestRestartDuringInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("interrupts a running dump and accepts a new start", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		runningBackup := owner.startBackup(t)

		owner.App.RestartModules(t)
		interruptedBackup := owner.getBackup(t, runningBackup.Id)
		restartedBackup := owner.startBackup(t)

		assert.Equal(t, apiclient.InstanceBackupStatusValueINTERRUPTED, interruptedBackup.Status)
		assert.NotNil(t, interruptedBackup.CompletedAt)
		assert.Equal(t, apiclient.InstanceBackupStatusValueINPROGRESS, restartedBackup.Status)
	})

	t.Run("returns an upload cut by a shutdown to the queue and stores it after the restart", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		owner.selectDestinations(t, namespace.StorageID)
		owner.App.RunJobPass(t, productionJob)
		heldUpload := namespace.HoldNextUpload(t, owner.RootAdmin, backupFilePrefix)
		passCtx, stopPass := context.WithCancel(t.Context())

		passResult := owner.runDeliveryInBackground(passCtx, t)
		<-heldUpload.Started
		stopPass()
		require.NoError(t, <-passResult)
		heldUpload.Release()
		uploadingBackup := owner.getLatestBackup(t)
		namespace.MakeAvailable(t, owner.RootAdmin)
		owner.App.RestartModules(t)
		queuedBackup := owner.getLatestBackup(t)
		owner.App.RunJobPass(t, deliveryJob)
		storedBackup := owner.getLatestBackup(t)

		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusUPLOADING,
			getCopy(t, uploadingBackup, namespace.StorageID).Status,
		)
		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusRETRYING,
			getCopy(t, queuedBackup, namespace.StorageID).Status,
		)
		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, storedBackup.Status)
		storedCopy := getCopy(t, storedBackup, namespace.StorageID)
		assert.Equal(t, 2, storedCopy.AttemptCount)
		namespace.RequireFile(t, owner.getStoredFileName(t, storedBackup, namespace.StorageID))
	})

	t.Run("interrupts the copies and asks for a new dump when the local file was lost", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.selectDestinations(t, namespace.StorageID)
		lostBackup := owner.backUp(t)
		removeSpoolFiles(t, owner)

		owner.App.RestartModules(t)
		interruptedBackup := owner.getBackup(t, lostBackup.Id)
		owner.App.Clock.Advance(time.Minute)
		owner.App.RunJobPass(t, productionJob)

		assert.Equal(t, apiclient.InstanceBackupStatusValueINTERRUPTED, interruptedBackup.Status)
		assert.Equal(
			t,
			apiclient.InstanceBackupCopyStatusINTERRUPTED,
			getCopy(t, interruptedBackup, namespace.StorageID).Status,
		)
		backups := owner.listBackups(t)
		require.Len(t, backups, 2)
		assert.NotEqual(t, lostBackup.Id, backups[0].Id)
	})

	t.Run("removes local files no backup needs", func(t *testing.T) {
		owner := newInstanceOwner(t)
		spoolFolder := owner.App.GetServerConfig().InstanceBackupSpoolFolder
		require.NoError(t, os.MkdirAll(spoolFolder, 0o700))
		require.NoError(
			t,
			os.WriteFile(filepath.Join(spoolFolder, "left-by-a-crash.dbbackup"), []byte("partial"), 0o600),
		)

		owner.App.RestartModules(t)

		assert.Empty(t, owner.getSpoolFileNames(t))
	})
}

func removeSpoolFiles(t *testing.T, owner instanceOwner) {
	t.Helper()

	spoolFolder := owner.App.GetServerConfig().InstanceBackupSpoolFolder
	for _, fileName := range owner.getSpoolFileNames(t) {
		require.NoError(t, os.Remove(filepath.Join(spoolFolder, fileName)))
	}
}
