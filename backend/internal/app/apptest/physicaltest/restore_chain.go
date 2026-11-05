package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func RunFullOnly(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTAL},
	)
	target := prepareRestoreTarget(t, image)

	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	createMarkerTable(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "before-full", Payload: "row-before-full"})

	team.enableBackups(t, false)
	team.waitForChain(t, 0)

	bundle := team.downloadRestoreBundle(t, nil)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, []string{"before-full"}, restoredPhases,
		"a full-only restore must recover the row written before the FULL")
}

func RunFullAndIncrementals(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTAL},
	)
	target := prepareRestoreTarget(t, image)

	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	createMarkerTable(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "before-full", Payload: "row-before-full"})

	team.enableBackups(t, false)
	chain := team.waitForChain(t, 0)

	insertMarker(t, sourceConn, markerRow{Phase: "after-full", Payload: "row-between-full-and-incr1"})
	chain = team.buildIncremental(t, sourceConn, incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1})

	insertMarker(t, sourceConn, markerRow{Phase: "after-incr1", Payload: "row-between-incr1-and-incr2"})
	team.buildIncremental(t, sourceConn, incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 2})

	bundle := team.downloadRestoreBundle(t, nil)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t,
		[]string{"before-full", "after-full", "after-incr1"},
		restoredPhases,
		"restoring the latest point must recover rows from the FULL and both incrementals")
}
