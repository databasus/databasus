package backuping_physical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

func newTestCancellationListener() *PhysicalBackupCancellationListener {
	return &PhysicalBackupCancellationListener{
		NewPhysicalBackupCanceller(
			physical_repositories.GetInFlightBackupRepository(),
			tasks_cancellation.GetRequester(),
			logger.GetLogger(),
		),
		physical_repositories.GetWalStreamerRepository(),
		tasks_cancellation.GetRequester(),
		logger.GetLogger(),
	}
}

func seedClaimAndStreamer(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	ok, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: databaseID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   uuid.New(),
		})
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, physical_repositories.GetWalStreamerRepository().Claim(databaseID))
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

func Test_OnBeforeDatabaseRemove_CancelsInFlightAndDeletesStreamer(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedClaimAndStreamer(t, prereqs.DB.ID)
	listener := newTestCancellationListener()

	require.NoError(t, listener.OnBeforeDatabaseRemove(t.Context(), prereqs.DB.ID))

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim)

	streamer, _ := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, streamer)
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
