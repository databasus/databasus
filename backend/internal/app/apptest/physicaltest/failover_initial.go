package physicaltest

import (
	"archive/tar"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func RunFailoverInitialRecovery(t *testing.T, manager FailoverManager) {
	t.Helper()

	source := manager.startSource(t, failoverSourceSpec{})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-full", Payload: "full baseline"})

	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)

	source.Schema.CreatePostFullObjects(t, connection)
	source.Sequences.PrepareCalls(t, connection)
	source.Schema.ApplyBetweenIncrementals(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-incremental", Payload: "incremental data"})

	chain = source.Team.buildIncremental(
		t,
		connection,
		incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1},
	)

	require.Len(t, chain, 2)
	require.Equal(t, "INCREMENTAL", chain[1].Type)
	require.Equal(t, "COMPLETED", chain[1].Status)

	source.Schema.BeginWalChanges(t, connection)
	source.Sequences.ApplyWalChanges(t, connection)
	source.Schema.execute(t, connection, "COMMIT")
	insertMarker(t, connection, markerRow{Phase: "after-incremental", Payload: "WAL data"})
	throughLSN := currentLSN(t, connection)
	source.Team.streamPostFullSegments(t, connection, walArchiveSpec{Chain: chain, Count: 2})
	source.Team.waitForReplayableThroughLSN(t, chain, throughLSN)

	bundle := source.Team.downloadRestoreBundle(t, nil)
	requireIncrementalBundle(t, bundle)
	target := prepareRestoreTarget(t, source.Image)
	source.reconstructForBoot(t, target, restoreSpec{App: source.Team.App, Image: source.Image, BundlePath: bundle})
	startRestoredCluster(t, target, source.Image)

	restored := openConnAt(t, target.Host(), target.MappedPort())
	require.Equal(t, []string{"after-incremental", "before-full", "before-incremental"}, queryMarkerRows(t, restored))
	source.requireRestoredSchema(t, restored)
}

func requireIncrementalBundle(t *testing.T, path string) {
	t.Helper()

	bundle, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = bundle.Close() }()

	entries := map[string]bool{}
	reader := tar.NewReader(bundle)
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		entries[entry.Name] = true
	}

	require.Contains(t, entries, "full/backup_manifest")
	require.Contains(t, entries, "incr-1/backup_manifest")
}
