//go:build testhelpers

package physicaltesting

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/walmath"
)

const (
	restoreWorkDir     = "/restore"
	restoredPgUser     = "testuser"
	restoredPgPassword = "testpassword"
	restoredPgDatabase = "testdb"

	// Replaying a PITR chain is disk-bound and runs while the test holds several containers, so this
	// is sized for a loaded machine rather than for the few seconds it takes when idle.
	recoveryCompletionTimeout = 3 * time.Minute
)

// setupReplicationOnlyFixture boots a throwaway replication-capable source for one PostgreSQL major,
// wires a scheduler-driven physical fixture against it, and switches its backup credentials to a
// replication-only user over the API — the common preamble every physical E2E shares. The returned
// router is the one all subsequent API calls go through.
func setupReplicationOnlyFixture(
	t *testing.T,
	version string,
	image string,
	backupType postgresql_physical.BackupType,
	sourceOptions ...containers.PhysicalPostgresOption,
) (*gin.Engine, *postgresql_executor.PhysicalDBFixture) {
	t.Helper()

	source := containers.StartPhysicalPostgres(t, image, sourceOptions...)

	router := newPhysicalTestRouter()
	fixture := postgresql_executor.SetupPhysicalDBForScheduledBackupVersion(
		t, source.Host, source.Port, version, backupType)
	useReplicationOnlyUserViaAPI(t, router, fixture)

	return router, fixture
}

// prepareRestoreTarget boots a throwaway idle restore-target container whose major matches the
// source backup's (a PG 18 PGDATA cannot start under PG 17). It is terminated with its tmpfs /restore
// when the test ends, so no pre/post wipe is needed.
func prepareRestoreTarget(t *testing.T, image string) containers.RestoreTarget {
	t.Helper()

	return containers.StartPhysicalRestoreTarget(t, image)
}

// restoredClusterDir is where the recovery script builds PGDATA under restoreWorkDir, mirroring the
// image's layout: PG 18 nests <major>/docker, PG <=17 keeps <out>/data.
func restoredClusterDir(image string) string {
	if containers.PostgresMajorVersion(image) >= 18 {
		return restoreWorkDir + "/" + strconv.Itoa(containers.PostgresMajorVersion(image)) + "/docker"
	}

	return restoreWorkDir + "/data"
}

// hostVolumeDirForMount maps the recovery script's output dir to the directory that must be bind-
// mounted at the image's data VOLUME: PG 18's volume root is the output dir itself (the cluster
// nests at <out>/<major>/docker), while PG <=17's volume is PGDATA, i.e. <out>/data.
func hostVolumeDirForMount(image, outDir string) string {
	if containers.PostgresMajorVersion(image) >= 18 {
		return outDir
	}

	return filepath.Join(outDir, "data")
}

// hostClusterDir is the on-host PGDATA the recovery script builds under outDir, mirroring
// restoredClusterDir but rooted at a real host path so a test can read the written config files.
func hostClusterDir(image, outDir string) string {
	if containers.PostgresMajorVersion(image) >= 18 {
		return filepath.Join(outDir, strconv.Itoa(containers.PostgresMajorVersion(image)), "docker")
	}

	return filepath.Join(outDir, "data")
}

// runRecoveryScriptOnHost runs the server-shipped recovery script on the host in --combine-image mode
// (pg_combinebackup runs in a throwaway postgres container; zstd/tar/docker live on the host), exactly
// as a user pipes curl | sh. It returns the script's combined output and its error so callers can
// assert either success or a refusal.
func runRecoveryScriptOnHost(
	t *testing.T,
	scriptPath, bundlePath, outDir, image string,
	targetTime *time.Time,
) ([]byte, error) {
	t.Helper()

	args := []string{scriptPath, "--combine-image", image}
	if targetTime != nil {
		args = append(args, "--target-time", targetTime.UTC().Format("2006-01-02 15:04:05-07:00"))
	}
	args = append(args, bundlePath, outDir)

	out, err := exec.CommandContext(t.Context(), "sh", args...).CombinedOutput()

	return out, err
}

// reconstructOnHost runs the recovery script on the host (--combine-image mode) into a fresh output
// dir and returns it. Separate from the boot step so callers can probe the re-restore guard while the
// output is still test-owned (the entrypoint chowns PGDATA to the postgres uid once it boots). The
// output dir's name holds a space, as a user's might, so every path the script passes to
// pg_combinebackup must stay one argument.
func reconstructOnHost(
	t *testing.T,
	router *gin.Engine,
	image, bundle string,
	targetTime *time.Time,
) string {
	t.Helper()

	script := fetchRecoveryScript(t, router)
	outDir := filepath.Join(t.TempDir(), "restored cluster")

	out, err := runRecoveryScriptOnHost(t, script, bundle, outDir, image, targetTime)
	require.NoError(t, err, "recovery script must succeed into an empty dir:\n%s", out)

	return outDir
}

// bootBoundCluster serves a reconstructed output dir through the postgres image's own entrypoint with
// the output bind-mounted at the image VOLUME - the docker-compose path the in-container pg_ctl tests
// bypass.
func bootBoundCluster(t *testing.T, image, outDir string) containers.Endpoint {
	t.Helper()

	return containers.StartPostgresWithBoundDataDir(t, image, hostVolumeDirForMount(image, outDir))
}

// requireRecoveryRefusesReRestore asserts the guard refuses a second restore into an output dir whose
// cluster path the postgres entrypoint already initialized - the empty-DB trap that motivated it.
func requireRecoveryRefusesReRestore(t *testing.T, router *gin.Engine, image, bundle, outDir string) {
	t.Helper()

	script := fetchRecoveryScript(t, router)

	out, err := runRecoveryScriptOnHost(t, script, bundle, outDir, image, nil)
	require.Error(t, err, "re-restoring into an initialized cluster dir must fail; output:\n%s", out)
	require.Contains(t, string(out), "already holds a PostgreSQL cluster",
		"the guard must name the populated-cluster refusal; output:\n%s", out)
}

// requireRecoveryRefusesMisaimedDir asserts the pre-flight guard refuses an output dir that already
// looks like a live PostgreSQL data/install directory (the user aimed the restore at the wrong place).
// It needs no bundle: the check runs before any download.
func requireRecoveryRefusesMisaimedDir(t *testing.T, router *gin.Engine, image string) {
	t.Helper()

	outDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "pg_hba.conf"), []byte("x\n"), 0o600))

	script := fetchRecoveryScript(t, router)
	missingBundle := filepath.Join(t.TempDir(), "missing.tar")

	out, err := runRecoveryScriptOnHost(t, script, missingBundle, outDir, image, nil)
	require.Error(t, err, "restoring into a live data/install dir must fail; output:\n%s", out)
	require.Contains(t, string(out), "looks like a PostgreSQL data or install directory",
		"the guard must explain the misaimed-directory refusal; output:\n%s", out)
}

// newPhysicalTestRouter wires the physical controller's public (restore-stream)
// and protected routes plus the supporting controllers, mirroring production.
func newPhysicalTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	backups_controllers_physical.GetPhysicalBackupController().RegisterPublicRoutes(v1)

	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
		databases.GetDatabaseController().RegisterRoutes(routerGroup)
		backups_config_physical.GetBackupConfigController().RegisterRoutes(routerGroup)
		backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(routerGroup)
	}

	storages.SetupDependencies()
	databases.SetupDependencies()
	notifiers.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	return router
}

// cronInterval is a once-a-year cron used for both cadences so the scheduler
// never auto-fires on its own clock: the only FULL is the bootstrap one (no prior
// full ⇒ due immediately), and incrementals come solely from the trigger
// endpoint. Cron also satisfies the "incremental strictly more frequent than
// full" config rule trivially.
func cronInterval() intervals.Interval {
	expr := "0 0 1 1 *"

	return intervals.Interval{Type: intervals.IntervalCron, CronExpression: &expr}
}

// enablePhysicalBackupsViaAPI turns on backups for the fixture's database through
// the config endpoint. isWalStream must match the DB's BackupType: the WAL-stream
// config requires a positive lag threshold, the plain incremental config requires
// zero.
func enablePhysicalBackupsViaAPI(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	isWalStream bool,
) {
	t.Helper()

	cfg := backups_config_physical.PhysicalBackupConfig{
		DatabaseID:                fixture.DB.ID,
		IsBackupsEnabled:          true,
		FullBackupInterval:        cronInterval(),
		IncrementalBackupInterval: cronInterval(),
		Retention:                 backups_config_physical.RetentionChains,
		ChainsRetention:           backups_config_physical.ChainsRetention{Count: 50},
		Encryption:                backups_core_enums.BackupEncryptionNone,
		StorageID:                 &fixture.Storage.ID,
		Storage:                   fixture.Storage,
	}

	if isWalStream {
		cfg.WalLagThresholdBytes = 64 * 1024 * 1024
	}

	test_utils.MakePostRequest(t, router, "/api/v1/backup-configs/physical/save",
		"Bearer "+fixture.Owner.Token, cfg, http.StatusOK)
}

func triggerIncrementalViaAPI(t *testing.T, router *gin.Engine, fixture *postgresql_executor.PhysicalDBFixture) {
	t.Helper()

	test_utils.MakePostRequest(t, router,
		"/api/v1/backups/physical/database/"+fixture.DB.ID.String()+"/trigger",
		"Bearer "+fixture.Owner.Token,
		backups_dto_physical.TriggerBackupRequest{Type: backups_dto_physical.TriggerBackupTypeIncremental},
		http.StatusAccepted)
}

// useReplicationOnlyUserViaAPI provisions a fresh LOGIN+REPLICATION role on the
// source through the public API and switches the database's stored backup
// credentials to it, so every subsequent FULL / incremental / WAL operation runs
// through a least-privilege user instead of the superuser the fixture is seeded
// with. The backuper re-fetches the database at execution time, so the switch is
// persisted via the update endpoint — mutating the in-memory fixture alone would
// not reach the running backup.
func useReplicationOnlyUserViaAPI(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
) {
	t.Helper()

	physical := fixture.DB.PostgresqlPhysical

	// The provisioning call connects to the source as the current admin user to run
	// CREATE ROLE; the database is identified by ID, the connection block carries the
	// superuser the source container was started with.
	provisionRequest := databases.Database{
		ID:          fixture.DB.ID,
		WorkspaceID: fixture.DB.WorkspaceID,
		Name:        fixture.DB.Name,
		Type:        databases.DatabaseTypePostgresPhysical,
		Notifiers:   []notifiers.Notifier{*fixture.Notifier},
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    physical.Version,
			Host:       physical.Host,
			Port:       physical.Port,
			Username:   restoredPgUser,
			Password:   restoredPgPassword,
			BackupType: physical.BackupType,
		},
	}

	var provisioned databases.CreateReadOnlyUserResponse
	test_utils.MakePostRequestAndUnmarshal(t, router,
		"/api/v1/databases/create-replication-only-user",
		"Bearer "+fixture.Owner.Token, provisionRequest, http.StatusOK, &provisioned)
	require.NotEmpty(t, provisioned.Username)
	require.NotEmpty(t, provisioned.Password)

	switchRequest := provisionRequest
	switchRequest.PostgresqlPhysical = &postgresql_physical.PostgresqlPhysicalDatabase{
		Version:    physical.Version,
		Host:       physical.Host,
		Port:       physical.Port,
		Username:   provisioned.Username,
		Password:   provisioned.Password,
		BackupType: physical.BackupType,
	}
	test_utils.MakePostRequest(t, router, "/api/v1/databases/update",
		"Bearer "+fixture.Owner.Token, switchRequest, http.StatusOK)

	// Mirror the switch onto the in-memory fixture so direct source connections in
	// the test (slot inspection) use the same identity the backups now run as.
	physical.Username = provisioned.Username
	physical.Password = provisioned.Password

	shouldSuggestReplicationOnlyUser, excessivePrivileges, err := physical.ShouldSuggestReplicationOnlyUser(
		t.Context(), logger.GetLogger(), encryption.GetFieldEncryptor())
	require.NoError(t, err)
	require.False(t, shouldSuggestReplicationOnlyUser,
		"backups must run through a replication-only user; excessive privileges: %v", excessivePrivileges)
	require.Empty(t, excessivePrivileges)
}

// deleteDatabaseViaAPI removes the fixture's database through the public DELETE
// endpoint, which fires the OnBeforeDatabaseRemove listeners (slot + streamer
// cleanup). The fixture's own t.Cleanup re-deletes at test end, but DeleteForTest
// is idempotent (a missing row is a no-op and the listeners tolerate it), so no
// extra guard is needed here.
func deleteDatabaseViaAPI(t *testing.T, router *gin.Engine, fixture *postgresql_executor.PhysicalDBFixture) {
	t.Helper()

	test_utils.MakeDeleteRequest(t, router,
		"/api/v1/databases/"+fixture.DB.ID.String(),
		"Bearer "+fixture.Owner.Token, http.StatusNoContent)
}

// waitForSlotPresent polls pg_replication_slots until slotName appears — the WAL
// stream supervisor creates the persistent slot a tick or two after WAL-stream
// backups are enabled — failing the test if it never does.
func waitForSlotPresent(t *testing.T, conn *pgx.Conn, slotName string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		if postgresql_executor.SlotExists(t, conn, slotName) {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("replication slot %q never appeared within %s", slotName, timeout)
}

// requireDatabaseSlotsGone polls until neither of the database's source-side slots
// remains — the persistent WAL streamer slot (databasus_slot_*) and any transient
// per-backup slot (databasus_basebackup_*) — proving the deleted database left
// nothing pinning WAL. Scoped to this database's own slot names. It polls because a
// bootstrap FULL can briefly hold its per-backup slot and the streamer slot drops
// only once its receiver has fully detached.
func requireDatabaseSlotsGone(
	t *testing.T,
	conn *pgx.Conn,
	fixture *postgresql_executor.PhysicalDBFixture,
	timeout time.Duration,
) {
	t.Helper()

	streamerSlot := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	backupSlot := postgresql_executor.SlotName(fixture.DB.PostgresqlPhysical.ID)

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		if !postgresql_executor.SlotExists(t, conn, streamerSlot) &&
			!postgresql_executor.SlotExists(t, conn, backupSlot) {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf(
		"database %s still has a replication slot pinning WAL %s after deletion "+
			"(streamer %q present=%v, per-backup %q present=%v)",
		fixture.DB.ID, timeout,
		streamerSlot, postgresql_executor.SlotExists(t, conn, streamerSlot),
		backupSlot, postgresql_executor.SlotExists(t, conn, backupSlot),
	)
}

// listChainBackups returns the database's FULL and incremental rows (the chain
// backups, excluding committed WAL) in chronological order, the oldest — the
// bootstrap FULL — first and the newest at the tip. The flat list endpoint
// interleaves WAL rows newest-first and paginates, so a single page can be all
// WAL and bury the chain; the type filter the API now exposes fetches FULL and
// INCREMENTAL directly, immune to WAL volume. These tests build exactly one
// chain (backups enabled once ⇒ one bootstrap FULL; the once-a-year cron never
// fires a second), so every non-WAL row belongs to that single chain.
func listChainBackups(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
) []backups_dto_physical.PhysicalBackupListItem {
	t.Helper()

	fullRows := listBackupsByType(t, router, fixture, physical_enums.PhysicalBackupTypeFull)
	incrRows := listBackupsByType(t, router, fixture, physical_enums.PhysicalBackupTypeIncremental)

	chain := slices.Concat(fullRows, incrRows)
	slices.SortFunc(chain, func(a, b backups_dto_physical.PhysicalBackupListItem) int {
		if byTime := a.CreatedAt.Compare(b.CreatedAt); byTime != 0 {
			return byTime
		}

		return cmp.Compare(a.ID.String(), b.ID.String())
	})

	return chain
}

// listBackupsByType pulls every backup of one type for the database. The page
// limit is set well above any chain length these tests produce so a single call
// returns them all.
func listBackupsByType(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	backupType physical_enums.PhysicalBackupType,
) []backups_dto_physical.PhysicalBackupListItem {
	t.Helper()

	var response backups_dto_physical.GetPhysicalBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backups/physical/database/"+fixture.DB.ID.String()+"/backups?limit=1000&type="+string(backupType),
		"Bearer "+fixture.Owner.Token, http.StatusOK, &response)

	return response.Backups
}

// waitForChainBackups polls the flat backup list until the database's chain holds
// a COMPLETED FULL plus wantIncrementals COMPLETED incrementals, failing fast if
// any backup reaches ERROR or CHAIN_BROKEN. Returns the matched chain (oldest
// first) so the caller can read LSNs from the tip.
func waitForChainBackups(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	wantIncrementals int,
	timeout time.Duration,
) []backups_dto_physical.PhysicalBackupListItem {
	t.Helper()

	completed := string(physical_enums.PhysicalBackupStatusCompleted)
	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		chain := listChainBackups(t, router, fixture)

		failFastOnTerminalBackup(t, chain)

		fullCompleted := false
		completedIncrementals := 0

		for _, backup := range chain {
			if backup.Status != completed {
				continue
			}

			if backup.Type == physical_enums.PhysicalBackupTypeFull {
				fullCompleted = true
			} else {
				completedIncrementals++
			}
		}

		if fullCompleted && completedIncrementals == wantIncrementals {
			return chain
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("chain never reached 1 COMPLETED full + %d COMPLETED incrementals within %s",
		wantIncrementals, timeout)

	return nil
}

// failFastOnTerminalBackup aborts the wait the moment any chain row is ERROR or
// CHAIN_BROKEN, turning a 3-minute timeout into an immediate, labelled failure.
// A CHAIN_BROKEN incremental still appears as an INCREMENTAL row in the flat
// list, so it is caught here.
func failFastOnTerminalBackup(t *testing.T, chain []backups_dto_physical.PhysicalBackupListItem) {
	t.Helper()

	for _, backup := range chain {
		if backup.Status == string(physical_enums.PhysicalBackupStatusError) ||
			backup.Status == string(physical_enums.PhysicalBackupStatusChainBroken) {
			t.Fatalf("backup %s (%s) reached terminal failure status %s",
				backup.ID, backup.Type, backup.Status)
		}
	}
}

// rootFullBackupID returns the id of the chain's FULL backup — the key the
// WAL-gap and restore-set resolvers identify a chain by. The FULL's own id is
// the chain's root full id.
func rootFullBackupID(t *testing.T, chain []backups_dto_physical.PhysicalBackupListItem) uuid.UUID {
	t.Helper()

	for _, backup := range chain {
		if backup.Type == physical_enums.PhysicalBackupTypeFull {
			return backup.ID
		}
	}

	t.Fatalf("chain has no FULL backup to derive the root full id from")

	return uuid.Nil
}

// chainTipStopLSN is the stop_lsn of the chain's newest backup — the point the
// next incremental's WAL summaries must cover before it can be built.
func chainTipStopLSN(t *testing.T, chain []backups_dto_physical.PhysicalBackupListItem) walmath.LSN {
	t.Helper()

	require.NotEmpty(t, chain, "chain must hold at least the FULL")

	tip := chain[len(chain)-1]

	return parseLSN(t, tip.StopLSN)
}

// parseLSN parses a textual LSN carried by a backup list item, failing the test
// on a malformed value.
func parseLSN(t *testing.T, text string) walmath.LSN {
	t.Helper()

	lsn, err := walmath.ParseLSN(text)
	require.NoError(t, err)

	return lsn
}

// buildIncrementalViaAPI drives one incremental end to end through the HTTP API:
// it crosses a WAL segment boundary and waits for summaries past parentStopLSN
// (pg_basebackup --incremental needs them), triggers the incremental, and waits
// for the chain to show wantIncrementalsAfter completed incrementals. Returns the
// updated chain.
func buildIncrementalViaAPI(
	t *testing.T,
	ctx context.Context,
	router *gin.Engine,
	conn *pgx.Conn,
	fixture *postgresql_executor.PhysicalDBFixture,
	parentStopLSN walmath.LSN,
	wantIncrementalsAfter int,
) []backups_dto_physical.PhysicalBackupListItem {
	t.Helper()

	_, err := postgresql_executor.GenerateWalActivity(ctx, conn, 32*1024*1024)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	_, err = conn.Exec(ctx, "SELECT pg_switch_wal()")
	require.NoError(t, err)

	require.NoError(t, postgresql_executor.WaitForWalSummaries(ctx, conn, parentStopLSN, 2*time.Minute))

	triggerIncrementalViaAPI(t, router, fixture)

	return waitForChainBackups(t, router, fixture, wantIncrementalsAfter, 3*time.Minute)
}

// requestRestoreTokenViaAPI issues a restore-stream token for the given PITR
// target (nil ⇒ latest) and returns it.
func requestRestoreTokenViaAPI(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	targetTime *time.Time,
) string {
	t.Helper()

	var response backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, router,
		"/api/v1/backups/physical/database/"+fixture.DB.ID.String()+"/restore-token",
		"Bearer "+fixture.Owner.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{TargetTime: targetTime},
		http.StatusOK, &response)

	require.NotEmpty(t, response.Token)

	return response.Token
}

// requestRestoreTokenExpectingStatus posts a restore-token request expecting a
// specific non-200 status (used by the WAL-gap test, which expects 422) and
// returns the response for body assertions.
func requestRestoreTokenExpectingStatus(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	targetTime *time.Time,
	expectedStatus int,
) *test_utils.TestResponse {
	t.Helper()

	return test_utils.MakePostRequest(t, router,
		"/api/v1/backups/physical/database/"+fixture.DB.ID.String()+"/restore-token",
		"Bearer "+fixture.Owner.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{TargetTime: targetTime},
		expectedStatus)
}

// downloadRestoreBundleViaAPI requests a restore token then streams the bundle tar
// from the public restore-stream endpoint to a host temp file, returning its path.
func downloadRestoreBundleViaAPI(
	t *testing.T,
	router *gin.Engine,
	fixture *postgresql_executor.PhysicalDBFixture,
	targetTime *time.Time,
) string {
	t.Helper()

	token := requestRestoreTokenViaAPI(t, router, fixture, targetTime)

	recorder := workspaces_testing.MakeAPIRequest(router, "GET",
		"/api/v1/backups/physical/restore-stream?token="+token, "", nil)
	require.Equal(t, http.StatusOK, recorder.Code,
		"restore-stream must return 200; body: %s", recorder.Body.String())

	hostPath := filepath.Join(t.TempDir(), "restore.tar")
	require.NoError(t, os.WriteFile(hostPath, recorder.Body.Bytes(), 0o600))

	return hostPath
}

// committedWalSegmentsInOrder reads the database's committed (uploaded) WAL
// segments on one timeline from the catalog, ordered by start_lsn ascending. The
// backup list API groups segments into ranges, so it cannot hand out a single
// segment to delete.
func committedWalSegmentsInOrder(
	t *testing.T,
	fixture *postgresql_executor.PhysicalDBFixture,
	timelineID int,
) []*physical_models.PhysicalWalSegment {
	t.Helper()

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		fixture.DB.ID, timelineID, walmath.LSN(0), chain_view.LSNMax,
	)
	require.NoError(t, err)

	return slices.DeleteFunc(segments, func(segment *physical_models.PhysicalWalSegment) bool {
		return segment.FileName == nil
	})
}

// streamPostFullSegments forces WAL rotations until at least minCount committed
// segments lie past the chain tip's stop_lsn on its timeline (i.e. in the
// replayable range after the tip), returning them ordered by start_lsn.
// pg_receivewal writes full, contiguous segments, so the run has no gap until a
// caller deletes one.
func streamPostFullSegments(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	fixture *postgresql_executor.PhysicalDBFixture,
	chain []backups_dto_physical.PhysicalBackupListItem,
	minCount int,
	timeout time.Duration,
) []*physical_models.PhysicalWalSegment {
	t.Helper()

	tipStopLSN := chainTipStopLSN(t, chain)
	tipTimelineID := chain[len(chain)-1].TimelineID

	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		_, err := postgresql_executor.ForceWalRotation(ctx, conn)
		require.NoError(t, err)

		var postTipSegments []*physical_models.PhysicalWalSegment
		for _, segment := range committedWalSegmentsInOrder(t, fixture, tipTimelineID) {
			if segment.EndLSN > tipStopLSN {
				postTipSegments = append(postTipSegments, segment)
			}
		}

		if len(postTipSegments) >= minCount {
			return postTipSegments
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("fewer than %d committed post-FULL WAL segments archived within %s", minCount, timeout)

	return nil
}

// createMarkerTable (re)creates the restore_marker table on the source DB, with
// cleanup. Each restore test seeds phase rows here and asserts which survived on
// the restored cluster.
func createMarkerTable(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS restore_marker`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DROP TABLE IF EXISTS restore_marker`)
	})

	_, err = conn.Exec(
		ctx,
		`CREATE TABLE restore_marker (phase TEXT PRIMARY KEY, payload TEXT NOT NULL)`)
	require.NoError(t, err)
}

func insertMarker(t *testing.T, ctx context.Context, conn *pgx.Conn, phase, payload string) {
	t.Helper()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO restore_marker (phase, payload) VALUES ($1, $2)`, phase, payload)
	require.NoError(t, err)
}

// waitForReplayableThroughLSN blocks until the resolver's latest restore set has a
// contiguous WAL run reaching throughLSN, i.e. the streamed WAL covering the PITR
// target is gap-free and shippable. It drives the same resolver the restore stream
// uses, so it waits on exactly the condition the stream needs.
func waitForReplayableThroughLSN(t *testing.T, databaseID uuid.UUID, throughLSN walmath.LSN, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)

	var lastReachable walmath.LSN
	for time.Now().UTC().Before(deadline) {
		set, err := chain_view.GetChainViewService().ResolveRestoreSet(databaseID, nil)
		require.NoError(t, err)

		lastReachable = set.MaxReplayableLSN
		if set.MaxReplayableLSN >= throughLSN {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("contiguous replayable WAL never reached LSN %s within %s (latest reachable %s)",
		throughLSN.String(), timeout, lastReachable.String())
}

// Reaching the target's LSN is not enough on its own: the resolver compares the
// target against the receive time of the last segment in the contiguous run, so a
// run that already extends past the target by LSN can still end at a segment
// archived before it.
func waitForTargetTimeReplayable(t *testing.T, databaseID uuid.UUID, targetTime time.Time, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)

	var lastErr error
	for time.Now().UTC().Before(deadline) {
		_, lastErr = chain_view.GetChainViewService().ResolveRestoreSet(databaseID, &targetTime)
		if lastErr == nil {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("target time %s never became restorable within %s: %v", targetTime, timeout, lastErr)
}

// seedChainAndStreamPastTarget seeds the marker table, builds FULL -> INCR -> INCR over the API, then
// streams WAL past a captured target time (with a post-target row that must be dropped). It returns
// the PITR target and the phases that must survive recovery to it - shared by the in-container and the
// entrypoint PITR tests so both drive the identical source history.
func seedChainAndStreamPastTarget(
	t *testing.T,
	ctx context.Context,
	router *gin.Engine,
	sourceConn *pgx.Conn,
	fixture *postgresql_executor.PhysicalDBFixture,
) (time.Time, []string) {
	t.Helper()

	createMarkerTable(t, ctx, sourceConn)
	insertMarker(t, ctx, sourceConn, "before-full", "row-in-base-backup")

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	chain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	insertMarker(t, ctx, sourceConn, "after-full", "row-between-full-and-incr1")
	chain = buildIncrementalViaAPI(t, ctx, router, sourceConn, fixture, chainTipStopLSN(t, chain), 1)

	insertMarker(t, ctx, sourceConn, "after-incr1", "row-between-incr1-and-incr2")
	buildIncrementalViaAPI(t, ctx, router, sourceConn, fixture, chainTipStopLSN(t, chain), 2)

	// 'before-target' is committed after the last INCR; PITR must replay it from streamed WAL. Fill
	// the segment with natural WAL so it rotates and archives (pg_switch_wal would leave a partial
	// segment the resolver treats as a gap).
	insertMarker(t, ctx, sourceConn, "before-target", "row-replayed-up-to-target")

	_, err := postgresql_executor.GenerateWalActivity(ctx, sourceConn, 64*1024*1024)
	require.NoError(t, err)

	// A margin wider than the whole-second recovery_target_time truncation keeps the cut unambiguous.
	time.Sleep(2 * time.Second)
	targetTime := time.Now().UTC()
	time.Sleep(2 * time.Second)

	insertMarker(t, ctx, sourceConn, "after-target", "row-after-target-must-be-absent")

	var afterTargetLSN walmath.LSN
	require.NoError(t, sourceConn.QueryRow(ctx, `SELECT pg_current_wal_lsn()::text`).Scan(&afterTargetLSN))

	_, err = postgresql_executor.GenerateWalActivity(ctx, sourceConn, 64*1024*1024)
	require.NoError(t, err)

	waitForReplayableThroughLSN(t, fixture.DB.ID, afterTargetLSN, 90*time.Second)

	return targetTime, []string{"before-full", "after-full", "after-incr1", "before-target"}
}

// reconstructCluster rebuilds PGDATA the way a user does: it runs the
// server-shipped recovery script (fetched from the public endpoint) inside the
// restore target. The script extracts the bundle, folds the incremental chain with
// pg_combinebackup, decompresses WAL on the host up front and arms PITR from the
// --target-time argument (nil ⇒ latest) — its restore_command is a plain cp, so the
// started cluster needs no zstd. The test drives the real restore path end to end,
// not a reimplementation.
func reconstructCluster(
	t *testing.T,
	target containers.RestoreTarget,
	router *gin.Engine,
	image string,
	hostBundle string,
	targetTime *time.Time,
) {
	t.Helper()

	hostScript := fetchRecoveryScript(t, router)

	// docker cp (testcontainers CopyFileToContainer) cannot write into a tmpfs mount,
	// so stage the bundle and script under /tmp; the script runs in-container and
	// writes its output into the /restore tmpfs itself, which is fine.
	stagedBundle := "/tmp/restore.tar"
	stagedScript := "/tmp/databasus-recovery.sh"

	target.CopyFileToContainer(t, hostBundle, stagedBundle, 0o644)
	target.CopyFileToContainer(t, hostScript, stagedScript, 0o755)

	if targetTime != nil {
		formattedTarget := targetTime.UTC().Format("2006-01-02 15:04:05-07:00")
		target.Exec(t, "sh", stagedScript, "--target-time", formattedTarget, stagedBundle, restoreWorkDir)
	} else {
		target.Exec(t, "sh", stagedScript, stagedBundle, restoreWorkDir)
	}

	// pg_ctl runs as the postgres user, so the reconstructed data dir and the WAL
	// the restore_command reads must be owned by it.
	target.Exec(t, "chown", "-R", "postgres:postgres", restoreWorkDir)
	target.Exec(t, "chmod", "0700", restoredClusterDir(image))
}

// fetchRecoveryScript pulls the restore helper from the public endpoint — the same
// bytes the UI tells the user to curl — and writes it to a host temp file.
func fetchRecoveryScript(t *testing.T, router *gin.Engine) string {
	t.Helper()

	recorder := workspaces_testing.MakeAPIRequest(router, "GET",
		"/api/v1/backups/physical/recovery-script", "", nil)
	require.Equal(t, http.StatusOK, recorder.Code,
		"recovery-script must return 200; body: %s", recorder.Body.String())
	require.NotEmpty(t, recorder.Body.Bytes(), "recovery-script must not be empty")

	hostPath := filepath.Join(t.TempDir(), "databasus-recovery.sh")
	require.NoError(t, os.WriteFile(hostPath, recorder.Body.Bytes(), 0o700))

	return hostPath
}

// requireRestoredClusterNeedsNoZstd proves the runtime image needs no zstd: the
// recovery script must have inflated all WAL on the host (no *.zst left in the
// plaintext archive) and wired a plain-cp restore_command, and the cluster must
// then start and replay with the zstd CLI removed. Decompression having already
// happened on the host (during reconstructCluster) is what makes this safe — only
// the runtime dependency is being stripped.
func requireRestoredClusterNeedsNoZstd(t *testing.T, target containers.RestoreTarget, image string) {
	t.Helper()

	clusterDir := restoredClusterDir(image)

	autoConf := string(target.Exec(t, "cat", clusterDir+"/postgresql.auto.conf"))
	require.Contains(t, autoConf, "restore_command = 'cp ",
		"restore_command must be a plain cp so the runtime cluster never calls zstd")
	require.NotContains(t, autoConf, "zstd",
		"the recovery config must not reference zstd")

	leftoverCompressed := strings.TrimSpace(string(
		target.Exec(t, "sh", "-c", "ls "+clusterDir+"/databasus_wal_restore/*.zst 2>/dev/null | wc -l")))
	require.Equal(t, "0", leftoverCompressed,
		"the WAL archive the cluster reads must hold no still-compressed segments")

	// Strip the zstd CLI; libzstd (the server's link-time dep) stays, so PostgreSQL
	// still starts — only an on-demand restore_command would now fail.
	target.Exec(t, "sh", "-c",
		`p=$(command -v zstd); [ -n "$p" ] && rm -f "$p"; ! command -v zstd >/dev/null`)
}

func startRestoredCluster(t *testing.T, target containers.RestoreTarget, image string) {
	t.Helper()

	clusterDir := restoredClusterDir(image)

	target.Exec(t, "sh", "-c",
		"touch "+restoreWorkDir+"/pg.log && chown postgres:postgres "+restoreWorkDir+"/pg.log")

	// Surface the server log on failure — recovery errors are otherwise invisible
	// behind pg_ctl's generic "could not start server".
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		t.Logf("=== restored pg.log ===\n%s", target.ExecBestEffort("", "cat", restoreWorkDir+"/pg.log"))
	})

	target.ExecAs(t, "postgres",
		"pg_ctl", "-D", clusterDir, "-l", restoreWorkDir+"/pg.log", "-w", "start")

	t.Cleanup(func() {
		target.ExecBestEffort("postgres",
			"pg_ctl", "-D", clusterDir, "-m", "immediate", "stop")
	})

	waitForRecoveryToFinish(t, target)
}

// pg_ctl -w returns as soon as the cluster accepts connections, which during recovery means
// read-only and mid-replay: a query issued then sees whichever prefix of the WAL has been applied so
// far. Recovery with a target promotes when it reaches it, so leaving recovery is the real signal
// that the restored data is complete.
func waitForRecoveryToFinish(t *testing.T, target containers.RestoreTarget) {
	t.Helper()

	deadline := time.Now().UTC().Add(recoveryCompletionTimeout)
	for time.Now().UTC().Before(deadline) {
		recoveryState := target.ExecBestEffort("postgres",
			"psql", "-U", restoredPgUser, "-d", restoredPgDatabase, "-tAc", "SELECT pg_is_in_recovery()")
		if strings.TrimSpace(string(recoveryState)) == "f" {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("the restored cluster was still replaying WAL after %s", recoveryCompletionTimeout)
}

func queryRestoredMarkerRows(t *testing.T, target containers.RestoreTarget) []string {
	t.Helper()

	return queryMarkerRowsAt(t, target.Host(), target.MappedPort())
}

// queryMarkerRowsAt reads the restore_marker phases from a restored cluster at host:port - whether it
// was started by pg_ctl in the restore target or by the postgres entrypoint on a bound volume.
func queryMarkerRowsAt(t *testing.T, host string, port int) []string {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, restoredPgUser, restoredPgPassword, restoredPgDatabase)

	conn := connectWithRetry(t, dsn, 60*time.Second)
	defer func() { _ = conn.Close(t.Context()) }()

	rows, err := conn.Query(t.Context(), `SELECT phase FROM restore_marker ORDER BY phase`)
	require.NoError(t, err)
	defer rows.Close()

	var phases []string
	for rows.Next() {
		var phase string
		require.NoError(t, rows.Scan(&phase))
		phases = append(phases, phase)
	}
	require.NoError(t, rows.Err())

	return phases
}

func openSourceTestDBConn(t *testing.T, fixture *postgresql_executor.PhysicalDBFixture) *pgx.Conn {
	t.Helper()

	return openConnAt(t, fixture.DB.PostgresqlPhysical.Host, fixture.DB.PostgresqlPhysical.Port)
}

// openConnAt opens a superuser connection (the seeded testuser) to a PostgreSQL endpoint, closed on
// cleanup. The cross-timeline test reaches the primary through this, since the fixture's own
// connection targets the standby Databasus backs up.
func openConnAt(t *testing.T, host string, port int) *pgx.Conn {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, restoredPgUser, restoredPgPassword, restoredPgDatabase)

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

// waitForMarkerOnStandby blocks until phase is visible on the standby, i.e. streaming replication
// has carried the primary's write across. The FULL is taken from the standby, so a row must land
// there before it can appear in the base backup. The marker table itself replicates via WAL too, so
// a query before it exists errors with undefined_table — treated here as "not yet replicated".
func waitForMarkerOnStandby(
	t *testing.T,
	ctx context.Context,
	standbyConn *pgx.Conn,
	phase string,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		var exists bool
		if err := standbyConn.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM restore_marker WHERE phase = $1)`, phase,
		).Scan(&exists); err == nil && exists {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("marker %q never replicated to the standby within %s", phase, timeout)
}

// PromoteStandby promotes a streaming standby to a primary on a fresh timeline and waits until it
// has left recovery. pg_promote keeps the postmaster up, so a pg_receivewal already streaming from
// this node follows the switch and fetches the new timeline's .history file — the path issue #643
// regressed. Callers can write on the new timeline as soon as it returns.
func PromoteStandby(t *testing.T, ctx context.Context, standbyConn *pgx.Conn) {
	t.Helper()

	var promoted bool
	require.NoError(t, standbyConn.QueryRow(
		ctx,
		`SELECT pg_promote(wait => true, wait_seconds => 60)`).Scan(&promoted))
	require.True(t, promoted, "pg_promote must finish promotion within its wait window")

	deadline := time.Now().UTC().Add(30 * time.Second)
	for time.Now().UTC().Before(deadline) {
		var isInRecovery bool
		require.NoError(t, standbyConn.QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&isInRecovery))
		if !isInRecovery {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatal("standby still in recovery 30s after pg_promote reported success")
}

// waitForTimelineHistoryOnParent blocks until the WAL stream supervisor has cataloged the timeline's
// .history row, then asserts it is keyed on the parent databases.id and not the
// postgresql_physical_databases PK. Pre-fix (issue #643) the insert violated
// fk_physical_wal_history_files_database_id and the row never appeared, so this both proves the
// failover path runs end to end and guards the regression.
func waitForTimelineHistoryOnParent(
	t *testing.T,
	fixture *postgresql_executor.PhysicalDBFixture,
	timelineID int,
	timeout time.Duration,
) {
	t.Helper()

	historyRepo := physical_repositories.GetWalHistoryRepository()
	parentDatabaseID := fixture.DB.ID
	physicalPK := fixture.DB.PostgresqlPhysical.ID

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		byParent, err := historyRepo.FindByDatabaseTimeline(parentDatabaseID, timelineID)
		require.NoError(t, err)

		if byParent != nil {
			byPhysicalPK, err := historyRepo.FindByDatabaseTimeline(physicalPK, timelineID)
			require.NoError(t, err)
			require.Nil(t, byPhysicalPK,
				"timeline-%d history must be keyed on the parent databases.id, not the physical PK", timelineID)

			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("WAL stream supervisor never cataloged the timeline-%d history row on the parent databases.id "+
		"within %s (issue #643: keying on the physical PK fails fk_physical_wal_history_files_database_id)",
		timelineID, timeout)
}

// Re-anchoring crosses the incremental executor, the scheduler and the FULL executor,
// and each of them can stop it for its own reason, so the timeout prints the catalog
// rows it last read rather than only the verdict.
func waitForReanchoredFull(
	t *testing.T,
	fixture *postgresql_executor.PhysicalDBFixture,
	staleFullID uuid.UUID,
	newTimelineID int,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)

	var state reanchorCatalogState

	for time.Now().UTC().Before(deadline) {
		state = readReanchorCatalogState(t, fixture, staleFullID)

		hasReplacementFull := len(state.CompletedFulls) >= 2 &&
			state.CompletedFulls[0].TimelineID == newTimelineID

		if hasTimelineRefusal(state.StaleChainIncrementals) && hasReplacementFull {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("promotion did not re-anchor within %s: the stale chain must refuse an incremental with %s, "+
		"and a COMPLETED FULL on timeline %d must follow it\n"+
		"  stale-chain incrementals: %s\n"+
		"  completed fulls:          %s\n"+
		"  newest full (any status): %s",
		timeout, physical_enums.PhysicalBackupErrorTimelineSwitchDetected, newTimelineID,
		describeIncrementalOutcomes(state.StaleChainIncrementals),
		describeFullOutcomes(state.CompletedFulls),
		describeFullOutcome(state.NewestFullAnyStatus))
}

type reanchorCatalogState struct {
	StaleChainIncrementals []*physical_models.PhysicalIncrementalBackup
	CompletedFulls         []*physical_models.PhysicalFullBackup
	// A replacement FULL that failed is absent from the completed list.
	NewestFullAnyStatus *physical_models.PhysicalFullBackup
}

// waitForReanchoredFull polls for a typed error reason, which only the repository rows
// carry; the backup list DTO has just the free-text fail message.
func readReanchorCatalogState(
	t *testing.T,
	fixture *postgresql_executor.PhysicalDBFixture,
	staleFullID uuid.UUID,
) reanchorCatalogState {
	t.Helper()

	incrementalRepo := physical_repositories.GetIncrementalBackupRepository()
	fullRepo := physical_repositories.GetFullBackupRepository()

	staleChainIncrementals, err := incrementalRepo.FindAllByRootFull(staleFullID)
	require.NoError(t, err)

	completedFulls, err := fullRepo.FindCompletedNewestFirstByDatabase(fixture.DB.ID)
	require.NoError(t, err)

	newestFullAnyStatus, err := fullRepo.FindLastFullAnyStatusByDatabase(fixture.DB.ID)
	require.NoError(t, err)

	return reanchorCatalogState{
		StaleChainIncrementals: staleChainIncrementals,
		CompletedFulls:         completedFulls,
		NewestFullAnyStatus:    newestFullAnyStatus,
	}
}

func hasTimelineRefusal(incrementals []*physical_models.PhysicalIncrementalBackup) bool {
	return slices.ContainsFunc(
		incrementals,
		func(incremental *physical_models.PhysicalIncrementalBackup) bool {
			return incremental.Status == physical_enums.PhysicalBackupStatusChainBroken &&
				incremental.ErrorReason != nil &&
				*incremental.ErrorReason == physical_enums.PhysicalBackupErrorTimelineSwitchDetected
		},
	)
}

func describeIncrementalOutcomes(incrementals []*physical_models.PhysicalIncrementalBackup) string {
	descriptions := make([]string, 0, len(incrementals))

	for _, incremental := range incrementals {
		descriptions = append(descriptions, fmt.Sprintf("%s %s (%s)",
			incremental.ID, incremental.Status, describeErrorReason(incremental.ErrorReason)))
	}

	return joinOutcomesOrNone(descriptions)
}

func describeFullOutcomes(fulls []*physical_models.PhysicalFullBackup) string {
	descriptions := make([]string, 0, len(fulls))

	for _, full := range fulls {
		descriptions = append(descriptions, describeFullOutcome(full))
	}

	return joinOutcomesOrNone(descriptions)
}

func describeFullOutcome(full *physical_models.PhysicalFullBackup) string {
	if full == nil {
		return "none"
	}

	return fmt.Sprintf("%s %s on TL %d (%s)",
		full.ID, full.Status, full.TimelineID, describeErrorReason(full.ErrorReason))
}

func joinOutcomesOrNone(descriptions []string) string {
	if len(descriptions) == 0 {
		return "none"
	}

	return strings.Join(descriptions, "; ")
}

func describeErrorReason(reason *physical_enums.PhysicalBackupErrorReason) string {
	if reason == nil {
		return "no error reason"
	}

	return string(*reason)
}

func connectWithRetry(t *testing.T, dsn string, timeout time.Duration) *pgx.Conn {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	var lastErr error

	for time.Now().UTC().Before(deadline) {
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)

		conn, err := pgx.Connect(ctx, dsn)
		cancel()

		if err == nil {
			return conn
		}

		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("could not connect to restored PG within %s: %v", timeout, lastErr)

	return nil
}
