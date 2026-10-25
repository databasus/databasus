//go:build testhelpers

package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	users_dto "databasus-backend/internal/features/users/dto"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/walmath"
)

// PhysicalDBFixture is a fully-wired physical DB ready for FULL backup
// dispatch: workspace, storage, notifier, DB row, backup config, in-flight
// claim, and a backup row in IN_PROGRESS. BackupID + DB are the two handles
// most tests need; the rest are kept so tests can mutate config or hit the
// source PG.
type PhysicalDBFixture struct {
	Owner     *users_dto.SignInResponseDTO
	Workspace *workspaces_models.Workspace
	Storage   *storages.Storage
	Notifier  *notifiers.Notifier
	DB        *databases.Database
	BackupID  uuid.UUID
}

// SetupPhysicalDBForBackup builds a PG17 fixture against the test source
// cluster. A missing container DSN fails the run loudly (see
// SetupPhysicalDBForBackupVersion) — it never skips.
func SetupPhysicalDBForBackup(t *testing.T) *PhysicalDBFixture {
	return SetupPhysicalDBForBackupVersion(t, "17")
}

// SetupPhysicalDBForBackupVersion builds a fixture against a specific PG major
// version. The container DSN env vars are validated non-empty in config.go at
// startup (it os.Exit(1)s when one is unset), so this helper does not re-check
// them — physical backup tests fail, never skip, when the source DB is
// unavailable.
func SetupPhysicalDBForBackupVersion(t *testing.T, version string) *PhysicalDBFixture {
	t.Helper()

	return setupPhysicalFixture(t, func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
		return databases.CreateTestPhysicalPostgresDatabase(workspaceID, notifier, version)
	})
}

// SetupPhysicalDBForStreamMtls builds a fixture against the replication-capable mTLS PG 17 cluster
// at host:port, with the DB's BackupType set to WAL_STREAM. Used by the FULL- and
// WAL-stream-over-mTLS tests. The caller starts the throwaway mTLS source and passes its endpoint.
func SetupPhysicalDBForStreamMtls(t *testing.T, host string, port int) *PhysicalDBFixture {
	t.Helper()

	return setupPhysicalFixture(t, func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
		return databases.CreateTestPhysicalPostgresDatabaseMtls(host, port, workspaceID, notifier)
	})
}

// SetupPhysicalDBForBackupNoSummary builds a PG17 fixture against a summarize_wal=off cluster at
// host:port, so the incremental pre-check reaches the SUMMARIZER_OFF branch deterministically. The
// caller starts the throwaway no-summary source and passes its endpoint.
func SetupPhysicalDBForBackupNoSummary(t *testing.T, host string, port int) *PhysicalDBFixture {
	t.Helper()

	return setupPhysicalFixture(t, func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
		return databases.CreateTestPhysicalPostgresDatabaseNoSummary(host, port, workspaceID, notifier, "17")
	})
}

// BuildAndClaimIncremental seeds an IN_PROGRESS incremental row rooted on the
// fixture's FULL (fixture.BackupID) and takes the per-database in-flight claim,
// mirroring what the scheduler does before dispatching an INCR. parentIncrID is
// nil for the first incremental (parent is the root FULL). Both the row and the
// claim are cleaned up at test end. Returns the new incremental's ID.
func BuildAndClaimIncremental(
	t *testing.T,
	fixture *PhysicalDBFixture,
	parentIncrID *uuid.UUID,
) uuid.UUID {
	t.Helper()

	incrID := uuid.New()
	incrRow := &physical_models.PhysicalIncrementalBackup{
		ID:                        incrID,
		DatabaseID:                fixture.DB.ID,
		StorageID:                 fixture.Storage.ID,
		RootFullBackupID:          fixture.BackupID,
		ParentIncrementalBackupID: parentIncrID,
		TimelineID:                1,
		Status:                    physical_enums.PhysicalBackupStatusInProgress,
		Encryption:                backups_core_enums.BackupEncryptionNone,
		CreatedAt:                 time.Now().UTC(),
	}

	require.NoError(t, physical_repositories.GetIncrementalBackupRepository().Save(incrRow))
	t.Cleanup(func() {
		_ = physical_repositories.GetIncrementalBackupRepository().DeleteByID(incrID)
	})

	claimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: fixture.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeIncremental,
			BackupID:   incrID,
		})
	require.NoError(t, err)
	require.True(t, claimed, "INCR in-flight claim must succeed after the FULL released the slot")
	t.Cleanup(func() {
		_ = physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID)
	})

	return incrID
}

// SetupPhysicalDBForScheduledBackupVersion wires a physical fixture for the
// scheduler-driven, HTTP-API E2E tests. Unlike SetupPhysicalDBForBackup* it
// leaves backups DISABLED and seeds no backup row (BackupID is uuid.Nil): the
// test enables backups and triggers FULL/INCR through the controller while the
// scheduler started in TestMain executes them. backupType lands
// on the source DB (FULL_AND_INCREMENTAL to build chains, or
// FULL_INCREMENTAL_AND_WAL_STREAM to also stream WAL) because the scheduler's
// incremental eligibility reads that DB-level field, which the config API can't
// set.
func SetupPhysicalDBForScheduledBackupVersion(
	t *testing.T,
	host string,
	port int,
	version string,
	backupType postgresql_physical.BackupType,
) *PhysicalDBFixture {
	t.Helper()

	return wirePhysicalDBFixture(t, func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
		return databases.CreateTestPhysicalPostgresDatabaseWithType(
			host,
			port,
			workspaceID,
			notifier,
			version,
			backupType,
		)
	})
}

func SetupPhysicalDBFixtureWithTunnel(
	t *testing.T,
	spec databases.PhysicalTestDatabaseSpec,
) *PhysicalDBFixture {
	t.Helper()

	return wirePhysicalDBFixture(t, func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
		return databases.CreateTestPhysicalPostgresDatabaseWithTunnel(spec, workspaceID, notifier)
	})
}

// The source publishes no port, so every connection a test on this fixture makes has to go through
// the bastion — a direct route would keep it green after the tunnel stopped being used.
func SetupPhysicalDBForBackupBehindSshBastion(t *testing.T) *PhysicalDBFixture {
	t.Helper()

	topology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")

	return SetupPhysicalDBFixtureWithTunnel(t, databases.PhysicalTestDatabaseSpec{
		Host:       topology.Database.Host,
		Port:       topology.Database.Port,
		VersionTag: "17",
		BackupType: postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		SshTunnel:  bastion.GetTunnelConfig(topology),
	})
}

// wirePhysicalDBFixture provisions the scaffolding every physical fixture needs —
// workspace, storage, notifier, the source DB (via createDB), populated data, and
// the persisted system_identifier — returning it with backups still disabled and
// no backup row seeded (BackupID is uuid.Nil). setupPhysicalFixture builds on it
// for the direct-MakeBackup pattern; the scheduler-driven fixture returns it as is.
func wirePhysicalDBFixture(
	t *testing.T,
	createDB func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database,
) *PhysicalDBFixture {
	t.Helper()

	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
	)

	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	testStorage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), testStorage.ID) })

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	t.Cleanup(func() { notifiers.RemoveTestNotifier(notifier) })

	db := createDB(workspace.ID, notifier)
	t.Cleanup(func() { databases.RemoveTestDatabase(t.Context(), db) })

	encryptor := encryption.GetFieldEncryptor()
	log := logger.GetLogger()

	// Through the tunnel, exactly as the database service does it, so a bastioned fixture reaches its
	// source and the discovered identity is carried back onto the row that gets persisted below.
	tunneledDatabase, err := postgresql_physical.OpenTunnel(t.Context(), postgresql_physical.OpenTunnelSpec{
		Database:  db.PostgresqlPhysical,
		Logger:    log,
		Encryptor: encryptor,
	})
	require.NoError(t, err)
	t.Cleanup(tunneledDatabase.Close)

	require.NoError(t, tunneledDatabase.GetDatabaseThroughTunnel().PopulateDbData(log, encryptor))
	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	// CreateTestPhysicalPostgresDatabase saved the row before PopulateDbData ran,
	// so the captured system_identifier lives only on the in-memory object. The
	// backuper reloads the DB from the repo and the manifest's System-Identifier
	// comes from the stored row, so persist it here — mirroring production, where
	// the database service populates before saving.
	require.NoError(t, storage.GetDb().
		Model(db.PostgresqlPhysical).
		Update("system_identifier", db.PostgresqlPhysical.SystemIdentifier).Error)

	return &PhysicalDBFixture{
		Owner:     owner,
		Workspace: workspace,
		Storage:   testStorage,
		Notifier:  notifier,
		DB:        db,
	}
}

// setupPhysicalFixture is the shared body behind the version / mTLS fixtures:
// it wires the DB, enables backups, and seeds an IN_PROGRESS FULL row + in-flight
// claim for the direct-MakeBackup pattern.
func setupPhysicalFixture(
	t *testing.T,
	createDB func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database,
) *PhysicalDBFixture {
	t.Helper()

	fixture := wirePhysicalDBFixture(t, createDB)

	cfgService := backups_config_physical.GetBackupConfigService()

	cfg, err := cfgService.GetBackupConfigByDbId(fixture.DB.ID)
	require.NoError(t, err)

	cfg.IsBackupsEnabled = true
	cfg.StorageID = &fixture.Storage.ID
	cfg.Storage = fixture.Storage
	cfg.Encryption = backups_core_enums.BackupEncryptionNone
	cfg.PostgresqlPhysical = fixture.DB.PostgresqlPhysical
	cfg.FullBackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: new("04:00"),
	}

	_, err = cfgService.SaveBackupConfig(t.Context(), cfg)
	require.NoError(t, err)

	backupID := uuid.New()
	backupRow := &physical_models.PhysicalFullBackup{
		ID:         backupID,
		DatabaseID: fixture.DB.ID,
		StorageID:  fixture.Storage.ID,
		TimelineID: 1,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		Encryption: backups_core_enums.BackupEncryptionNone,
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, physical_repositories.GetFullBackupRepository().Save(backupRow))
	t.Cleanup(func() {
		_ = physical_repositories.GetFullBackupRepository().DeleteByID(backupID)
	})

	claimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: fixture.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   backupID,
		})
	require.NoError(t, err)
	require.True(t, claimed)
	t.Cleanup(func() {
		_ = physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID)
	})

	fixture.BackupID = backupID

	return fixture
}

// Through the production opener rather than a hand-rolled dial: a bastioned source publishes no
// port, so a test that reached it any other way would stay green after the tunnel stopped being
// used. With no tunnel configured OpenTunnel hands back the original, so direct fixtures are
// unaffected.
func OpenAdminConn(t *testing.T, fixture *PhysicalDBFixture) *pgx.Conn {
	t.Helper()

	encryptor := encryption.GetFieldEncryptor()

	tunneledDatabase, err := postgresql_physical.OpenTunnel(t.Context(), postgresql_physical.OpenTunnelSpec{
		Database:  fixture.DB.PostgresqlPhysical,
		Logger:    logger.GetLogger(),
		Encryptor: encryptor,
	})
	require.NoError(t, err)
	t.Cleanup(tunneledDatabase.Close)

	conn, err := tunneledDatabase.GetDatabaseThroughTunnel().
		OpenInspectionConn(context.Background(), encryptor)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

// SlotExists reports whether the named slot is present in pg_replication_slots.
func SlotExists(t *testing.T, conn *pgx.Conn, slotName string) bool {
	t.Helper()

	var exists bool
	err := conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
		slotName,
	).Scan(&exists)
	require.NoError(t, err)

	return exists
}

// RunBackupAndPoll spawns runBackup in a goroutine while a second goroutine
// polls pg_replication_slots for slotName on its own connection. Returns
// once runBackup completes. The bool is set when the slot was seen at least
// once during the run.
func RunBackupAndPoll(
	t *testing.T,
	fixture *PhysicalDBFixture,
	slotName string,
	runBackup func(),
) bool {
	t.Helper()

	pollConn, err := fixture.DB.PostgresqlPhysical.OpenInspectionConn(
		context.Background(), encryption.GetFieldEncryptor(),
	)
	require.NoError(t, err)
	defer func() { _ = pollConn.Close(context.Background()) }()

	var observed atomic.Bool

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

			var exists bool
			queryErr := pollConn.QueryRow(context.Background(),
				"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
				slotName,
			).Scan(&exists)
			if queryErr == nil && exists {
				observed.Store(true)
				return
			}
		}
	}()

	<-pollerReady

	go func() {
		defer close(backupDone)
		runBackup()
	}()

	<-backupDone
	<-pollDone

	return observed.Load()
}

// GenerateWalActivity inserts rows into a throwaway LOGGED table until the
// cluster's pg_current_wal_lsn() advances by at least minBytes. The table is
// dropped on return. UNLOGGED/TEMP would skip WAL — LOGGED guarantees the
// WAL advance the test is asserting on.
func GenerateWalActivity(
	ctx context.Context,
	conn *pgx.Conn,
	minBytes int64,
) (int64, error) {
	tableName := "wal_activity_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]

	// Test helper: the table name is generated from a UUID and the setting values come from the server itself.
	// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
	if _, err := conn.Exec(
		ctx,
		fmt.Sprintf(`CREATE TABLE %s (id BIGSERIAL PRIMARY KEY, payload TEXT)`, tableName),
	); err != nil {
		return 0, fmt.Errorf("create wal activity table: %w", err)
	}

	defer func() {
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName))
	}()

	var startLSN walmath.LSN
	if err := conn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&startLSN); err != nil {
		return 0, fmt.Errorf("read start LSN: %w", err)
	}

	const rowsPerBatch = 1000

	const payloadSize = 1024

	for {
		// Test helper: the table name is generated from a UUID and the setting values come from the server itself.
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		_, err := conn.Exec(
			ctx,
			fmt.Sprintf(
				`INSERT INTO %s (payload) SELECT repeat('x', %d) FROM generate_series(1, %d)`,
				tableName, payloadSize, rowsPerBatch,
			),
		)
		if err != nil {
			return 0, fmt.Errorf("insert wal activity rows: %w", err)
		}

		var currentLSN walmath.LSN
		if err := conn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&currentLSN); err != nil {
			return 0, fmt.Errorf("read current LSN: %w", err)
		}

		delta := int64(currentLSN) - int64(startLSN)
		if delta >= minBytes {
			return delta, nil
		}

		if ctx.Err() != nil {
			return delta, ctx.Err()
		}
	}
}

// WaitForWalSummaries polls pg_available_wal_summaries() until the summarizer
// covers untilLSN or the timeout expires.
func WaitForWalSummaries(
	ctx context.Context,
	conn *pgx.Conn,
	untilLSN walmath.LSN,
	timeout time.Duration,
) error {
	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		var exists bool
		err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_available_wal_summaries()
				WHERE start_lsn <= $1::pg_lsn AND end_lsn >= $1::pg_lsn
			)
		`, untilLSN.String()).Scan(&exists)
		if err != nil {
			return fmt.Errorf("query pg_available_wal_summaries: %w", err)
		}

		if exists {
			return nil
		}

		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("wal summary coverage of %s not reached within %s", untilLSN.String(), timeout)
}

// WaitForBackupStatus polls the typed table for backupID until it reaches
// the expected status (and optional error_reason) or the timeout expires.
// kind selects which table (FULL or INCREMENTAL).
func WaitForBackupStatus(
	t *testing.T,
	backupID uuid.UUID,
	kind physical_enums.PhysicalBackupType,
	expectedStatus physical_enums.PhysicalBackupStatus,
	expectedReason *physical_enums.PhysicalBackupErrorReason,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		status, reason, found := readBackupStatus(t, backupID, kind)
		if found && status == expectedStatus && reasonsMatch(reason, expectedReason) {
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	status, reason, _ := readBackupStatus(t, backupID, kind)

	t.Fatalf(
		"backup %s did not reach status=%s reason=%s within %s (observed status=%s reason=%s)",
		backupID, expectedStatus, reasonString(expectedReason), timeout,
		status, reasonString(reason),
	)
}

func readBackupStatus(
	t *testing.T,
	backupID uuid.UUID,
	kind physical_enums.PhysicalBackupType,
) (physical_enums.PhysicalBackupStatus, *physical_enums.PhysicalBackupErrorReason, bool) {
	t.Helper()

	switch kind {
	case physical_enums.PhysicalBackupTypeFull:
		var row physical_models.PhysicalFullBackup
		if err := storage.GetDb().Where("id = ?", backupID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", nil, false
			}

			t.Fatalf("read full backup row: %v", err)
		}

		return row.Status, row.ErrorReason, true

	case physical_enums.PhysicalBackupTypeIncremental:
		var row physical_models.PhysicalIncrementalBackup
		if err := storage.GetDb().Where("id = ?", backupID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", nil, false
			}

			t.Fatalf("read incr backup row: %v", err)
		}

		return row.Status, row.ErrorReason, true
	}

	t.Fatalf("unsupported backup kind: %s", kind)

	return "", nil, false
}

func reasonsMatch(a, b *physical_enums.PhysicalBackupErrorReason) bool {
	if b == nil {
		return true
	}

	if a == nil {
		return false
	}

	return *a == *b
}

func reasonString(r *physical_enums.PhysicalBackupErrorReason) string {
	if r == nil {
		return "<nil>"
	}

	return string(*r)
}

// SetSummarizerEnabled toggles summarize_wal at runtime via ALTER SYSTEM +
// pg_reload_conf. Auto-restores on t.Cleanup so tests can flip the
// summarizer mid-test without leaking state across cases.
func SetSummarizerEnabled(t *testing.T, conn *pgx.Conn, enabled bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var previous string
	if err := conn.QueryRow(
		ctx,
		"SELECT setting FROM pg_settings WHERE name = 'summarize_wal'",
	).Scan(&previous); err != nil {
		t.Fatalf("read summarize_wal: %v", err)
	}

	desired := "off"
	if enabled {
		desired = "on"
	}

	if previous == desired {
		return
	}

	// Test helper: the table name is generated from a UUID and the setting values come from the server itself.
	// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
	if _, err := conn.Exec(
		ctx,
		fmt.Sprintf("ALTER SYSTEM SET summarize_wal = '%s'", desired),
	); err != nil {
		t.Fatalf("ALTER SYSTEM summarize_wal=%s: %v", desired, err)
	}

	if _, err := conn.Exec(ctx, "SELECT pg_reload_conf()"); err != nil {
		t.Fatalf("pg_reload_conf: %v", err)
	}

	t.Cleanup(func() {
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer restoreCancel()

		// Test helper: the table name is generated from a UUID and the setting values come from the server itself.
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		if _, err := conn.Exec(restoreCtx,
			fmt.Sprintf("ALTER SYSTEM SET summarize_wal = '%s'", previous),
		); err != nil {
			t.Logf("restore summarize_wal=%s: %v", previous, err)

			return
		}

		_, _ = conn.Exec(restoreCtx, "SELECT pg_reload_conf()")
	})
}

// ExpireWalSummaries forces summary file expiry by temporarily dropping
// wal_summary_keep_time to 1 minute and waiting one sweep cycle. Auto-
// restores the original value on t.Cleanup.
func ExpireWalSummaries(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var previous string
	if err := conn.QueryRow(
		ctx,
		"SELECT setting FROM pg_settings WHERE name = 'wal_summary_keep_time'",
	).Scan(&previous); err != nil {
		t.Fatalf("read wal_summary_keep_time: %v", err)
	}

	if _, err := conn.Exec(
		ctx,
		"ALTER SYSTEM SET wal_summary_keep_time = '1min'",
	); err != nil {
		t.Fatalf("ALTER SYSTEM wal_summary_keep_time: %v", err)
	}

	if _, err := conn.Exec(ctx, "SELECT pg_reload_conf()"); err != nil {
		t.Fatalf("pg_reload_conf: %v", err)
	}

	t.Cleanup(func() {
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer restoreCancel()

		// Test helper: the table name is generated from a UUID and the setting values come from the server itself.
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		if _, err := conn.Exec(restoreCtx,
			fmt.Sprintf("ALTER SYSTEM SET wal_summary_keep_time = '%s'", previous),
		); err != nil {
			t.Logf("restore wal_summary_keep_time: %v", err)

			return
		}

		_, _ = conn.Exec(restoreCtx, "SELECT pg_reload_conf()")
	})
}

// ForceWalRotation generates a little WAL then forces the current segment to
// rotate (pg_switch_wal), so pg_receivewal finalizes a segment the uploader can
// pick up. Returns the LSN after the switch. pg_receivewal only uploads fully
// rotated segments, so a streamer test must rotate explicitly rather than wait
// for the 16 MB boundary.
func ForceWalRotation(ctx context.Context, conn *pgx.Conn) (walmath.LSN, error) {
	if _, err := GenerateWalActivity(ctx, conn, 1); err != nil {
		return 0, err
	}

	var switchLSN walmath.LSN
	if err := conn.QueryRow(ctx, "SELECT pg_switch_wal()::text").Scan(&switchLSN); err != nil {
		return 0, fmt.Errorf("pg_switch_wal: %w", err)
	}

	return switchLSN, nil
}

// WaitForCommittedWalSegmentCount polls the catalog until at least minCount
// committed (file_name NOT NULL) WAL segments exist for the database, or the
// timeout expires. Committed segments are the ones the uploader durably archived.
func WaitForCommittedWalSegmentCount(
	t *testing.T,
	databaseID uuid.UUID,
	minCount int,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		var count int64

		err := storage.GetDb().
			Model(&physical_models.PhysicalWalSegment{}).
			Where("database_id = ? AND file_name IS NOT NULL", databaseID).
			Count(&count).Error
		require.NoError(t, err)

		if count >= int64(minCount) {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("did not observe %d committed wal segments for %s within %s", minCount, databaseID, timeout)
}

// CountCommittedWalSegments returns the number of committed (file_name NOT NULL)
// WAL segments archived for the database — used to assert progress past a known
// baseline (e.g. that a post-rebuild segment landed, not just pre-rebuild ones).
func CountCommittedWalSegments(t *testing.T, databaseID uuid.UUID) int64 {
	t.Helper()

	var count int64
	err := storage.GetDb().
		Model(&physical_models.PhysicalWalSegment{}).
		Where("database_id = ? AND file_name IS NOT NULL", databaseID).
		Count(&count).Error
	require.NoError(t, err)

	return count
}

// WalStreamerForTest exposes the watch dir so a test can restart a streamer
// against the queue an earlier run left behind.
type WalStreamerForTest struct {
	Supervisor *WalStreamSupervisor
	WatchDir   string
	Stop       func()
}

// fixedStorageLocator resolves every storage ID to the one saver a test handed in,
// so the store writes and deletes exactly where the test can observe it.
type fixedStorageLocator struct {
	storage storages.StorageFileSaver
}

func (l fixedStorageLocator) GetFileWriter(
	context.Context, uuid.UUID,
) (storage_files.FileWriter, error) {
	return l.storage, nil
}

func (l fixedStorageLocator) GetFileRemover(
	context.Context, uuid.UUID,
) (storage_files.FileRemover, error) {
	return l.storage, nil
}

func NewTestFileStore(saver storages.StorageFileSaver) *storage_files.Store {
	return storage_files.NewStore(storage_files.Dependencies{
		Repository:     &storage_files.PendingDeletionRepository{},
		Locator:        fixedStorageLocator{storage: saver},
		FieldEncryptor: encryption.GetFieldEncryptor(),
		Logger:         logger.GetLogger(),
		Timings:        storage_files.TimingsForTest(),
	})
}

type WalStreamerTestSpec struct {
	Fixture                   *PhysicalDBFixture
	Storage                   storages.StorageFileSaver
	FileStore                 *storage_files.Store
	WatchDirRoot              string
	WalLagThresholdBytes      int64
	ForcedRotationInterval    time.Duration
	ArchiveStalenessThreshold time.Duration
	OnSlotRebuilt             func(ctx context.Context, reason string) error
	OnChainAtRisk             func(report ChainRiskReport)

	// Overrides the local queue size at which the receiver is paused, so a test
	// can reach back pressure with a few segments instead of half a gigabyte.
	// Zero keeps the production size.
	LocalQueuePauseBytes int64
}

// A replication slot is meant to outlive a streamer restart (so WAL is never
// lost), so the supervisor deliberately does NOT drop it on shutdown. In tests
// that would leak one slot per run until max_replication_slots is exhausted, so
// drop this streamer's slot here. Registered before the caller's stop cleanup,
// it runs (LIFO) right after the streamer has stopped and before the conn closes.
func StartWalStreamerForTest(t *testing.T, spec WalStreamerTestSpec) *WalStreamerForTest {
	t.Helper()

	fixture := spec.Fixture

	adminConn := OpenAdminConn(t, fixture)
	t.Cleanup(func() {
		DropReplicationSlotExternally(t, adminConn, fixture.DB.PostgresqlPhysical.ReplicationSlotName)
	})

	if spec.FileStore == nil {
		spec.FileStore = NewTestFileStore(spec.Storage)
	}

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:                fixture.DB.ID,
		SourceDB:                  fixture.DB.PostgresqlPhysical,
		StorageID:                 fixture.Storage.ID,
		FileStore:                 spec.FileStore,
		Encryption:                backups_core_enums.BackupEncryptionNone,
		FieldEncryptor:            encryption.GetFieldEncryptor(),
		WalSegmentRepo:            physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:               physical_repositories.GetWalHistoryRepository(),
		WatchDirRoot:              spec.WatchDirRoot,
		WalLagThresholdBytes:      spec.WalLagThresholdBytes,
		ForcedRotationInterval:    spec.ForcedRotationInterval,
		ArchiveStalenessThreshold: spec.ArchiveStalenessThreshold,
		OnSlotRebuilt:             spec.OnSlotRebuilt,
		OnChainAtRisk:             spec.OnChainAtRisk,
		Logger:                    logger.GetLogger(),
	})

	if spec.LocalQueuePauseBytes > 0 {
		supervisor.setLocalQueuePauseBytes(spec.LocalQueuePauseBytes)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = supervisor.Run(ctx)
	}()

	return &WalStreamerForTest{
		Supervisor: supervisor,
		WatchDir:   filepath.Join(spec.WatchDirRoot, "wal-queue", fixture.DB.ID.String()),
		Stop: func() {
			cancel()

			select {
			case <-done:
			case <-time.After(30 * time.Second):
				t.Log("streamer did not stop within timeout")
			}
		},
	}
}

// chain_view only treats a COMPLETED FULL as a chain anchor, so streamer tests
// that assert on chain shape need one to anchor against.
func MarkFullCompleted(t *testing.T, fullID uuid.UUID, timelineID int, startLSN, stopLSN walmath.LSN) {
	t.Helper()

	fileName := "full-" + fullID.String()

	err := storage.GetDb().
		Model(&physical_models.PhysicalFullBackup{}).
		Where("id = ?", fullID).
		Updates(map[string]any{
			"status":         physical_enums.PhysicalBackupStatusCompleted,
			"timeline_id":    timelineID,
			"start_lsn":      startLSN.String(),
			"stop_lsn":       stopLSN.String(),
			"file_name":      fileName,
			"backup_size_mb": 1.0,
			"completed_at":   time.Now().UTC(),
		}).Error
	require.NoError(t, err)
}

// SlotLagBytes returns the source slot's lag in bytes (distance from the slot's
// restart_lsn to the cluster's current WAL LSN). Zero when the slot is absent.
func SlotLagBytes(t *testing.T, conn *pgx.Conn, slotName string) int64 {
	t.Helper()

	var lag int64
	err := conn.QueryRow(context.Background(), `
		SELECT COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn), 0)::bigint
		FROM pg_replication_slots
		WHERE slot_name = $1
	`, slotName).Scan(&lag)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0
	}
	require.NoError(t, err)

	return lag
}

// ForceReplicationLag advances the source WAL by at least minBytes (rotating a
// segment so the change is durable). With no consumer attached — or a consumer
// that cannot drain — this grows the slot's restart_lsn distance, the signal the
// lag monitor reads.
func ForceReplicationLag(t *testing.T, conn *pgx.Conn, minBytes int64) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := GenerateWalActivity(ctx, conn, minBytes)
	require.NoError(t, err)

	if _, err := conn.Exec(ctx, "SELECT pg_switch_wal()"); err != nil {
		t.Fatalf("pg_switch_wal: %v", err)
	}
}

// WaitUntilSlotLag polls until the slot's lag reaches at least minLagBytes or the
// timeout expires.
func WaitUntilSlotLag(t *testing.T, conn *pgx.Conn, slotName string, minLagBytes int64, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		if SlotLagBytes(t, conn, slotName) >= minLagBytes {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("slot %s did not reach lag %d bytes within %s", slotName, minLagBytes, timeout)
}

// WaitForWalGap polls FindWalGapsInChain for the chain rooted at fullID until at
// least one gap appears (or the timeout expires) and returns the gaps. A gap is
// the derived record of a WAL discontinuity — no catalog marker row exists.
func WaitForWalGap(t *testing.T, fullID uuid.UUID, timeout time.Duration) []chain_view.LSNRange {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		gaps, err := chain_view.GetChainViewService().FindWalGapsInChain(fullID)
		require.NoError(t, err)

		if len(gaps) > 0 {
			return gaps
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("no wal gap appeared in chain %s within %s", fullID, timeout)

	return nil
}

// WaitForExtendableChain polls until the database has an extendable chain (newest
// COMPLETED FULL with no downstream CHAIN_BROKEN INCR) and returns it. A lossy
// chain (internal WAL gap) is still extendable, so this also covers the
// gap-then-still-extendable assertion.
func WaitForExtendableChain(t *testing.T, databaseID uuid.UUID, timeout time.Duration) *chain_view.ChainView {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		chain, err := chain_view.GetChainViewService().FindLastExtendableChainByDatabase(databaseID)
		require.NoError(t, err)

		if chain != nil {
			return chain
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("no extendable chain for database %s within %s", databaseID, timeout)

	return nil
}

// DropReplicationSlotExternally simulates slot loss: it terminates any active
// consumer backend on the slot, then drops it. Used to force the post-rebuild
// WAL gap that a real slot loss produces. The caller must stop our own streamer
// first so the slot is not re-acquired between terminate and drop.
func DropReplicationSlotExternally(t *testing.T, conn *pgx.Conn, slotName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, _ = conn.Exec(ctx, `
		SELECT pg_terminate_backend(active_pid)
		FROM pg_replication_slots
		WHERE slot_name = $1 AND active_pid IS NOT NULL
	`, slotName)

	deadline := time.Now().UTC().Add(10 * time.Second)
	for time.Now().UTC().Before(deadline) {
		_, err := conn.Exec(ctx, "SELECT pg_drop_replication_slot($1)", slotName)
		if err == nil {
			return
		}

		if !SlotExists(t, conn, slotName) {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("could not drop replication slot %s", slotName)
}
