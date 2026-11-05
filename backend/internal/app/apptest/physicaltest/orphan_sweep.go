package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func RunOrphanSweep(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM},
	)

	conn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	createMarkerTable(t, conn)
	insertMarker(t, conn, markerRow{Phase: "before-full", Payload: "row-in-base-backup"})
	team.enableBackups(t, true)
	waitForSlotPresent(t, conn)
	chain := team.waitForChain(t, 0)
	insertMarker(t, conn, markerRow{Phase: "after-full", Payload: "row-replayed-up-to-target"})
	team.streamPostFullSegments(t, conn, walArchiveSpec{Chain: chain, Count: 2})
	targetTime := targetTimeWithMargin(t, conn)
	insertMarker(t, conn, markerRow{Phase: "after-target", Payload: "row-after-target"})
	generateWalActivity(t, conn, 64*1024*1024)
	team.waitForTargetTimeReplayable(t, targetTime)

	team.App.RunJobPass(t, "backups_physical_cleaner")
	assert.NotEmpty(t, team.restoreToken(t, &targetTime))
}
