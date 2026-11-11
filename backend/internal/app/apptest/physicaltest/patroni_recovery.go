package physicaltest

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type patroniRecoveryPoint struct {
	Target      time.Time
	Timeline    int
	Observation recoveryObservation
	Phases      []string
}

func (source patroniSource) captureRecoveryPoint(t *testing.T, chain []backupRow, phase string) patroniRecoveryPoint {
	t.Helper()

	connection := source.connect(t)
	insertMarker(t, connection, markerRow{Phase: phase, Payload: phase})
	point := patroniRecoveryPoint{
		Timeline: chain[0].TimelineID,
		Observation: recoveryObservation{
			Schema:    source.Schema.GetState(t, connection),
			Sequences: source.Sequences.GetStates(t, connection),
		},
		Phases: queryMarkerRowsAt(t, source.Fixture.Endpoint.Host, source.Fixture.Endpoint.Port),
		Target: targetTimeWithMargin(t, connection),
	}
	insertMarker(t, connection, markerRow{Phase: "after-target-" + phase, Payload: "excluded from this recovery point"})
	throughLSN := currentLSN(t, connection)
	generateWalActivity(t, connection, 32*1024*1024)
	forceWalRotation(t, connection)
	source.Team.waitForReplayableThroughLSN(t, chain, throughLSN)
	source.Team.waitForTargetTimeReplayable(t, point.Target)

	return point
}

func (source patroniSource) restoreRecoveryPoint(t *testing.T, point patroniRecoveryPoint) {
	t.Helper()

	bundle := source.Team.downloadRestoreBundle(t, &point.Target)
	target := prepareRestoreTarget(t, source.Image)
	reconstructCluster(
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
		requireHistoryUsed(t, target, point.Timeline)
	}
}

func requireHistoryUsed(t *testing.T, target containers.RestoreTarget, timeline int) {
	t.Helper()

	logs := string(target.Exec(t, "cat", restoreWorkDir+"/pg.log"))
	require.Contains(t, logs, fmt.Sprintf("restored log file \"%08X.history\" from archive", timeline))
}

func (source patroniSource) requireUnreachableTarget(t *testing.T, target time.Time) {
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
