package physicaltest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

type PatroniIndependentSpec struct {
	Image                           string
	Mode                            string
	ShouldUseExternalConfig         bool
	ShouldHaveDefaultAuthentication bool
}

func RunPatroniIndependentRecovery(t *testing.T, spec PatroniIndependentSpec) {
	t.Helper()

	source := newPatroniSource(
		t,
		patroniSourceSpec{
			Image:                   spec.Image,
			ShouldIgnoreSlot:        true,
			ShouldUseExternalConfig: spec.ShouldUseExternalConfig || spec.Mode == "backup",
		},
	)
	connection := source.connect(t)
	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-full", Payload: "preserved on the elected node"})
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	previous := source.Fixture.GetIdentity(t)
	promoted := source.Fixture.Switchover(t)
	connection = source.connect(t)
	var sourceAuthenticationHash string
	var sourceIdentHash string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT md5(pg_read_file(current_setting('hba_file'))), md5(pg_read_file(current_setting('ident_file')))").
			Scan(&sourceAuthenticationHash, &sourceIdentHash),
	)
	insertMarker(t, connection, markerRow{Phase: "after-switch", Payload: "in the replacement full"})
	for _, statement := range []string{
		fmt.Sprintf("ALTER SYSTEM SET primary_conninfo = 'host=%s port=5432 user=replicator password=testpassword'", previous.Node),
		"ALTER SYSTEM SET restore_command = 'false'",
		"ALTER SYSTEM SET recovery_target_time = '2001-01-01 00:00:00+00'",
		"ALTER SYSTEM SET recovery_target_xid = '1'",
		"ALTER SYSTEM SET recovery_target_action = 'pause'",
	} {
		_, err := connection.Exec(t.Context(), statement)
		require.NoError(t, err)
	}

	node := source.Fixture.GetNode(promoted.Node)
	code, _, err := node.Container.Exec(
		t.Context(),
		[]string{"touch", "/var/lib/postgresql/patroni/standby.signal", "/var/lib/postgresql/patroni/recovery.signal"},
	)
	require.NoError(t, err)
	require.Zero(t, code)

	source.prepareInheritedConfiguration(t, spec)
	if spec.Mode == "backup" {
		source.prepareCopiedAuthentication(t, spec.ShouldHaveDefaultAuthentication)
	}

	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	replacement := source.waitForNewFull(t, chain[0].ID)
	fullPhases := queryMarkerRowsAt(t, source.Fixture.Endpoint.Host, source.Fixture.Endpoint.Port)
	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-replacement-full")
	source.Fixture.KillNode(t, previous.Node)

	var bundle string
	var targetTime *time.Time
	phases := point.Phases
	switch spec.Mode {
	case "target":
		targetTime = &point.Target
		bundle = source.Team.downloadRestoreBundle(t, targetTime)
	case "latest":
		bundle = source.Team.downloadRestoreBundle(t, nil)
		phases = queryMarkerRowsAt(t, source.Fixture.Endpoint.Host, source.Fixture.Endpoint.Port)
	case "backup":
		bundle = source.Team.downloadBackupBundle(t, apptest.PhysicalBackupTokenRequest(replacement.ID))
		phases = fullPhases
	default:
		t.Fatalf("unknown independent recovery mode %s", spec.Mode)
	}

	target := prepareRestoreTarget(t, spec.Image)
	output := reconstructCluster(
		t,
		target,
		restoreSpec{App: source.Team.App, Image: spec.Image, BundlePath: bundle, TargetTime: targetTime},
	)

	if spec.ShouldUseExternalConfig {
		require.Contains(t, string(output), "ACTION REQUIRED: the restored cluster has no configuration files")
		prepareIndependentConfiguration(t, target, spec.Image)
	}

	for _, signal := range []string{"standby.signal", "recovery.signal"} {
		target.Exec(t, "test", "-f", restoreWorkDir+"/bundle/recon/full/"+signal)
	}

	inputConfiguration := string(target.Exec(t, "cat", restoreWorkDir+"/bundle/recon/full/postgresql.auto.conf"))
	require.Contains(t, inputConfiguration, "primary_conninfo")
	require.Contains(t, inputConfiguration, "recovery_target_xid")

	configuration := string(target.Exec(t, "cat", restoredClusterDir(spec.Image)+"/postgresql.auto.conf"))
	require.NotContains(t, configuration, "primary_conninfo")
	require.NotContains(t, configuration, "recovery_target_xid")
	target.Exec(t, "test", "!", "-e", restoredClusterDir(spec.Image)+"/standby.signal")

	if spec.Mode == "backup" {
		require.NotContains(t, string(output), "ACTION REQUIRED")
		target.Exec(t, "test", "!", "-e", restoredClusterDir(spec.Image)+"/recovery.signal")
		target.Exec(t, "sh", "-c", "test ! -d /restore/bundle/wal || test -z \"$(ls -A /restore/bundle/wal)\"")
		for _, filename := range []string{"pg_hba.conf", "pg_ident.conf"} {
			if spec.ShouldHaveDefaultAuthentication {
				target.Exec(t, "test", "-f", restoreWorkDir+"/bundle/recon/full/"+filename)
				target.Exec(t, "test", "-f", restoredClusterDir(spec.Image)+"/"+filename)
			} else {
				target.Exec(t, "test", "!", "-e", restoreWorkDir+"/bundle/recon/full/"+filename)
				target.Exec(t, "test", "!", "-e", restoredClusterDir(spec.Image)+"/"+filename)
			}
		}
		localConfiguration := string(target.Exec(t, "cat", restoredClusterDir(spec.Image)+"/postgresql.conf"))
		require.Contains(t, localConfiguration, "hba_file '"+restoredClusterDir(spec.Image)+"/auth/hba.rules'")
		require.Contains(t, localConfiguration, "ident_file '"+restoredClusterDir(spec.Image)+"/auth/ident.rules'")
	}

	startRestoredCluster(t, target, spec.Image)
	restored := openConnAt(t, target.Host(), target.MappedPort())
	require.Equal(t, point.Observation.Schema, source.Schema.GetState(t, restored))
	require.Equal(t, point.Observation.Sequences, source.Sequences.GetStates(t, restored))
	require.ElementsMatch(t, phases, queryRestoredMarkerRows(t, target))
	source.requireRestoredSchema(t, restored)

	if spec.Mode == "backup" {
		authentication := copiedAuthentication{HBAHash: sourceAuthenticationHash, IdentHash: sourceIdentHash}
		requireCopiedAuthentication(t, restored, authentication)
		source.requireCopiedAuthenticationOnHost(t, bundle, authentication)
	}

	state, err := source.Fixture.GetNode(previous.Node).Container.State(t.Context())
	require.NoError(t, err)
	require.False(t, state.Running)

	var authenticationHash string
	var identHash string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT md5(pg_read_file(current_setting('hba_file'))), md5(pg_read_file(current_setting('ident_file')))").
			Scan(&authenticationHash, &identHash),
	)
	require.Equal(t, sourceAuthenticationHash, authenticationHash)
	require.Equal(t, sourceIdentHash, identHash)
	source.Team.requireRestrictedPromotionRole(t)
}
