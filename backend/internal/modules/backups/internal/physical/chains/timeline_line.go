package chains

import (
	"slices"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// TimelineLineRange is the part of one timeline a line runs through: from StartLSN up to EndLSN,
// the fork point's LSN at which the line leaves the timeline, or catalog.LSNMax on the line's
// newest timeline.
type TimelineLineRange struct {
	TimelineID int
	StartLSN   walmath.LSN
	EndLSN     walmath.LSN
}

// TimelineLine is the WAL a FULL's chain can replay: the FULL's timeline from its start, then every
// later timeline of the newest line, each up to the fork point where the line leaves it. A FULL
// off the newest line, taken on a former timeline after its fork point or on an abandoned branch,
// gets its own timeline only and is not used for a restore by time or for the latest restore.
type TimelineLine struct {
	Ranges    []TimelineLineRange
	IsOffLine bool
}

// BuildTimelineLine computes the line of a completed FULL from the database's history rows. Only
// rows in the FULL's storage take part, and a line crosses a fork point only when the child
// timeline's row records it and was never marked conflicting. A FULL on a timeline newer than the
// newest line, whose history is not published yet, keeps its own timeline and stays usable.
func BuildTimelineLine(histories []*catalog.WalHistoryFile, full *catalog.Backup) TimelineLine {
	ownTimeline := TimelineLineRange{TimelineID: full.TimelineID, StartLSN: *full.StartLSN, EndLSN: catalog.LSNMax}
	newestLine := buildNewestTimelineLine(histories, full.StorageID)

	fullRangeIndex := slices.IndexFunc(newestLine, func(lineRange TimelineLineRange) bool {
		return lineRange.TimelineID == full.TimelineID
	})
	if fullRangeIndex < 0 {
		isAheadOfLine := full.TimelineID > newestLine[len(newestLine)-1].TimelineID

		return TimelineLine{Ranges: []TimelineLineRange{ownTimeline}, IsOffLine: !isAheadOfLine}
	}

	if *full.StartLSN >= newestLine[fullRangeIndex].EndLSN {
		return TimelineLine{Ranges: []TimelineLineRange{ownTimeline}, IsOffLine: true}
	}

	ownTimeline.EndLSN = newestLine[fullRangeIndex].EndLSN
	ranges := append([]TimelineLineRange{ownTimeline}, newestLine[fullRangeIndex+1:]...)

	return TimelineLine{Ranges: ranges, IsOffLine: false}
}

// buildNewestTimelineLine starts at the highest timeline with a usable history row in the storage
// and follows usable parent links down to the first timeline whose own link is missing or unusable.
// Without usable rows the newest line is timeline 1.
func buildNewestTimelineLine(histories []*catalog.WalHistoryFile, storageID uuid.UUID) []TimelineLineRange {
	usableForks := make(map[int]*catalog.WalHistoryFile)
	topTimelineID := 1
	for _, history := range histories {
		if history.StorageID != storageID || !history.HasUsableFork() {
			continue
		}

		usableForks[history.TimelineID] = history
		topTimelineID = max(topTimelineID, history.TimelineID)
	}

	ranges := []TimelineLineRange{{TimelineID: topTimelineID, StartLSN: 0, EndLSN: catalog.LSNMax}}
	for {
		child := ranges[0]
		fork, isUsable := usableForks[child.TimelineID]
		if !isUsable || *fork.ParentTimelineID >= child.TimelineID || *fork.ForkLSN > child.EndLSN {
			return ranges
		}

		ranges[0].StartLSN = *fork.ForkLSN
		parent := TimelineLineRange{TimelineID: *fork.ParentTimelineID, StartLSN: 0, EndLSN: *fork.ForkLSN}
		ranges = append([]TimelineLineRange{parent}, ranges...)
	}
}

// getSpans cuts the line to an LSN range, giving the part of each timeline a WAL lookup reads.
func (line TimelineLine) getSpans(span catalog.LSNRange) []catalog.TimelineSpan {
	spans := make([]catalog.TimelineSpan, 0, len(line.Ranges))
	for _, lineRange := range line.Ranges {
		start := max(lineRange.StartLSN, span.Start)
		end := min(lineRange.EndLSN, span.End)
		if start < end {
			spans = append(spans, catalog.TimelineSpan{
				TimelineID: lineRange.TimelineID,
				Span:       catalog.LSNRange{Start: start, End: end},
			})
		}
	}

	return spans
}

// getRangeIndexContaining returns the range that holds a position on the given timeline, or -1.
func (line TimelineLine) getRangeIndexContaining(timelineID int, lsn walmath.LSN) int {
	return slices.IndexFunc(line.Ranges, func(lineRange TimelineLineRange) bool {
		return lineRange.TimelineID == timelineID && lineRange.StartLSN <= lsn && lsn < lineRange.EndLSN
	})
}

// getRangeIndexHoldingStop returns the range a backup that stopped at lsn on the given timeline
// continues from, or -1. A backup that stopped exactly at the fork point still continues on the
// line: the next timeline carries every later record.
func (line TimelineLine) getRangeIndexHoldingStop(timelineID int, lsn walmath.LSN) int {
	return slices.IndexFunc(line.Ranges, func(lineRange TimelineLineRange) bool {
		return lineRange.TimelineID == timelineID && lineRange.StartLSN <= lsn && lsn <= lineRange.EndLSN
	})
}
