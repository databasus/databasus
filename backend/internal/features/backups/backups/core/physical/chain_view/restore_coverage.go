package chain_view

import (
	"errors"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

type TimeRange struct {
	From time.Time
	To   time.Time
}

// RestoreCoverage says what the newest chain can restore right now. It is
// empty while no full backup has completed.
type RestoreCoverage struct {
	// LatestRestorableAt is the moment a restore to the latest available point
	// reaches.
	LatestRestorableAt *time.Time

	// RestorableRanges are the stretches of time a restore can target, oldest
	// first. The last one ends at LatestRestorableAt.
	RestorableRanges []TimeRange

	// BlockingGaps are the WAL gaps above the newest completed backup of the
	// chain, which the latest restore cannot cross. A gap below that backup is
	// bridged by it.
	BlockingGaps []LSNRange
}

type restoreAnchor struct {
	stopLSN     walmath.LSN
	completedAt time.Time
}

// GetRestoreCoverage shares the resolution of ResolveRestoreSet, so a screen
// showing it never promises more than the restore delivers.
func (s *ChainViewService) GetRestoreCoverage(databaseID uuid.UUID) (*RestoreCoverage, error) {
	reach, err := s.resolveRestoreReach(databaseID, nil)
	if errors.Is(err, ErrNoChainForRestore) {
		return &RestoreCoverage{RestorableRanges: []TimeRange{}}, nil
	}

	if err != nil {
		return nil, err
	}

	blockingGaps := make([]LSNRange, 0)

	for _, gap := range findGapsBetweenSegments(reach.walSegments) {
		if gap.End > reach.lastStopLSN {
			blockingGaps = append(blockingGaps, gap)
		}
	}

	return &RestoreCoverage{
		LatestRestorableAt: &reach.reachableTime,
		RestorableRanges:   findRestorableRanges(getRestoreAnchors(reach), reach.walSegments),
		BlockingGaps:       blockingGaps,
	}, nil
}

// getRestoreAnchors lists the backups a restore can start from: the full
// backup and the incrementals a latest restore includes.
func getRestoreAnchors(reach *restoreReach) []restoreAnchor {
	anchors := make([]restoreAnchor, 0, len(reach.incrementals)+1)
	anchors = append(anchors, restoreAnchor{
		stopLSN:     *reach.chain.RootFull.StopLSN,
		completedAt: derefTime(reach.chain.RootFull.CompletedAt),
	})

	for _, incremental := range reach.incrementals {
		anchors = append(anchors, restoreAnchor{
			stopLSN:     *incremental.StopLSN,
			completedAt: *incremental.CompletedAt,
		})
	}

	return anchors
}

// findRestorableRanges opens a range at each anchor's completion and ends it
// where contiguousWalRun from that anchor would stop. The run ends are found
// once for all anchors, so the cost does not grow with anchors × segments.
func findRestorableRanges(
	anchors []restoreAnchor,
	segments []*physical_models.PhysicalWalSegment,
) []TimeRange {
	committedSegments := make([]*physical_models.PhysicalWalSegment, 0, len(segments))

	for _, segment := range segments {
		if segment.FileName != nil {
			committedSegments = append(committedSegments, segment)
		}
	}

	runEnds := findContiguousRunEnds(committedSegments)
	ranges := make([]TimeRange, 0, len(anchors))

	for _, anchor := range anchors {
		reachedAt := anchor.completedAt

		firstAfterAnchor := sort.Search(len(committedSegments), func(index int) bool {
			return committedSegments[index].EndLSN > anchor.stopLSN
		})

		if firstAfterAnchor < len(committedSegments) &&
			committedSegments[firstAfterAnchor].StartLSN <= anchor.stopLSN {
			reachedAt = committedSegments[runEnds[firstAfterAnchor]].ReceivedAt
		}

		if reachedAt.Before(anchor.completedAt) {
			reachedAt = anchor.completedAt
		}

		ranges = append(ranges, TimeRange{From: anchor.completedAt, To: reachedAt})
	}

	return mergeTimeRanges(ranges)
}

// findContiguousRunEnds maps each segment to the last segment of the unbroken
// run it starts, by the continuity rule of contiguousWalRun.
func findContiguousRunEnds(segments []*physical_models.PhysicalWalSegment) []int {
	runEnds := make([]int, len(segments))

	for index := len(segments) - 1; index >= 0; index-- {
		runEnds[index] = index

		if index+1 < len(segments) && segments[index+1].StartLSN <= segments[index].EndLSN {
			runEnds[index] = runEnds[index+1]
		}
	}

	return runEnds
}

func mergeTimeRanges(ranges []TimeRange) []TimeRange {
	slices.SortFunc(ranges, func(a, b TimeRange) int { return a.From.Compare(b.From) })

	merged := make([]TimeRange, 0, len(ranges))

	for _, timeRange := range ranges {
		last := len(merged) - 1

		if last >= 0 && !timeRange.From.After(merged[last].To) {
			if timeRange.To.After(merged[last].To) {
				merged[last].To = timeRange.To
			}

			continue
		}

		merged = append(merged, timeRange)
	}

	return merged
}
