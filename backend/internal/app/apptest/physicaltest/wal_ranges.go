package physicaltest

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/testwait"
)

type walArchiveSpec struct {
	Chain []backupRow
	Count int
}

func (team Team) streamPostFullSegments(t *testing.T, conn *pgx.Conn, spec walArchiveSpec) {
	t.Helper()

	tipStopLSN := chainTipStopLSN(t, spec.Chain)
	timelineID := spec.Chain[len(spec.Chain)-1].TimelineID
	testwait.Until(t, "post-backup WAL segments are archived", func(context.Context) error {
		forceWalRotation(t, conn)
		segmentCount := 0
		for _, wal := range team.listBackups(t, "WAL") {
			if wal.TimelineID == timelineID && parseLSN(t, wal.StopLSN) > tipStopLSN && wal.WalSegmentsCount != nil {
				// A grouped range may straddle the tip; count only its post-tip segments.
				startLSN := max(parseLSN(t, wal.StartLSN), tipStopLSN)
				segmentCount += int((parseLSN(t, wal.StopLSN) - startLSN + (16*1024*1024 - 1)) / (16 * 1024 * 1024))
			}
		}

		if segmentCount < spec.Count {
			return fmt.Errorf("only %d of %d post-backup WAL segments archived", segmentCount, spec.Count)
		}
		return nil
	})
}

func (team Team) waitForReplayableThroughLSN(t *testing.T, chain []backupRow, throughLSN uint64) {
	t.Helper()

	tip := chain[len(chain)-1]
	testwait.Until(t, "contiguous WAL reaches the target LSN", func(context.Context) error {
		reachable := parseLSN(t, tip.StopLSN)
		walRows := team.listBackups(t, "WAL")
		slices.SortFunc(walRows, func(first, second backupRow) int {
			return cmp.Compare(parseLSN(t, first.StartLSN), parseLSN(t, second.StartLSN))
		})

		for _, wal := range walRows {
			if wal.TimelineID != tip.TimelineID || parseLSN(t, wal.StopLSN) <= reachable {
				continue
			}
			if wal.HasGapBefore || parseLSN(t, wal.StartLSN) > reachable {
				break
			}
			reachable = parseLSN(t, wal.StopLSN)
		}

		if reachable < throughLSN {
			return fmt.Errorf("WAL reaches %X, target %X; rows %+v", reachable, throughLSN, walRows)
		}
		return nil
	})
}
