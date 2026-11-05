package physicaltest

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func (team Team) seedChainAndStreamPastTarget(
	t *testing.T,
	sourceConn *pgx.Conn,
) (targetTime time.Time, survivingPhases []string) {
	t.Helper()

	createMarkerTable(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "before-full", Payload: "row-in-base-backup"})
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	insertMarker(t, sourceConn, markerRow{Phase: "after-full", Payload: "row-between-full-and-incr1"})
	chain = team.buildIncremental(t, sourceConn, incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1})
	insertMarker(t, sourceConn, markerRow{Phase: "after-incr1", Payload: "row-between-incr1-and-incr2"})
	chain = team.buildIncremental(t, sourceConn, incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 2})

	insertMarker(t, sourceConn, markerRow{Phase: "before-target", Payload: "row-replayed-up-to-target"})
	generateWalActivity(t, sourceConn, 64*1024*1024)
	targetTime = targetTimeWithMargin(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "after-target", Payload: "row-after-target-must-be-absent"})

	afterTargetLSN := currentLSN(t, sourceConn)
	generateWalActivity(t, sourceConn, 64*1024*1024)
	team.waitForReplayableThroughLSN(t, chain, afterTargetLSN)
	return targetTime, []string{"before-full", "after-full", "after-incr1", "before-target"}
}
