package physicaltest

import "testing"

func RunPatroniInitialRecovery(t *testing.T, image string) {
	t.Helper()

	source := newPatroniSource(t, patroniSourceSpec{Image: image, ShouldIgnoreSlot: true})
	connection := source.connect(t)
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	throughLSN := currentLSN(t, connection)
	source.Team.streamPostFullSegments(t, connection, walArchiveSpec{Chain: chain, Count: 2})
	source.Team.waitForReplayableThroughLSN(t, chain, throughLSN)

	bundle := source.Team.downloadRestoreBundle(t, nil)
	target := prepareRestoreTarget(t, image)
	reconstructCluster(t, target, restoreSpec{App: source.Team.App, Image: image, BundlePath: bundle})
	startRestoredCluster(t, target, image)

	source.requireRestoredSchema(t, openConnAt(t, target.Host(), target.MappedPort()))
}
