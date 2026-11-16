package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

// RunTimelineHistory pins issue #643: the history row must reference databases.id.
func RunTimelineHistory(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	team.createPromotionDatabase(t, promotionSourceSpec{
		Primary: primary, Standby: standby, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	})
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
	team.requirePublishedTimelineHistory(t, standbyConn)
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

// RunPromotionContinuesChain promotes the standby Databasus backs up from. The chain taken on
// timeline 1 continues on timeline 2 without a replacement FULL, and targets on both sides of the
// promotion restore before and after the cleanup stages.
func RunPromotionContinuesChain(t *testing.T, image string) {
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
	chain = team.buildIncrementalOn(t, primaryConn, standbyConn, incrementalSpec{
		ParentStopLSN: chainTipStopLSN(t, chain), Count: 1,
	})
	insertMarker(t, primaryConn, markerRow{Phase: "before-promotion", Payload: "row-before-promotion"})
	waitForMarkerOnStandby(t, standbyConn, "before-promotion")

	promoteStandby(t, standbyConn)
	afterPromotion := source.markRecoveryPoint(t, standbyConn, "after-promotion")
	require.Equal(t, 2, afterPromotion.Timeline)
	generateWalActivity(t, standbyConn, 32*1024*1024)
	forceWalRotation(t, standbyConn)
	team.waitForTimelineHistory(t, 2)
	team.requirePublishedTimelineHistory(t, standbyConn)
	team.waitForTargetTimeReplayable(t, afterPromotion.Target)
	source.restoreRecoveryPoint(t, afterPromotion)
	team.requireLatestRestoreOnTimeline(t, latestRestoreSpec{
		Image: image, TimelineID: 2, Phases: queryMarkerRows(t, standbyConn),
	})

	team.runCleanupStages(t)
	source.restoreRecoveryPoint(t, afterPromotion)

	chain = team.buildIncrementalOn(t, standbyConn, standbyConn, incrementalSpec{
		ParentStopLSN: chainTipStopLSN(t, chain), Count: 2,
	})
	require.Len(
		t,
		team.listBackups(t, "FULL"),
		1,
		"the chain continues across the promotion without a replacement FULL",
	)
	require.Equal(t, 2, chain[len(chain)-1].TimelineID)
	afterIncremental := source.captureRecoveryPointOn(t, standbyConn, chain, "after-promoted-incremental")
	source.restoreRecoveryPoint(t, afterIncremental)

	team.runCleanupStages(t)
	pair.Primary.StopContainer(t)
	pair.Standby.StopContainer(t)
	source.restoreRecoveryPoint(t, afterPromotion)
	source.restoreRecoveryPoint(t, afterIncremental)
}

func (team Team) runCleanupStages(t *testing.T) {
	t.Helper()

	team.App.RunJobPass(t, "backups_physical_cleaner")
	team.App.RunJobPass(t, "storage_file_deletion")
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

type latestRestoreSpec struct {
	Image      string
	TimelineID int
	Phases     []string
}

// requireLatestRestoreOnTimeline restores the latest archived state, which the recovery script
// replays on the given timeline, and checks it holds every committed marker.
func (team Team) requireLatestRestoreOnTimeline(t *testing.T, spec latestRestoreSpec) {
	t.Helper()

	bundle := team.downloadRestoreBundle(t, nil)
	target := prepareRestoreTarget(t, spec.Image)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: spec.Image, BundlePath: bundle, TargetTime: nil})
	require.Equal(t, spec.TimelineID, readRecoveryTargetTimeline(t, target, spec.Image))
	startRestoredCluster(t, target, spec.Image)
	require.ElementsMatch(t, spec.Phases, queryRestoredMarkerRows(t, target))
}
