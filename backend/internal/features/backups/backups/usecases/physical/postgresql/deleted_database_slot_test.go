package usecases_physical_postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/logger"
)

// A streamer whose cancellation has not arrived yet can still reach its slot creation after the
// deletion dropped the slot; it must not leave a slot of the deleted database behind.
func Test_WalStreamSupervisorRun_WhenTheDatabaseIsDeleted_LeavesNoSlot(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	walSlotName := fixture.SourceDB.Engine.ReplicationSlotName
	adminConn := OpenAdminConn(t, fixture)
	t.Cleanup(func() {
		DropReplicationSlotExternally(t, adminConn, walSlotName)
	})

	databases_bridge.RemoveTestDatabase(t.Context(), fixture.DB)

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:     fixture.DB.ID,
		SourceDB:       fixture.SourceDB,
		StorageID:      fixture.Storage.ID,
		FileStore:      newMockWalStoreFor(newMockWalStorage()),
		BackupStatus:   databases_bridge.GetModule(),
		Databases:      databases_bridge.GetModule(),
		Encryption:     backups_core_enums.BackupEncryptionNone,
		WalSegmentRepo: physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:    physical_repositories.GetWalHistoryRepository(),
		WatchDirRoot:   t.TempDir(),
		Logger:         logger.GetLogger(),
	})

	// Without the check the supervisor would stream until cancelled, so the bound turns that into a
	// failed assertion instead of a hung test.
	runCtx, cancelRun := context.WithTimeout(t.Context(), time.Minute)
	defer cancelRun()

	err := supervisor.Run(runCtx)

	require.NoError(t, err)
	assert.False(t, SlotExists(t, adminConn, walSlotName), "the slot of the deleted database must be dropped")
}
