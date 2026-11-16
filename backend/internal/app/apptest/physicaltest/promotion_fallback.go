package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// RunPromotionWithoutSummaries promotes a standby whose WAL summarization was turned off after
// Databasus accepted it as a source. The INCREMENTAL after the promotion cannot continue the chain, so exactly one replacement FULL follows, while a target
// between the promotion and that FULL restores from the earlier chain.
func RunPromotionWithoutSummaries(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	pair := containers.StartPhysicalReplicationPair(t, image)
	team.createPromotionDatabase(t, promotionSourceSpec{
		Primary: pair.Primary.Endpoint, Standby: pair.Standby.Endpoint,
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	})
	primaryConn := openConnAt(t, pair.Primary.Host, pair.Primary.Port)
	standbyConn := openConnAt(t, pair.Standby.Host, pair.Standby.Port)
	schema := NewSchemaRecovery(t, primaryConn)
	source := restorableSource{
		Team: team, Image: image, Schema: schema, Sequences: NewSequenceRecovery(t, primaryConn, schema),
	}
	createMarkerTable(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-full", Payload: "row-in-base-backup"})
	waitForMarkerOnStandby(t, standbyConn, "before-full")
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	source.applySchemaChanges(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-promotion", Payload: "row-before-promotion"})
	waitForMarkerOnStandby(t, standbyConn, "before-promotion")
	for _, statement := range []string{"ALTER SYSTEM SET summarize_wal = off", "SELECT pg_reload_conf()"} {
		_, err := standbyConn.Exec(t.Context(), statement)
		require.NoError(t, err)
	}

	promoteStandby(t, standbyConn)
	beforeReplacement := source.markRecoveryPoint(t, standbyConn, "after-promotion")
	generateWalActivity(t, standbyConn, 32*1024*1024)
	forceWalRotation(t, standbyConn)
	team.waitForTimelineHistory(t, 2)
	team.triggerIncremental(t)
	team.waitForReanchoredFull(t, chain[0].ID, 2)
	replacement := source.waitForNewFull(t, chain[0].ID)
	require.Equal(t, 2, replacement.TimelineID)
	source.requireSingleReplacement(t, fullReplacement{Preceding: chain[0].ID, Replacement: replacement.ID})

	replacementPoint := source.captureRecoveryPointOn(
		t,
		standbyConn,
		[]backupRow{replacement},
		"after-replacement-full",
	)
	source.restoreRecoveryPoint(t, replacementPoint)
	require.True(t, beforeReplacement.Target.Before(replacement.CreatedAt))
	source.restoreRecoveryPoint(t, beforeReplacement)
}
