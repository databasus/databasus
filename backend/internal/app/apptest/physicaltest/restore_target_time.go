package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func RunTargetTime(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM},
	)
	target := prepareRestoreTarget(t, image)

	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	targetTime, survivingPhases := team.seedChainAndStreamPastTarget(t, sourceConn)

	bundle := team.downloadRestoreBundle(t, &targetTime)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: &targetTime})
	requireRestoredClusterNeedsNoZstd(t, target, image)
	startRestoredCluster(t, target, image)

	restoredPhases := queryRestoredMarkerRows(t, target)
	assert.ElementsMatch(t, survivingPhases, restoredPhases,
		"PITR must replay the chain plus rows committed at/before the target and drop the row after it")
}
