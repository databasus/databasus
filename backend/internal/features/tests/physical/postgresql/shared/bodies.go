//go:build testhelpers

package physicaltesting

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/testing/containers"
)

// RunFullOnlyRecoversBaseRows drives the whole happy path through the HTTP control plane: seed a
// row, enable backups (the scheduler bootstraps the FULL), then pull the restore bundle through the
// restore-token + restore-stream endpoints, reconstruct the cluster (pg_combinebackup), and start it.
// The restored cluster must hold the row written before the FULL. Backups run through a
// replication-only user provisioned over the API.
func RunFullOnlyRecoversBaseRows(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(t, version, image, postgresql_physical.BackupTypeFullAndIncremental)
	target := prepareRestoreTarget(t, image)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	createMarkerTable(t, ctx, sourceConn)
	insertMarker(t, ctx, sourceConn, "before-full", "row-before-full")

	enablePhysicalBackupsViaAPI(t, router, fixture, false)
	waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, nil)
	reconstructCluster(t, target, router, image, bundle, nil)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, []string{"before-full"}, restoredPhases,
		"a full-only restore must recover the row written before the FULL")
}

// RunFullPlusTwoIncrementalsRecoversAllRows builds a FULL → INCR → INCR chain entirely over HTTP
// (config-enable for the FULL, the trigger endpoint for each incremental), restores the latest point,
// and asserts every row written before each backup survives — proving the incrementals chain and
// combine correctly.
func RunFullPlusTwoIncrementalsRecoversAllRows(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(t, version, image, postgresql_physical.BackupTypeFullAndIncremental)
	target := prepareRestoreTarget(t, image)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	createMarkerTable(t, ctx, sourceConn)
	insertMarker(t, ctx, sourceConn, "before-full", "row-before-full")

	enablePhysicalBackupsViaAPI(t, router, fixture, false)
	chain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	insertMarker(t, ctx, sourceConn, "after-full", "row-between-full-and-incr1")
	chain = buildIncrementalViaAPI(t, ctx, router, sourceConn, fixture, chainTipStopLSN(t, chain), 1)

	insertMarker(t, ctx, sourceConn, "after-incr1", "row-between-incr1-and-incr2")
	buildIncrementalViaAPI(t, ctx, router, sourceConn, fixture, chainTipStopLSN(t, chain), 2)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, nil)
	reconstructCluster(t, target, router, image, bundle, nil)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t,
		[]string{"before-full", "after-full", "after-incr1"},
		restoredPhases,
		"restoring the latest point must recover rows from the FULL and both incrementals")
}

// RunFullTwoIncrementalsPlusWalRecoversToTarget extends the chain test with point-in-time recovery:
// after FULL → INCR → INCR it streams WAL past a captured target, restores the bundle WITH that
// target time, and asserts the chain rows plus the pre-target row replay while the post-target row is
// dropped — proving full + incrementals + streamed WAL combine and stop at the target. WAL streaming
// is driven purely by the config API: enabling the WAL-stream backup type makes the supervisor claim
// the database and start pg_receivewal. It also strips the zstd CLI before starting the cluster,
// proving the recovery script inflated WAL on the host and the runtime image needs no zstd.
func RunFullTwoIncrementalsPlusWalRecoversToTarget(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	target := prepareRestoreTarget(t, image)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	targetTime, survivingPhases := seedChainAndStreamPastTarget(t, ctx, router, sourceConn, fixture)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, &targetTime)
	reconstructCluster(t, target, router, image, bundle, &targetTime)
	requireRestoredClusterNeedsNoZstd(t, target, image)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, survivingPhases, restoredPhases,
		"PITR must replay the chain plus rows committed at/before the target and drop the row after it")
}

// RunWalStreamCatalogsHistoryOnTimelineSwitch exercises the failover path no other test covers:
// Databasus streams WAL from a standby, the standby is promoted to a new timeline mid-stream, and
// the live WAL stream supervisor must observe the switch and catalog the new timeline's .history
// row. Promotion makes pg_receivewal fetch 00000002.history, driving the supervisor's
// handleHistoryFile -> UploadHistoryFile path — the exact insert that regressed in issue #643:
// keyed on the postgresql_physical_databases PK it violated fk_physical_wal_history_files_database_id
// and aborted the whole WAL stream. The test asserts the history row is cataloged on the parent
// databases.id (and not the physical PK), proving the failover path runs end to end.
//
// It deliberately stops at the catalog assertion rather than running a cross-timeline PITR restore:
// the restore-set resolver loads a chain's WAL on the FULL's timeline only
// (buildChainView -> FindByChainSpan keys on full.TimelineID), so it cannot ship post-fork WAL and a
// cross-timeline restore is unsupported by the current resolver regardless of issue #643. The
// single-timeline PITR restore is already covered by RunFullTwoIncrementalsPlusWalRecoversToTarget.
func RunWalStreamCatalogsHistoryOnTimelineSwitch(t *testing.T, version, image string) {
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)

	// Databasus backs up the STANDBY: pg_basebackup and pg_receivewal run against it, so promoting it
	// makes the live stream follow the timeline switch — the production failover path. Backups run as
	// the seeded superuser (a standby is read-only, so the replication-only-user provisioning the
	// single-node fixtures do would fail its CREATE ROLE here).
	router := newPhysicalTestRouter()
	fixture := postgresql_executor.SetupPhysicalDBForScheduledBackupVersion(
		t, standby.Host, standby.Port, version, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	primaryConn := openConnAt(t, primary.Host, primary.Port)
	standbyConn := openSourceTestDBConn(t, fixture)

	// Timeline 1: seed and drive WAL on the primary, let it replicate to the standby, and back the
	// standby up. Writes must hit the primary — the standby is read-only until promotion.
	createMarkerTable(t, ctx, primaryConn)
	insertMarker(t, ctx, primaryConn, "tl1-row", "row-on-tl1")
	waitForMarkerOnStandby(t, ctx, standbyConn, "tl1-row", 60*time.Second)

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	chain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	// Stream a couple of committed TL1 segments so the slot is actively consuming when we promote.
	streamPostFullSegments(t, ctx, primaryConn, fixture, chain, 2, 90*time.Second)

	// Promote: the standby becomes a primary on timeline 2. pg_receivewal follows the switch and
	// uploads 00000002.history, driving the issue #643 path in the live supervisor.
	PromoteStandby(t, ctx, standbyConn)

	// A write on the new timeline guarantees TL2 WAL, so pg_receivewal advances onto timeline 2 and
	// ships its .history.
	insertMarker(t, ctx, standbyConn, "tl2-row", "row-on-tl2")
	_, err := postgresql_executor.GenerateWalActivity(ctx, standbyConn, 16*1024*1024)
	require.NoError(t, err)

	// The regression guard: the supervisor must catalog the timeline-2 history row on the parent
	// databases.id. Pre-fix the FK violation drops it and the wait fails.
	waitForTimelineHistoryOnParent(t, fixture, 2, 60*time.Second)
}

func RunPromotionReanchorsIncrementalChain(t *testing.T, version, image string) {
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	router := newPhysicalTestRouter()
	fixture := postgresql_executor.SetupPhysicalDBForScheduledBackupVersion(
		t, standby.Host, standby.Port, version, postgresql_physical.BackupTypeFullIncrementalAndWalStream,
	)
	target := prepareRestoreTarget(t, image)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	primaryConn := openConnAt(t, primary.Host, primary.Port)
	standbyConn := openSourceTestDBConn(t, fixture)
	createMarkerTable(t, ctx, primaryConn)
	insertMarker(t, ctx, primaryConn, "before-promotion", "row-before-promotion")
	waitForMarkerOnStandby(t, ctx, standbyConn, "before-promotion", 60*time.Second)

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	initialChain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)
	initialFullID := rootFullBackupID(t, initialChain)

	PromoteStandby(t, ctx, standbyConn)
	insertMarker(t, ctx, standbyConn, "after-promotion", "row-after-promotion")
	_, err := postgresql_executor.GenerateWalActivity(ctx, standbyConn, 16*1024*1024)
	require.NoError(t, err)
	waitForTimelineHistoryOnParent(t, fixture, 2, 60*time.Second)
	triggerIncrementalViaAPI(t, router, fixture)

	waitForReanchoredFull(t, fixture, initialFullID, 2, 4*time.Minute)

	backuping_physical.RunOrphanWalCleanupForTest(t, fixture.DB.ID)
	bundle := downloadRestoreBundleViaAPI(t, router, fixture, nil)
	reconstructCluster(t, target, router, image, bundle, nil)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, []string{"before-promotion", "after-promotion"}, restoredPhases)
}

// RunBootViaEntrypointVolumeMountRecoversBaseRows reproduces the user's docker-compose flow: restore on
// the host in --combine-image mode, then serve the result through the postgres image's own entrypoint
// with the output bind-mounted at the image VOLUME (the volume root on PG 18). The booted cluster must
// hold the base row - the regression the in-container pg_ctl tests miss because they start the cluster
// with pg_ctl -D and so never exercise the entrypoint's PGDATA-layout detection. It then proves the
// guard refuses a second restore onto the now-initialized volume.
func RunBootViaEntrypointVolumeMountRecoversBaseRows(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(t, version, image, postgresql_physical.BackupTypeFullAndIncremental)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	createMarkerTable(t, ctx, sourceConn)
	insertMarker(t, ctx, sourceConn, "before-full", "row-before-full")

	enablePhysicalBackupsViaAPI(t, router, fixture, false)
	waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, nil)
	outDir := reconstructOnHost(t, router, image, bundle, nil)

	// Probe the populated-cluster guard before booting: once the entrypoint runs it chowns PGDATA to
	// the postgres uid, which a host-side re-run could no longer touch.
	requireRecoveryRefusesReRestore(t, router, image, bundle, outDir)

	endpoint := bootBoundCluster(t, image, outDir)

	restoredPhases := queryMarkerRowsAt(t, endpoint.Host, endpoint.Port)
	assert.ElementsMatch(t, []string{"before-full"}, restoredPhases,
		"the entrypoint must serve the restored cluster from the bound volume, not a fresh empty one")
}

// RunPitrBootViaEntrypointVolumeMountRecoversToTarget runs a FULL+INCR+INCR+WAL PITR on the host and
// serves it through the image entrypoint on a bound volume, proving the PGDATA-relative WAL archive
// replays inside a real container - not only under pg_ctl -D. The source runs max_connections=200,
// above the restored container's default 100, so archive recovery would abort with "insufficient
// parameter settings" unless the script pinned the primary's parameters into postgresql.auto.conf -
// this also covers that recovery-parameter auto-config.
func RunPitrBootViaEntrypointVolumeMountRecoversToTarget(t *testing.T, version, image string) {
	const primaryMaxConnections = 200

	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		containers.WithMaxConnections(primaryMaxConnections))

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	targetTime, survivingPhases := seedChainAndStreamPastTarget(t, ctx, router, sourceConn, fixture)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, &targetTime)
	outDir := reconstructOnHost(t, router, image, bundle, &targetTime)

	autoConf, err := os.ReadFile(filepath.Join(hostClusterDir(image, outDir), "postgresql.auto.conf"))
	require.NoError(t, err, "the recovery script must write postgresql.auto.conf")
	require.Contains(t, string(autoConf), "max_connections = 200",
		"the script must pin the primary's max_connections so archive recovery does not abort")

	endpoint := bootBoundCluster(t, image, outDir)

	restoredPhases := queryMarkerRowsAt(t, endpoint.Host, endpoint.Port)
	assert.ElementsMatch(t, survivingPhases, restoredPhases,
		"PITR replayed through the entrypoint must recover rows at/before the target and drop the later one")
}

// RunRejectsMisaimedRestoreTarget asserts the recovery script refuses an output dir that already looks
// like a live PostgreSQL data/install directory, before downloading anything. Version-independent, so
// it boots no source - only the public recovery-script endpoint and a host docker for --combine-image.
func RunRejectsMisaimedRestoreTarget(t *testing.T, _, image string) {
	requireRecoveryRefusesMisaimedDir(t, newPhysicalTestRouter(), image)
}

// RunWhenWalGapBeforeTargetTokenRequestReturns422 proves a WAL gap is refused at token-issue time:
// it streams a contiguous run of segments, punches a hole by deleting a committed middle segment, then
// asks for a restore token targeting a time past the gap and expects HTTP 422. No reconstruction —
// the contract is that an unreachable target never mints a token.
func RunWhenWalGapBeforeTargetTokenRequestReturns422(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	chain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	// Stream a run of post-FULL segments, then drop a committed middle one so a
	// real hole sits in the replayable WAL ahead of the target.
	postFull := streamPostFullSegments(t, ctx, sourceConn, fixture, chain, 3, 90*time.Second)
	removed := postFull[len(postFull)/2]
	require.NoError(t, physical_repositories.GetWalSegmentRepository().DeleteByID(removed.ID))

	gaps := postgresql_executor.WaitForWalGap(t, rootFullBackupID(t, chain), 30*time.Second)
	require.NotEmpty(t, gaps, "deleting a committed middle segment must surface a gap in the chain")

	targetTime := time.Now().UTC()
	response := requestRestoreTokenExpectingStatus(
		t, router, fixture, &targetTime, http.StatusUnprocessableEntity)

	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body, &body))
	assert.Contains(t, body["error"], "wal gap",
		"the gap must be reported so the user never burns a token on an unreachable target")
}

// A FULL's start_lsn sits inside the segment that carries it, so that segment's file-aligned start
// is lower than the FULL's and the retention sweep must still keep it: it holds the bytes replay
// begins from. No incremental is taken on purpose. An incremental moves the replay window past the
// FULL's own boundary segment and hides whether the sweep kept it.
func RunWhenOrphanSweepRunsAfterFullTargetPastFullStaysRestorable(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	sourceConn := openSourceTestDBConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	createMarkerTable(t, ctx, sourceConn)
	insertMarker(t, ctx, sourceConn, "before-full", "row-in-base-backup")

	enablePhysicalBackupsViaAPI(t, router, fixture, true)

	// Wait for the slot before writing anything: WAL produced before the supervisor claims the
	// database is never streamed, and that hole at the head of the window would break the restore
	// on its own, hiding whatever the sweep does.
	waitForSlotPresent(t, postgresql_executor.OpenAdminConn(t, fixture),
		fixture.DB.PostgresqlPhysical.ReplicationSlotName, 60*time.Second)

	chain := waitForChainBackups(t, router, fixture, 0, 3*time.Minute)

	insertMarker(t, ctx, sourceConn, "after-full", "row-replayed-up-to-target")
	streamPostFullSegments(t, ctx, sourceConn, fixture, chain, 2, 90*time.Second)

	// A margin wider than the whole-second recovery_target_time truncation on either side keeps the
	// target unambiguously inside the streamed range.
	time.Sleep(2 * time.Second)
	targetTime := time.Now().UTC()
	time.Sleep(2 * time.Second)

	insertMarker(t, ctx, sourceConn, "after-target", "row-after-target")

	// Natural WAL rather than a forced switch: the segment covering the target has to fill and
	// archive before the resolver counts it, and a switch would leave it partial.
	_, err := postgresql_executor.GenerateWalActivity(ctx, sourceConn, 64*1024*1024)
	require.NoError(t, err)

	waitForTargetTimeReplayable(t, fixture.DB.ID, targetTime, 90*time.Second)

	backuping_physical.RunOrphanWalCleanupForTest(t, fixture.DB.ID)

	token := requestRestoreTokenViaAPI(t, router, fixture, &targetTime)
	assert.NotEmpty(t, token,
		"a target covered by archived WAL must stay restorable after the orphan sweep has run")
}

// RunWalSlotAppearsWhenBackupingStartsRemovedWhenDatabaseDeleted proves the WAL replication-slot
// lifecycle end to end and purely over the API: no slot exists until WAL-stream backups are enabled,
// enabling them makes the supervisor create the persistent slot, and deleting the database (DELETE
// endpoint → cleanup listeners) removes it so nothing is left behind.
func RunWalSlotAppearsWhenBackupingStartsRemovedWhenDatabaseDeleted(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	require.NotEmpty(t, slotName, "a physical database must be assigned a slot name on creation")
	require.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"no WAL slot must exist before backuping is enabled")

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	waitForSlotPresent(t, adminConn, slotName, 30*time.Second)

	deleteDatabaseViaAPI(t, router, fixture)

	requireDatabaseSlotsGone(t, adminConn, fixture, 30*time.Second)
}

// RunWalSlotWhenDatabaseDeletedWithStreamedWalSlotRemovedSoNoWalStuck covers the failure-prone
// case: a database whose slot has actively streamed WAL (so it is pinning WAL on the source) is
// deleted. The cleanup must still drop the slot — otherwise WAL is stuck in the container forever.
// Generating real WAL first makes the slot's receiver active at deletion time, exercising the path a
// naive "refuse to drop an active slot" cleanup would orphan.
func RunWalSlotWhenDatabaseDeletedWithStreamedWalSlotRemovedSoNoWalStuck(t *testing.T, version, image string) {
	router, fixture := setupReplicationOnlyFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	waitForChainBackups(t, router, fixture, 0, 3*time.Minute)
	waitForSlotPresent(t, adminConn, slotName, 30*time.Second)

	// Drive real WAL so the streamer's slot is actively consuming and pinning
	// WAL — the case where a cleanup that refuses an active slot would orphan it.
	sourceConn := openSourceTestDBConn(t, fixture)
	_, err := postgresql_executor.GenerateWalActivity(ctx, sourceConn, 64*1024*1024)
	require.NoError(t, err)

	deleteDatabaseViaAPI(t, router, fixture)

	requireDatabaseSlotsGone(t, adminConn, fixture, 60*time.Second)
}

// The source publishes no port, so nothing on this path can silently fall back to a direct route.
func RunOverSshTunnelRecoversToTarget(t *testing.T, version, image string) {
	router, fixture, topology := setupBastionedFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	target := prepareRestoreTarget(t, image)

	sourceConn := openSourceTestDBConnThroughBastion(t, topology)

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()

	targetTime, survivingPhases := seedChainAndStreamPastTarget(t, ctx, router, sourceConn, fixture)

	bundle := downloadRestoreBundleViaAPI(t, router, fixture, &targetTime)
	reconstructCluster(t, target, router, image, bundle, &targetTime)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, survivingPhases, restoredPhases,
		"a chain built entirely through the bastion must restore the same rows as a direct one")
}

// SSH failures are classified apart from receiver crashes, and this is why. Killing the bastion
// makes every pg_receivewal spawn fail instantly, which unclassified reads as five rapid failures:
// the streamer would be marked FAILED and a chain-broken alert sent, leaving nobody consuming the
// slot until a later supervisor tick reclaims it — for a blip the forwarder heals on its own.
func RunWhenBastionDiesMidStreamKeepsTheSlotAndResumes(t *testing.T, version, image string) {
	router, fixture, topology := setupBastionedFixture(
		t, version, image, postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	require.NotEmpty(t, slotName)

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()

	enablePhysicalBackupsViaAPI(t, router, fixture, true)
	waitForChainBackups(t, router, fixture, 0, 5*time.Minute)

	adminConn := postgresql_executor.OpenAdminConn(t, fixture)
	waitForSlotPresent(t, adminConn, slotName, 60*time.Second)

	fullBackupsBeforeOutage := len(waitForChainBackups(t, router, fixture, 0, time.Minute))

	stopBastion(t, topology)

	// Comfortably longer than the five rapid failures plus their 2/4/8/16s backoff would need to
	// escalate, so a regression has every chance to tear the streamer down before we look.
	time.Sleep(75 * time.Second)

	requireStreamerNotFailed(t, fixture)

	startBastion(t, topology)

	resumedConn := postgresql_executor.OpenAdminConn(t, fixture)
	require.True(t, postgresql_executor.SlotExists(t, resumedConn, slotName),
		"the slot must survive the outage: rebuilding it opens a WAL gap and costs a fresh full backup")

	segmentsBeforeResume := getCommittedWalSegmentCount(t, fixture)

	_, err := postgresql_executor.GenerateWalActivity(ctx, resumedConn, 64*1024*1024)
	require.NoError(t, err)

	waitForCommittedWalSegmentsAbove(t, fixture, segmentsBeforeResume, 3*time.Minute)

	assert.Len(t, waitForChainBackups(t, router, fixture, 0, time.Minute), fullBackupsBeforeOutage,
		"a transport blip must not request an out-of-cadence full backup")
}
