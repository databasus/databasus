package usecases_physical_postgresql_test

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func Test_SlotName_IsDeterministicForSameDbID(t *testing.T) {
	dbID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	first := postgresql_executor.SlotName(dbID)
	second := postgresql_executor.SlotName(dbID)

	assert.Equal(t, first, second)
}

func Test_SlotName_UsesValidPGIdentifierCharacters(t *testing.T) {
	identifier := regexp.MustCompile(`^[a-z0-9_]+$`)

	for range 100 {
		name := postgresql_executor.SlotName(uuid.New())

		assert.True(t, identifier.MatchString(name),
			"slot name %q must contain only lowercase letters, digits, and underscores", name)
	}
}

func Test_SlotName_HasExpectedPrefix(t *testing.T) {
	name := postgresql_executor.SlotName(uuid.New())

	assert.True(t, strings.HasPrefix(name, "databasus_basebackup_"),
		"slot name %q must start with databasus_basebackup_", name)
}

func Test_SlotName_IsUnderPGMaxNameLength(t *testing.T) {
	const pgMaxNameLength = 63

	for range 100 {
		name := postgresql_executor.SlotName(uuid.New())

		assert.LessOrEqual(t, len(name), pgMaxNameLength,
			"slot name %q exceeds PG NAMEDATALEN-1 (%d bytes)", name, pgMaxNameLength)
	}
}

func Test_BackupSlot_PresentDuringRun_GoneAfterCompleted(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	require.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must not exist before backup")

	pollConn, err := fixture.DB.PostgresqlPhysical.OpenInspectionConn(
		context.Background(), encryption.GetFieldEncryptor(),
	)
	require.NoError(t, err)
	defer func() { _ = pollConn.Close(context.Background()) }()

	var slotObserved atomic.Bool
	var pollCount atomic.Int64
	var observedSlots sync.Map

	backupDone := make(chan struct{})
	pollerReady := make(chan struct{})

	pollDone := make(chan struct{})
	go func() {
		defer close(pollDone)
		close(pollerReady)

		for {
			select {
			case <-backupDone:
				return
			default:
			}

			pollCount.Add(1)

			rows, queryErr := pollConn.Query(context.Background(),
				"SELECT slot_name FROM pg_replication_slots",
			)
			if queryErr != nil {
				continue
			}

			for rows.Next() {
				var name string
				if scanErr := rows.Scan(&name); scanErr != nil {
					continue
				}

				observedSlots.Store(name, struct{}{})

				if name == slotName {
					slotObserved.Store(true)
				}
			}
			rows.Close()

			if slotObserved.Load() {
				return
			}
		}
	}()

	<-pollerReady

	go func() {
		defer close(backupDone)
		backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	}()

	<-backupDone
	<-pollDone

	t.Logf("poll count during run: %d, slotObserved: %v", pollCount.Load(), slotObserved.Load())

	var slotsSeen []string
	observedSlots.Range(func(k, _ any) bool {
		slotsSeen = append(slotsSeen, k.(string))
		return true
	})
	t.Logf("slots observed during run: %v", slotsSeen)

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.True(t, slotObserved.Load(),
		"per-backup slot %q should have been observed in pg_replication_slots during the run", slotName)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"per-backup slot %q must be dropped after COMPLETED", slotName)
}

func Test_BackupSlot_AbsentWhenTimelineCheckFails_ChainBroken(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	wrongSysID := "9999999999999999999"
	fixture.DB.PostgresqlPhysical.SystemIdentifier = &wrongSysID
	require.NoError(t, storage.GetDb().Save(fixture.DB.PostgresqlPhysical).Error)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusChainBroken, nil, 30*time.Second)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must never be created when the timeline check fails (it runs outside WithBackupSlot)")
}

func Test_BackupSlot_OrphanFromPriorCrash_DropIfExistsRecovers(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create orphan slot")

	require.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"orphan slot must exist before backup starts")

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be cleaned up after backup completes (orphan recovered + new slot dropped)")
}

// neverProtected and alwaysProtected pin the two slot-cleanup outcomes:
// neverProtected expects every orphan slot dropped (no backup in flight);
// alwaysProtected simulates a backup still in flight, whose slot must be
// preserved.
func neverProtected(uuid.UUID) (bool, error) { return false, nil }

func alwaysProtected(uuid.UUID) (bool, error) { return true, nil }

func Test_RunStartupCleanup_WhenSlotProtected_PreservesSlot(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create the slot a running backup would hold")
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`,
			slotName)
	})

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), alwaysProtected))

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"a protected slot (backup still in flight) must NOT be dropped")
}

func Test_RunStartupCleanup_DropsOrphanSlots(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create orphan slot")

	require.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"orphan slot must exist before cleanup")

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"RunStartupCleanup must drop the per-backup orphan slot")
}

// Boot cleanup gets one 5s budget per database, SSH handshake included, and a bastioned source that
// misses it is skipped. What must not happen is missing it because the sweep never tried the tunnel
// at all and dialled an address only the bastion can see.
func Test_RunStartupCleanup_WhenTheSourceIsBehindABastion_DropsOrphanSlots(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackupBehindSshBastion(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create orphan slot")

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`, slotName)
	})

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"the sweep must reach a bastioned source through the tunnel")
}

func Test_RunStartupCleanup_PreservesStreamerSlot(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	streamerSlot := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	require.NotEmpty(t, streamerSlot, "streamer slot name should be populated by BeforeCreate")

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	require.NoError(t, fixture.DB.PostgresqlPhysical.VerifyWalSlot(
		t.Context(), logger.GetLogger(), encryption.GetFieldEncryptor(),
	))
	t.Cleanup(func() {
		_ = fixture.DB.PostgresqlPhysical.DropWalSlot(
			context.Background(), logger.GetLogger(), encryption.GetFieldEncryptor(),
		)
	})

	require.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"streamer slot must exist before cleanup")

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, streamerSlot),
		"RunStartupCleanup must NOT drop the streamer slot (different prefix)")
}

func Test_RunStartupCleanup_PreservesUnrelatedSlots(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	unrelatedSlot := "test_unrelated_slot_" + uuid.New().String()[:8]
	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", unrelatedSlot)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`,
			unrelatedSlot)
	})

	require.True(t, postgresql_executor.SlotExists(t, adminConn, unrelatedSlot),
		"unrelated slot must exist before cleanup")

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, unrelatedSlot),
		"RunStartupCleanup must NOT drop slots that don't match the per-backup prefix")
}

func Test_RunStartupCleanup_SkipsUnreachableSource_DoesNotFail(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	fixture.DB.PostgresqlPhysical.Host = "127.0.0.1"
	fixture.DB.PostgresqlPhysical.Port = 1
	require.NoError(t, storage.GetDb().Save(fixture.DB.PostgresqlPhysical).Error)

	err := postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected)

	assert.NoError(t, err,
		"unreachable source must be logged + skipped, not surfaced as a failure")
}

// The "source DB was down" path, where even the deferred drop's retry on a fresh connection cannot
// get through: the slot is left behind. The first boot still cannot reach the source (skip, slot
// survives); the next boot after the source returns must drop the orphan. Without this, an outage
// spanning the whole backup would orphan a slot until the next one runs.
func Test_RunStartupCleanup_RecoversOrphanWhenSourceBecomesReachable(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create the orphan a defer-drop against a down source could not remove")
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`, slotName)
	})

	originalHost := fixture.DB.PostgresqlPhysical.Host
	originalPort := fixture.DB.PostgresqlPhysical.Port

	// First boot: source still unreachable — cleanup skips, the orphan survives.
	fixture.DB.PostgresqlPhysical.Host = "127.0.0.1"
	fixture.DB.PostgresqlPhysical.Port = 1
	require.NoError(t, storage.GetDb().Save(fixture.DB.PostgresqlPhysical).Error)

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))
	require.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"unreachable source: the orphan slot must survive the first boot")

	// Second boot: source is back — the orphan must now be dropped.
	fixture.DB.PostgresqlPhysical.Host = originalHost
	fixture.DB.PostgresqlPhysical.Port = originalPort
	require.NoError(t, storage.GetDb().Save(fixture.DB.PostgresqlPhysical).Error)

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"reachable source: the orphan slot must be dropped on the next boot")
}

func Test_BackupSlot_PresentDuringIncrementalRun_GoneAfterFinish(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	fullRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotNil(t, fullRow.StopLSN, "FULL must have stop_lsn for the INCR to anchor on")

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	// The INCR opens WithBackupSlot only after the summarizer pre-check clears, so
	// cross a segment boundary and flush a summary past the FULL's stop_lsn first;
	// without it the pre-check turns the INCR into CHAIN_BROKEN before any slot is
	// created and the slot could never be observed.
	_, err = postgresql_executor.GenerateWalActivity(ctx, adminConn, 32*1024*1024)
	require.NoError(t, err)

	_, err = adminConn.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	_, err = adminConn.Exec(ctx, "SELECT pg_switch_wal()")
	require.NoError(t, err)

	require.NoError(t, postgresql_executor.WaitForWalSummaries(ctx, adminConn, *fullRow.StopLSN, 2*time.Minute))

	incrID := postgresql_executor.BuildAndClaimIncremental(t, fixture, nil)

	observed := postgresql_executor.RunBackupAndPoll(t, fixture, slotName, func() {
		backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), incrID, false)
	})

	postgresql_executor.WaitForBackupStatus(t, incrID, physical_enums.PhysicalBackupTypeIncremental,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.True(
		t,
		observed,
		"per-backup slot must be observed during the INCR run (the INCR wraps pg_basebackup in the same WithBackupSlot helper as the FULL)",
	)
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"per-backup slot must be dropped after the INCR finishes")
}

// Test_BackupSlot_DroppedAfterFailedIncrementalRun pins the failure-path drop for
// the incremental: the summarizer pre-check clears so the INCR opens
// WithBackupSlot, but a corrupted parent manifest makes pg_basebackup --incremental
// reject the backup. The per-backup slot must still be gone afterwards, since the
// WithBackupSlot defer drops it on the failure path too. This asserts only the
// post-run drop — observing the slot mid-run is racy when the backup fails fast.
func Test_BackupSlot_DroppedAfterFailedIncrementalRun(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	fullRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotNil(t, fullRow.StopLSN, "FULL must have stop_lsn for the INCR to anchor on")
	require.NotNil(t, fullRow.ManifestFileName, "FULL must have a manifest file the INCR anchors on")

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	// Clear the summarizer pre-check so the INCR reaches WithBackupSlot (otherwise
	// it bails to CHAIN_BROKEN before any slot is created and this test would prove
	// nothing about the failure-path drop).
	_, err = postgresql_executor.GenerateWalActivity(ctx, adminConn, 32*1024*1024)
	require.NoError(t, err)

	_, err = adminConn.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	_, err = adminConn.Exec(ctx, "SELECT pg_switch_wal()")
	require.NoError(t, err)

	require.NoError(t, postgresql_executor.WaitForWalSummaries(ctx, adminConn, *fullRow.StopLSN, 2*time.Minute))

	// Overwrite the parent manifest with garbage so pg_basebackup --incremental
	// rejects it inside WithBackupSlot. The pre-check reads stop_lsn from the row,
	// not this file, so it still clears and the slot is still created.
	require.NoError(t, fixture.Storage.SaveFile(
		t.Context(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		*fullRow.ManifestFileName,
		strings.NewReader("fake-manifest-bytes-pg_basebackup-will-reject-this"),
	))

	incrID := postgresql_executor.BuildAndClaimIncremental(t, fixture, nil)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), incrID, false)

	incrRow, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrID)
	require.NoError(t, err)
	require.NotEqual(t, physical_enums.PhysicalBackupStatusCompleted, incrRow.Status,
		"a corrupted parent manifest must fail the incremental; got status=%s", incrRow.Status)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"per-backup slot must be dropped after a failed INCR (the WithBackupSlot defer runs on the failure path)")
}

func Test_BackupSlot_HasExpectedCharacteristics(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)

	pollConn, err := fixture.DB.PostgresqlPhysical.OpenInspectionConn(
		context.Background(), encryption.GetFieldEncryptor(),
	)
	require.NoError(t, err)
	defer func() { _ = pollConn.Close(context.Background()) }()

	var (
		observed        atomic.Bool
		capturedType    atomic.Value
		capturedNonZero atomic.Bool
	)

	backupDone := make(chan struct{})
	pollerReady := make(chan struct{})
	pollDone := make(chan struct{})

	go func() {
		defer close(pollDone)
		close(pollerReady)

		for {
			select {
			case <-backupDone:
				return
			default:
			}

			var (
				slotType   string
				restartLSN string
			)

			queryErr := pollConn.QueryRow(context.Background(),
				`SELECT slot_type, COALESCE(restart_lsn::text, '0/0')
				   FROM pg_replication_slots WHERE slot_name = $1`,
				slotName,
			).Scan(&slotType, &restartLSN)
			if queryErr != nil {
				continue
			}

			capturedType.Store(slotType)
			capturedNonZero.Store(restartLSN != "0/0")
			observed.Store(true)
			return
		}
	}()

	<-pollerReady

	go func() {
		defer close(backupDone)
		backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	}()

	<-backupDone
	<-pollDone

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	require.True(t, observed.Load(), "per-backup slot must be observed during run")

	assert.Equal(t, "physical", capturedType.Load(),
		"per-backup slot must be physical type (not logical)")
	assert.True(t, capturedNonZero.Load(),
		"restart_lsn must be set (immediately_reserve=true reserves WAL at creation — fetch retention depends on it)")
}

func Test_BackupSlot_ConcurrentBackupsOnDifferentDBs_NoCollision(t *testing.T) {
	fixtureA := postgresql_executor.SetupPhysicalDBForBackup(t)
	fixtureB := postgresql_executor.SetupPhysicalDBForBackup(t)

	require.NotEqual(t, fixtureA.DB.PostgresqlPhysical.ID, fixtureB.DB.PostgresqlPhysical.ID)

	slotA := postgresql_executor.SlotName(fixtureA.DB.PostgresqlPhysical.ID)
	slotB := postgresql_executor.SlotName(fixtureB.DB.PostgresqlPhysical.ID)
	require.NotEqual(t, slotA, slotB, "different DBs must produce different slot names")

	var observedA, observedB atomic.Bool

	doneA := make(chan struct{})
	doneB := make(chan struct{})

	go func() {
		defer close(doneA)
		observedA.Store(postgresql_executor.RunBackupAndPoll(t, fixtureA, slotA, func() {
			backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixtureA.BackupID, false)
		}))
	}()

	go func() {
		defer close(doneB)
		observedB.Store(postgresql_executor.RunBackupAndPoll(t, fixtureB, slotB, func() {
			backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixtureB.BackupID, false)
		}))
	}()

	<-doneA
	<-doneB

	postgresql_executor.WaitForBackupStatus(t, fixtureA.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)
	postgresql_executor.WaitForBackupStatus(t, fixtureB.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.True(t, observedA.Load(), "slot A should be observed during DB A's backup")
	assert.True(t, observedB.Load(), "slot B should be observed during DB B's backup")

	adminConnA := postgresql_executor.OpenAdminConn(t, fixtureA)
	adminConnB := postgresql_executor.OpenAdminConn(t, fixtureB)

	assert.False(t, postgresql_executor.SlotExists(t, adminConnA, slotA), "slot A must be dropped after DB A's backup")
	assert.False(t, postgresql_executor.SlotExists(t, adminConnB, slotB), "slot B must be dropped after DB B's backup")
}

func Test_RunStartupCleanup_OrphanFromDeletedDB_NotCleaned(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)

	// Capture conn-info BEFORE the test deletes the PostgresqlPhysical
	// row — the cleanup at the end of this test opens a fresh connection,
	// because adminConn below is closed by defer before t.Cleanup runs
	// (defer fires before t.Cleanup) and reusing it would silently no-op.
	sourceDB := *fixture.DB.PostgresqlPhysical

	adminConn, err := fixture.DB.PostgresqlPhysical.OpenInspectionConn(
		context.Background(), encryption.GetFieldEncryptor(),
	)
	require.NoError(t, err)
	defer func() { _ = adminConn.Close(context.Background()) }()

	_, err = adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupConn, cleanupErr := sourceDB.OpenInspectionConn(
			context.Background(), encryption.GetFieldEncryptor(),
		)
		if cleanupErr != nil {
			return
		}
		defer func() { _ = cleanupConn.Close(context.Background()) }()

		_, _ = cleanupConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`,
			slotName)
	})

	require.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"orphan slot must exist before deletion")

	require.NoError(t, storage.GetDb().Delete(fixture.DB.PostgresqlPhysical).Error,
		"delete the PostgresqlPhysical row — simulates a crashed OnBeforeDatabaseRemove hook "+
			"that removed the physical-DB record but didn't drop the slot on source")

	require.NoError(t, postgresql_executor.RunStartupCleanup(t.Context(), logger.GetLogger(), neverProtected))

	assert.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"RunStartupCleanup iterates registered physical DBs only (Preload returns nil for the deleted "+
			"PostgresqlPhysical) — orphan slots from deleted DBs remain. This is a known limitation; "+
			"recovery requires manual cleanup on the source PG.")
}

func Test_BackupSlot_TwoBackupsInRow_SecondReusesNameOk(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	require.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be gone after first backup")

	secondID := uuid.New()
	secondRow := &physical_models.PhysicalFullBackup{
		ID:         secondID,
		DatabaseID: fixture.DB.ID,
		StorageID:  fixture.Storage.ID,
		TimelineID: 1,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		Encryption: backups_core_enums.BackupEncryptionNone,
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, physical_repositories.GetFullBackupRepository().Save(secondRow))
	t.Cleanup(func() {
		_ = physical_repositories.GetFullBackupRepository().DeleteByID(secondID)
	})

	claimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: fixture.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   secondID,
		})
	require.NoError(t, err)
	require.True(t, claimed, "second in-flight claim must succeed (first one was released on COMPLETED)")
	t.Cleanup(func() {
		_ = physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID)
	})

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), secondID, false)
	postgresql_executor.WaitForBackupStatus(t, secondID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be gone after second backup; drop-if-exists + defer drop are robust to repeat runs")
}

func Test_BackupSlot_PresentDuringRun_PG18(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackupVersion(t, "18")

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	observed := postgresql_executor.RunBackupAndPoll(t, fixture, slotName, func() {
		backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	})

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.True(t, observed,
		"per-backup slot must be observed during run on PG18 (same lifecycle as PG17)")
	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be dropped after COMPLETED on PG18")
}

func Test_BackupSlot_RetryAfterFailure_ReusesName(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	flakyStorage := storages.CreateTestFlakyS3Storage(*fixture.DB.WorkspaceID,
		"http://127.0.0.1:1")
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), flakyStorage.ID) })

	cfgService := backups_config_physical.GetBackupConfigService()
	cfg, err := cfgService.GetBackupConfigByDbId(fixture.DB.ID)
	require.NoError(t, err)

	cfg.StorageID = &flakyStorage.ID
	cfg.Storage = flakyStorage
	_, err = cfgService.SaveBackupConfig(t.Context(), cfg)
	require.NoError(t, err)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)

	finalRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotEqual(t, physical_enums.PhysicalBackupStatusCompleted, finalRow.Status,
		"first backup must not COMPLETED with flaky storage; got status=%s", finalRow.Status)

	require.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be cleaned up after failed first run (defer-drop must run even when SaveFile fails)")

	cfg, err = cfgService.GetBackupConfigByDbId(fixture.DB.ID)
	require.NoError(t, err)

	cfg.StorageID = &fixture.Storage.ID
	cfg.Storage = fixture.Storage
	_, err = cfgService.SaveBackupConfig(t.Context(), cfg)
	require.NoError(t, err)

	_ = physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID)

	retryID := uuid.New()
	retryRow := &physical_models.PhysicalFullBackup{
		ID:         retryID,
		DatabaseID: fixture.DB.ID,
		StorageID:  fixture.Storage.ID,
		TimelineID: 1,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		Encryption: backups_core_enums.BackupEncryptionNone,
		CreatedAt:  time.Now().UTC(),
	}
	require.NoError(t, physical_repositories.GetFullBackupRepository().Save(retryRow))
	t.Cleanup(func() {
		_ = physical_repositories.GetFullBackupRepository().DeleteByID(retryID)
	})

	claimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: fixture.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   retryID,
		})
	require.NoError(t, err)
	require.True(t, claimed)
	t.Cleanup(func() {
		_ = physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID)
	})

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), retryID, false)
	postgresql_executor.WaitForBackupStatus(t, retryID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be cleaned up after retry COMPLETED; both runs left no orphan")
}

func Test_BackupSlot_DroppedAfterCancelledMidRun(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	pollConn, err := fixture.DB.PostgresqlPhysical.OpenInspectionConn(
		context.Background(), encryption.GetFieldEncryptor(),
	)
	require.NoError(t, err)
	defer func() { _ = pollConn.Close(context.Background()) }()

	var slotObserved atomic.Bool

	backupDone := make(chan struct{})
	cancelTriggered := make(chan struct{})

	go func() {
		defer close(backupDone)
		backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)
	}()

	go func() {
		defer close(cancelTriggered)

		for {
			select {
			case <-backupDone:
				return
			default:
			}

			var exists bool
			queryErr := pollConn.QueryRow(context.Background(),
				"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
				slotName,
			).Scan(&exists)
			if queryErr != nil {
				continue
			}

			if exists {
				slotObserved.Store(true)
				_ = tasks_cancellation.GetRequester().RequestCancellation(context.Background(), fixture.BackupID)
				return
			}
		}
	}()

	<-backupDone
	<-cancelTriggered

	if !slotObserved.Load() {
		t.Fatal("backup completed too fast for cancellation to land while slot existed — timing-dependent regression")
	}

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"slot must be dropped after cancellation (defer uses context.Background() so it survives ctx cancel)")
}

// A bastion in front of the source makes a mid-backup connection loss routine, and the connection
// that dies is the very one holding the slot open. Without a retry the slot survives until the next
// backup runs, pinning WAL on the source for a whole full-backup cadence.
func Test_WithBackupSlot_WhenTheHeldConnectionDied_DropsTheSlotOverAFreshConnection(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	slotName := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	sourceDB := postgresql_executor.CreateReplicationOnlyRole(t, adminConn, fixture)

	err := postgresql_executor.WithBackupSlot(
		t.Context(),
		sourceDB,
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		func() error {
			require.True(t, postgresql_executor.SlotExists(t, adminConn, slotName),
				"the slot must be held while the backup runs")

			_, terminateErr := adminConn.Exec(context.Background(),
				`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename = $1`,
				sourceDB.Username)

			return terminateErr
		},
	)
	require.NoError(t, err)

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"the deferred drop must reopen a connection rather than give up on the dead one")
}
