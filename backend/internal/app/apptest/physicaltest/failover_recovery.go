package physicaltest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type recoveryPoint struct {
	Target time.Time
	// Timeline is the source's timeline when the point was marked. A restore to a point on a promoted
	// timeline replays across its fork point, which needs that timeline's history.
	Timeline    int
	Observation recoveryObservation
	Phases      []string
}

func (source failoverSource) captureRecoveryPoint(t *testing.T, chain []backupRow, phase string) recoveryPoint {
	t.Helper()

	return source.captureRecoveryPointOn(t, source.connect(t), chain, phase)
}

// captureRecoveryPointOn marks a recovery point through connection and waits until it is restorable.
func (source restorableSource) captureRecoveryPointOn(
	t *testing.T,
	connection *pgx.Conn,
	chain []backupRow,
	phase string,
) recoveryPoint {
	t.Helper()

	point := source.markRecoveryPoint(t, connection, phase)
	throughLSN := currentLSN(t, connection)
	generateWalActivity(t, connection, 32*1024*1024)
	forceWalRotation(t, connection)
	source.Team.waitForReplayableThroughLSN(t, chain, throughLSN)
	source.Team.waitForTargetTimeReplayable(t, point.Target)

	return point
}

// markRecoveryPoint writes a phase marker and records the state a restore to the following target
// must reproduce, then writes a row the restore must exclude. It does not wait for archiving, so a
// scenario can mark a point whose WAL Databasus has not received.
func (source restorableSource) markRecoveryPoint(t *testing.T, connection *pgx.Conn, phase string) recoveryPoint {
	t.Helper()

	insertMarker(t, connection, markerRow{Phase: phase, Payload: phase})
	point := recoveryPoint{
		Timeline: getLiveTimeline(t, connection),
		Observation: recoveryObservation{
			Schema:    source.Schema.GetState(t, connection),
			Sequences: source.Sequences.GetStates(t, connection),
		},
		Phases: queryMarkerRows(t, connection),
		Target: targetTimeWithMargin(t, connection),
	}
	insertMarker(t, connection, markerRow{Phase: "after-target-" + phase, Payload: "excluded from this recovery point"})

	return point
}

// markReplayablePoint marks a recovery point through connection and waits until Databasus can
// restore it, whichever timeline the WAL after the chain's newest backup lies on.
func (source restorableSource) markReplayablePoint(t *testing.T, connection *pgx.Conn, phase string) recoveryPoint {
	t.Helper()

	point := source.markRecoveryPoint(t, connection, phase)
	generateWalActivity(t, connection, 32*1024*1024)
	forceWalRotation(t, connection)
	source.Team.waitForTargetTimeReplayable(t, point.Target)

	return point
}

func (source restorableSource) restoreRecoveryPoint(t *testing.T, point recoveryPoint) {
	t.Helper()

	bundle := source.Team.downloadRestoreBundle(t, &point.Target)
	target := prepareRestoreTarget(t, source.Image)
	source.reconstructForBoot(
		t,
		target,
		restoreSpec{App: source.Team.App, Image: source.Image, BundlePath: bundle, TargetTime: &point.Target},
	)

	// The script targets the newest timeline of the recovery set, which is at least the point's.
	targetTimeline := 0
	if point.Timeline > 1 {
		targetTimeline = readRecoveryTargetTimeline(t, target, source.Image)
		require.GreaterOrEqual(t, targetTimeline, point.Timeline)
		historyFilename := fmt.Sprintf("%08X.history", targetTimeline)
		target.Exec(t, "test", "-f", restoredClusterDir(source.Image)+"/databasus_wal_restore/"+historyFilename)
		target.Exec(t, "rm", "-f", restoredClusterDir(source.Image)+"/pg_wal/"+historyFilename)
	}

	startRestoredCluster(t, target, source.Image)
	connection := openConnAt(t, target.Host(), target.MappedPort())
	require.Equal(t, point.Observation.Schema, source.Schema.GetState(t, connection))
	require.Equal(t, point.Observation.Sequences, source.Sequences.GetStates(t, connection))
	require.ElementsMatch(t, point.Phases, queryRestoredMarkerRows(t, target))
	source.requireRestoredSchema(t, connection)

	if targetTimeline > 1 {
		requireHistoryUsed(t, readRestoredServerLogs(t.Context(), target, source.Image), targetTimeline)
	}
}

func readRecoveryTargetTimeline(t *testing.T, target containers.RestoreTarget, image string) int {
	t.Helper()

	output := target.Exec(
		t,
		"sed", "-n", `s/^recovery_target_timeline = '\([0-9]*\)'$/\1/p`,
		restoredClusterDir(image)+"/postgresql.auto.conf",
	)
	timelineID, err := strconv.Atoi(strings.TrimSpace(string(output)))
	require.NoError(t, err, "the recovery script sets one target timeline: %q", output)

	return timelineID
}

// requireHistoryUsed accepts the message in a plain log and in a CSV log, which doubles its quotes.
func requireHistoryUsed(t *testing.T, serverLog string, timeline int) {
	t.Helper()

	require.Regexp(t, fmt.Sprintf(`restored log file "+%08X\.history"+ from archive`, timeline), serverLog)
}

func (source restorableSource) requireUnreachableTarget(t *testing.T, target time.Time) {
	t.Helper()

	var problem apiclient.Problem
	body := source.Team.call(
		t,
		physicalResponseSpec{
			Call:        source.Team.restoreTokenCall(&target),
			Status:      http.StatusConflict,
			Destination: &problem,
		},
	)
	require.Equal(t, apiclient.ProblemCodeTargetTimeUnreachable, problem.Code)
	require.NotContains(t, string(body), "\"token\"")
}
