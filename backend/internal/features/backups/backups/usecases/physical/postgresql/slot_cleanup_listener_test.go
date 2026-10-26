package usecases_physical_postgresql_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	databases_bridge "databasus-backend/internal/features/databases"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

// The only coverage for the database-removal drop site — the sole place the long-lived streamer slot
// is removed. It also pins that the per-backup slot is the one the database stores: computing another
// name would silently orphan the slot forever, since the DB row is deleted with the commit and
// RunStartupCleanup only scans existing rows.
func Test_OnDatabaseDeleting_DropsBothPerBackupAndStreamerSlotsAfterTheCommit(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	perBackupSlot := fixture.SourceDB.BackupSlotName
	streamerSlot := fixture.SourceDB.Engine.ReplicationSlotName
	require.NotEmpty(t, streamerSlot, "streamer slot name should be populated by BeforeCreate")

	require.NoError(t, pgphysical.NewSource(fixture.SourceDB.Engine).VerifySlot(t.Context(), logger.GetLogger()))

	// Simulate an orphan per-backup slot a crashed/interrupted backup left behind
	// that neither the next run nor startup cleanup recovered before removal.
	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", perBackupSlot)
	require.NoError(t, err, "pre-create orphan per-backup slot")

	// Defensive: if the listener fails to drop either slot, don't leak it into the
	// shared cluster (max_replication_slots is finite).
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = ANY($1)`,
			[]string{perBackupSlot, streamerSlot})
	})

	require.True(t, postgresql_executor.SlotExists(t, adminConn, perBackupSlot),
		"per-backup slot must exist before removal")
	require.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"streamer slot must exist before removal")

	require.NoError(t, runSlotCleanupInCommittedDeletion(t, fixture.DB.ID))

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, perBackupSlot),
		"the stored per-backup slot must be dropped on DB removal")
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"WAL streamer slot must be dropped on DB removal")
}

// Removal is the only drop site with no second chance: the row goes away with it, and with it any
// record of which cluster to go looking on. A listener that dialled the source directly would never
// reach a bastioned one, and both slots would pin WAL on the customer's master forever.
func Test_OnDatabaseDeleting_WhenTheSourceIsBehindABastion_DropsBothPerBackupAndStreamerSlots(
	t *testing.T,
) {
	fixture := postgresql_executor.SetupPhysicalDBForBackupBehindSshBastion(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	perBackupSlot := fixture.SourceDB.BackupSlotName
	streamerSlot := fixture.SourceDB.Engine.ReplicationSlotName
	require.NotEmpty(t, streamerSlot, "streamer slot name should be populated by BeforeCreate")

	for _, slotName := range []string{perBackupSlot, streamerSlot} {
		_, err := adminConn.Exec(context.Background(),
			"SELECT pg_create_physical_replication_slot($1, true)", slotName)
		require.NoError(t, err, "pre-create slot %s", slotName)
	}

	// Defensive: if the listener fails to drop either slot, don't leak it into the
	// cluster (max_replication_slots is finite).
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = ANY($1)`,
			[]string{perBackupSlot, streamerSlot})
	})

	require.NoError(t, runSlotCleanupInCommittedDeletion(t, fixture.DB.ID))

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, perBackupSlot),
		"per-backup slot must be dropped through the tunnel on DB removal")
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"WAL streamer slot must be dropped through the tunnel on DB removal")
}

// The slots stay orphaned on a source we cannot reach — accepted so a dead source can never block
// the metadata delete.
func Test_OnDatabaseDeleting_WhenSourceUnreachable_DoesNotBlockRemoval(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	require.NoError(t, storage.GetDb().Exec(
		"UPDATE databases_postgresql_physical SET host = '127.0.0.1', port = 1 WHERE database_id = ?",
		fixture.DB.ID,
	).Error)

	assert.NoError(t, runSlotCleanupInCommittedDeletion(t, fixture.DB.ID),
		"unreachable source must be logged + skipped, never block the metadata delete")
}

// runSlotCleanupInCommittedDeletion runs the handler in a transaction that commits, as the deletion
// does, so its after-commit hook drops the slots before it returns.
func runSlotCleanupInCommittedDeletion(t *testing.T, databaseID uuid.UUID) error {
	t.Helper()

	listener := postgresql_executor.NewPhysicalSlotCleanupListener(databases_bridge.GetModule, logger.GetLogger())

	return storage.GetTransactor().Transaction(t.Context(), func(ctx context.Context) error {
		return listener.OnDatabaseDeleting(ctx, databases.DatabaseDeleting{DatabaseID: databaseID})
	})
}

// The safety guard the removal listener relies on: DropWalSlot must refuse to drop a slot a consumer
// (our pg_receivewal) still holds, rather than terminate the session — dropping an active slot would
// crash the live streamer.
func Test_DropWalSlot_WhenStreamerSlotActive_RefusesAndKeepsSlot(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	streamerSlot := fixture.SourceDB.Engine.ReplicationSlotName

	stop := postgresql_executor.StartWalStreamerForTest(t, postgresql_executor.WalStreamerTestSpec{
		Fixture:      fixture,
		FileStore:    storages_bridge.GetModule(),
		WatchDirRoot: t.TempDir(),
	}).Stop
	defer stop()

	waitUntilSlotActive(t, adminConn, streamerSlot, 30*time.Second)

	err := pgphysical.NewSource(fixture.SourceDB.Engine).DropSlot(context.Background(), logger.GetLogger())
	require.Error(t, err, "DropWalSlot must refuse a slot held by an active consumer")
	assert.Contains(t, err.Error(), "is active")

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"a refused drop must leave the active slot intact")
}

func waitUntilSlotActive(t *testing.T, conn *pgx.Conn, slotName string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		var isActive bool
		err := conn.QueryRow(context.Background(),
			"SELECT COALESCE(active, false) FROM pg_replication_slots WHERE slot_name = $1",
			slotName,
		).Scan(&isActive)
		if err == nil && isActive {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("slot %s did not become active within %s", slotName, timeout)
}
