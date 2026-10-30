package physicaltest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func RunHostRestore(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTAL},
	)

	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	createMarkerTable(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "before-full", Payload: "row-before-full"})

	team.enableBackups(t, false)
	team.waitForChain(t, 0)

	bundle := team.downloadRestoreBundle(t, nil)
	outDir := reconstructOnHost(t, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})

	requireRecoveryRefusesReRestore(t, restoreSpec{App: team.App, Image: image, BundlePath: bundle}, outDir)

	endpoint := bootBoundCluster(t, clusterDirectory{Image: image, OutputDirectory: outDir})

	restoredPhases := queryMarkerRowsAt(t, endpoint.Host, endpoint.Port)
	assert.ElementsMatch(t, []string{"before-full"}, restoredPhases,
		"the entrypoint must serve the restored cluster from the bound volume, not a fresh empty one")
}

func RunHostTargetTime(t *testing.T, image string) {
	t.Helper()

	const primaryMaxConnections = 200

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{
			Image:      image,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Options:    []containers.PhysicalPostgresOption{containers.WithMaxConnections(primaryMaxConnections)},
		},
	)

	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	targetTime, survivingPhases := team.seedChainAndStreamPastTarget(t, sourceConn)

	bundle := team.downloadRestoreBundle(t, &targetTime)
	outDir := reconstructOnHost(
		t,
		restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: &targetTime},
	)

	autoConf, err := os.ReadFile(
		filepath.Join(hostClusterDir(clusterDirectory{Image: image, OutputDirectory: outDir}), "postgresql.auto.conf"),
	)
	require.NoError(t, err, "the recovery script must write postgresql.auto.conf")
	require.Contains(t, string(autoConf), "max_connections = 200",
		"the script must pin the primary's max_connections so archive recovery does not abort")

	endpoint := bootBoundCluster(t, clusterDirectory{Image: image, OutputDirectory: outDir})

	restoredPhases := queryMarkerRowsAt(t, endpoint.Host, endpoint.Port)
	assert.ElementsMatch(t, survivingPhases, restoredPhases,
		"PITR replayed through the entrypoint must recover rows at/before the target and drop the later one")
}

func RunMisaimedDirectory(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	requireRecoveryRefusesMisaimedDir(t, team.App, image)
}
