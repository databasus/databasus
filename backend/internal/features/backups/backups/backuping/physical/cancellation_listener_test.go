package backuping_physical

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

func newTestCancellationListener() *PhysicalBackupCancellationListener {
	return &PhysicalBackupCancellationListener{
		NewPhysicalBackupCanceller(
			physical_repositories.GetInFlightBackupRepository(),
			backups_bridge.GetCancellationRegistry(),
			logger.GetLogger(),
		),
		physical_repositories.GetWalStreamerRepository(),
		backups_bridge.GetCancellationRegistry(),
		logger.GetLogger(),
	}
}

func seedClaimAndStreamer(t *testing.T, databaseID uuid.UUID) (backupID uuid.UUID) {
	t.Helper()

	backupID = uuid.New()
	ok, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: databaseID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   backupID,
		})
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, physical_repositories.GetWalStreamerRepository().Claim(databaseID))

	return backupID
}

func Test_OnBackupsDisabled_CancelsInFlightAndDeletesStreamer(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	listener.OnBackupsDisabled(t.Context(), prereqs.DB.ID)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim, "disabling backups must cancel + release the in-flight claim")

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer, "disabling backups must delete the streamer row")
}

func Test_OnWalStreamingDisabled_DeletesStreamerKeepsInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	listener.OnWalStreamingDisabled(t.Context(), prereqs.DB.ID)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.NotNil(t, claim, "demoting BackupType must leave in-flight FULL/INCR running")

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer, "demoting BackupType must delete the streamer row")
}

func Test_OnDatabaseDeleting_CancelsTheRunningBackupAndTheStreamerAfterTheCommit(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backupID := seedClaimAndStreamer(t, prereqs.DB.ID)
	backupTaskCtx := registerTestTask(t, backupID)
	streamerTaskCtx := registerTestTask(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	err := storage.GetTransactor().Transaction(t.Context(), func(ctx context.Context) error {
		err := listener.OnDatabaseDeleting(ctx, databases.DatabaseDeleting{DatabaseID: prereqs.DB.ID})
		if err != nil {
			return err
		}

		assert.NoError(t, backupTaskCtx.Err(), "the backup must keep running until the deletion commits")
		assert.NoError(t, streamerTaskCtx.Err(), "the streamer must keep running until the deletion commits")

		return nil
	})
	require.NoError(t, err)

	testwait.Until(t, "the backup and the streamer are cancelled", func(context.Context) error {
		if backupTaskCtx.Err() == nil {
			return errors.New("the backup task is still running")
		}

		if streamerTaskCtx.Err() == nil {
			return errors.New("the streamer task is still running")
		}

		return nil
	})
}

func registerTestTask(t *testing.T, taskID uuid.UUID) context.Context {
	t.Helper()

	taskCtx, cancelTask := context.WithCancel(context.Background())
	backups_bridge.GetCancellationRegistry().Register(taskID, cancelTask)

	t.Cleanup(func() {
		backups_bridge.GetCancellationRegistry().Unregister(taskID)
		cancelTask()
	})

	return taskCtx
}

func Test_OnBackupsStorageChanged_DeletesStreamerKeepsInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	listener.OnBackupsStorageChanged(t.Context(), prereqs.DB.ID)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.NotNil(t, claim, "the in-flight backup is stopped by the removal of the old chain, not here")

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer, "a storage change must stop the streamer so the next one targets the new storage")
}
