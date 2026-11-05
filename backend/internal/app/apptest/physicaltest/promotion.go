package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

// RunTimelineHistory pins issue #643: the history row must reference databases.id.
func RunTimelineHistory(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   standby,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		},
	)
	primaryConn := openConnAt(t, primary.Host, primary.Port)
	standbyConn := openConnAt(t, standby.Host, standby.Port)
	createMarkerTable(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "tl1-row", Payload: "row-on-tl1"})
	waitForMarkerOnStandby(t, standbyConn, "tl1-row")
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	team.streamPostFullSegments(t, primaryConn, walArchiveSpec{Chain: chain, Count: 2})

	promoteStandby(t, standbyConn)
	insertMarker(t, standbyConn, markerRow{Phase: "tl2-row", Payload: "row-on-tl2"})
	generateWalActivity(t, standbyConn, 16*1024*1024)
	team.waitForTimelineHistory(t, 2)
}

func (team Team) waitForTimelineHistory(t *testing.T, timelineID int) {
	t.Helper()

	testwait.Until(t, "the timeline history is cataloged on the parent database", func(context.Context) error {
		var hasHistory bool
		team.App.QuerySQL(
			t,
			&hasHistory,
			`SELECT EXISTS(SELECT 1 FROM backups_physical_wal_history_files WHERE database_id = ? AND timeline_id = ?)`,
			team.Database.Id,
			timelineID,
		)
		if !hasHistory {
			return fmt.Errorf("no history for database %s on timeline %d", team.Database.Id, timelineID)
		}
		return nil
	})
}

func RunPromotionReanchorsChain(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   standby,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		},
	)
	target := prepareRestoreTarget(t, image)
	primaryConn := openConnAt(t, primary.Host, primary.Port)
	standbyConn := openConnAt(t, standby.Host, standby.Port)
	createMarkerTable(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-promotion", Payload: "row-before-promotion"})
	waitForMarkerOnStandby(t, standbyConn, "before-promotion")
	team.enableBackups(t, true)
	initialChain := team.waitForChain(t, 0)
	initialFullID := initialChain[0].ID

	promoteStandby(t, standbyConn)
	insertMarker(t, standbyConn, markerRow{Phase: "after-promotion", Payload: "row-after-promotion"})
	generateWalActivity(t, standbyConn, 16*1024*1024)
	team.waitForTimelineHistory(t, 2)
	team.triggerIncremental(t)
	team.waitForReanchoredFull(t, initialFullID, 2)

	team.App.RunJobPass(t, "backups_physical_cleaner")
	bundle := team.downloadRestoreBundle(t, nil)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)
	assert.ElementsMatch(t, []string{"before-promotion", "after-promotion"}, queryRestoredMarkerRows(t, target))
}

func (team Team) waitForReanchoredFull(t *testing.T, staleFullID uuid.UUID, timelineID int) {
	t.Helper()

	testwait.Until(t, "promotion re-anchors the physical backup chain", func(context.Context) error {
		var incrementals []struct {
			Status      string
			ErrorReason string
		}
		team.App.QuerySQL(
			t,
			&incrementals,
			`SELECT status, COALESCE(error_reason, '') AS error_reason FROM backups_physical WHERE root_full_backup_id = ?`,
			staleFullID,
		)

		hasTimelineRefusal := false
		for _, incremental := range incrementals {
			if incremental.Status == "CHAIN_BROKEN" && incremental.ErrorReason == "TIMELINE_SWITCH_DETECTED" {
				hasTimelineRefusal = true
			}
		}

		fulls := team.listBackups(t, "FULL")
		completedFullCount := 0
		hasReplacementFull := false
		for _, full := range fulls {
			if full.Status == "COMPLETED" {
				if completedFullCount == 0 {
					hasReplacementFull = full.TimelineID == timelineID
				}
				completedFullCount++
			}
		}

		if hasTimelineRefusal && completedFullCount >= 2 && hasReplacementFull {
			return nil
		}
		return fmt.Errorf(
			"stale-chain incrementals: %+v; fulls newest first (all statuses): %+v; timeline refusal=%v",
			incrementals,
			fulls,
			hasTimelineRefusal,
		)
	})
}
