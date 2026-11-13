package physicaltest

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

type recoveryPoint struct {
	Target      time.Time
	Timeline    int
	Observation recoveryObservation
	Phases      []string
}

func (source failoverSource) captureRecoveryPoint(t *testing.T, chain []backupRow, phase string) recoveryPoint {
	t.Helper()

	connection := source.connect(t)
	point := source.markRecoveryPoint(t, connection, phase)
	point.Timeline = chain[0].TimelineID
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
func (source failoverSource) markRecoveryPoint(t *testing.T, connection *pgx.Conn, phase string) recoveryPoint {
	t.Helper()

	insertMarker(t, connection, markerRow{Phase: phase, Payload: phase})
	point := recoveryPoint{
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

func (source failoverSource) restoreRecoveryPoint(t *testing.T, point recoveryPoint) {
	t.Helper()

	bundle := source.Team.downloadRestoreBundle(t, &point.Target)
	target := prepareRestoreTarget(t, source.Image)
	source.reconstructForBoot(
		t,
		target,
		restoreSpec{App: source.Team.App, Image: source.Image, BundlePath: bundle, TargetTime: &point.Target},
	)

	if point.Timeline > 1 {
		historyFilename := fmt.Sprintf("%08X.history", point.Timeline)
		target.Exec(t, "test", "-f", restoredClusterDir(source.Image)+"/databasus_wal_restore/"+historyFilename)
		target.Exec(t, "rm", "-f", restoredClusterDir(source.Image)+"/pg_wal/"+historyFilename)
	}

	startRestoredCluster(t, target, source.Image)
	connection := openConnAt(t, target.Host(), target.MappedPort())
	require.Equal(t, point.Observation.Schema, source.Schema.GetState(t, connection))
	require.Equal(t, point.Observation.Sequences, source.Sequences.GetStates(t, connection))
	require.ElementsMatch(t, point.Phases, queryRestoredMarkerRows(t, target))
	source.requireRestoredSchema(t, connection)

	if point.Timeline > 1 {
		requireHistoryUsed(t, readRestoredServerLogs(t.Context(), target, source.Image), point.Timeline)
	}
}

// requireHistoryUsed accepts the message in a plain log and in a CSV log, which doubles its quotes.
func requireHistoryUsed(t *testing.T, serverLog string, timeline int) {
	t.Helper()

	require.Regexp(t, fmt.Sprintf(`restored log file "+%08X\.history"+ from archive`, timeline), serverLog)
}

func (source failoverSource) requireUnreachableTarget(t *testing.T, target time.Time) {
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
