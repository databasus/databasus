package physicaltest

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	containers "databasus-backend/internal/lib/testcontainers"
)

// CNPGRecoveryMode selects what a recovery set restores: an explicit target, the latest point or a
// single backup without WAL.
type CNPGRecoveryMode string

const (
	CNPGRecoveryTarget CNPGRecoveryMode = "target"
	CNPGRecoveryLatest CNPGRecoveryMode = "latest"
	CNPGRecoveryBackup CNPGRecoveryMode = "backup"
)

type CNPGIndependentSpec struct {
	Manager CNPGManagerSpec
	Mode    CNPGRecoveryMode
}

// Hibernation makes inherited replication settings unusable, so the copy must recover independently.
func RunCNPGIndependentRecovery(t *testing.T, spec CNPGIndependentSpec) {
	t.Helper()

	recovery := prepareCNPGIndependentRecovery(t, spec)
	recovery.Source.Fixture.Hibernate(t)
	recovery.boot(t)
}

type cnpgIndependentRecovery struct {
	Source      cnpgSource
	Mode        CNPGRecoveryMode
	Point       recoveryPoint
	Bundle      string
	TargetTime  *time.Time
	Phases      []string
	Target      containers.RestoreTarget
	Replacement backupRow
}

func prepareCNPGIndependentRecovery(t *testing.T, spec CNPGIndependentSpec) *cnpgIndependentRecovery {
	t.Helper()

	source := newCNPGSource(t, cnpgSourceSpec{Manager: spec.Manager, SlotMode: containers.CNPGSlotSynchronized})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-full", Payload: "preserved on the elected node"})

	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)

	previous := source.Cluster.GetIdentity(t)
	promoted := source.Cluster.Switchover(t)
	connection = source.connect(t)
	insertMarker(t, connection, markerRow{Phase: "after-switch", Payload: "in the replacement full"})

	source.plantInheritedRecoveryInputs(t, previous.Node)

	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	replacement := source.waitForNewFull(t, chain[0].ID)

	fullPhases := queryMarkerRows(t, connection)
	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-replacement-full")

	recovery := &cnpgIndependentRecovery{
		Source:      source,
		Mode:        spec.Mode,
		Point:       point,
		Replacement: replacement,
		Phases:      point.Phases,
	}
	switch spec.Mode {
	case CNPGRecoveryTarget:
		recovery.TargetTime = &point.Target
	case CNPGRecoveryLatest:
		recovery.Phases = queryMarkerRows(t, connection)
	case CNPGRecoveryBackup:
		recovery.Phases = fullPhases
	default:
		t.Fatalf("unknown recovery mode %s", spec.Mode)
	}

	return recovery
}

// CloudNativePG disables ALTER SYSTEM, so inherited recovery settings must be written to postgresql.auto.conf directly.
func (source cnpgSource) plantInheritedRecoveryInputs(t *testing.T, previousPrimary string) {
	t.Helper()

	primary := source.Fixture.GetIdentity(t).Node
	dataDirectory := source.Fixture.QueryNode(t, containers.NodeQuery{Node: primary, SQL: "SHOW data_directory"})
	settings := fmt.Sprintf(
		"primary_conninfo = 'host=%s port=5432 user=streaming_replica'\n"+
			"restore_command = 'false'\n"+
			"recovery_target_time = '2001-01-01 00:00:00+00'\n"+
			"recovery_target_xid = '1'\n"+
			"recovery_target_action = 'pause'\n",
		previousPrimary,
	)
	_, err := source.Fixture.ExecPostgresNodeCommand(
		t.Context(),
		containers.NodeCommand{Node: primary, Command: []string{
			"sh",
			"-c",
			`printf '%s' "$1" >> "$2/postgresql.auto.conf" && touch "$2/standby.signal" "$2/recovery.signal"`,
			"plant",
			settings,
			dataDirectory,
		}},
	)
	require.NoError(t, err)
}

func (recovery *cnpgIndependentRecovery) boot(t *testing.T) *pgx.Conn {
	t.Helper()

	source := recovery.Source
	if recovery.Mode == CNPGRecoveryBackup {
		recovery.Bundle = source.Team.downloadBackupBundle(
			t,
			apptest.PhysicalBackupTokenRequest(recovery.Replacement.ID),
		)
	} else {
		recovery.Bundle = source.Team.downloadRestoreBundle(t, recovery.TargetTime)
	}

	recovery.Target = prepareRestoreTarget(t, source.Image)
	source.reconstructForBoot(
		t,
		recovery.Target,
		restoreSpec{
			App:        source.Team.App,
			Image:      source.Image,
			BundlePath: recovery.Bundle,
			TargetTime: recovery.TargetTime,
		},
	)
	recovery.requireInheritedInputsIgnored(t)

	startRestoredCluster(t, recovery.Target, source.Image)
	restored := openConnAt(t, recovery.Target.Host(), recovery.Target.MappedPort())
	require.Equal(t, recovery.Point.Observation.Schema, source.Schema.GetState(t, restored))
	require.Equal(t, recovery.Point.Observation.Sequences, source.Sequences.GetStates(t, restored))
	require.ElementsMatch(t, recovery.Phases, queryRestoredMarkerRows(t, recovery.Target))

	var isInRecovery bool
	require.NoError(t, restored.QueryRow(t.Context(), "SELECT pg_is_in_recovery()").Scan(&isInRecovery))
	require.False(t, isInRecovery)
	source.requireRestoredSchema(t, restored)

	return restored
}

func (recovery *cnpgIndependentRecovery) requireInheritedInputsIgnored(t *testing.T) {
	t.Helper()

	target := recovery.Target
	clusterDirectory := restoredClusterDir(recovery.Source.Image)
	for _, signal := range []string{"standby.signal", "recovery.signal"} {
		target.Exec(t, "test", "-f", restoreWorkDir+"/bundle/recon/full/"+signal)
	}

	inputConfiguration := string(target.Exec(t, "cat", restoreWorkDir+"/bundle/recon/full/postgresql.auto.conf"))
	require.Contains(t, inputConfiguration, "primary_conninfo")
	require.Contains(t, inputConfiguration, "recovery_target_xid")

	configuration := string(target.Exec(t, "cat", clusterDirectory+"/postgresql.auto.conf"))
	require.NotContains(t, configuration, "primary_conninfo")
	require.NotContains(t, configuration, "recovery_target_xid")
	require.NotContains(t, configuration, "restore_command = 'false'")
	target.Exec(t, "test", "!", "-e", clusterDirectory+"/standby.signal")

	if recovery.Mode == CNPGRecoveryBackup {
		target.Exec(t, "test", "!", "-e", clusterDirectory+"/recovery.signal")
		target.Exec(t, "sh", "-c", "test ! -d /restore/bundle/wal || test -z \"$(ls -A /restore/bundle/wal)\"")
	}
}
