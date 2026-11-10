package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func RunFullHistoryAfterPromotion(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	team.createPromotionDatabase(
		t,
		promotionSourceSpec{Primary: primary, Standby: standby, BackupType: apiclient.PhysicalBackupTypeFULL},
	)
	primaryConn := openConnAt(t, primary.Host, primary.Port)
	standbyConn := openConnAt(t, standby.Host, standby.Port)
	createMarkerTable(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-promotion", Payload: "before"})
	waitForMarkerOnStandby(t, standbyConn, "before-promotion")
	promoteStandby(t, standbyConn)
	insertMarker(t, standbyConn, markerRow{Phase: "after-promotion", Payload: "after"})

	target := prepareRestoreTarget(t, image)

	team.enableBackups(t, false)
	chain := team.waitForChain(t, 0)
	team.waitForTimelineHistory(t, 2)
	team.requirePublishedTimelineHistory(t, standbyConn)
	team.requireRestrictedPromotionRole(t)

	bundle := team.downloadBackupBundle(t, apptest.PhysicalBackupTokenRequest(chain[0].ID))
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)

	assert.ElementsMatch(t, []string{"before-promotion", "after-promotion"}, queryRestoredMarkerRows(t, target))
}
