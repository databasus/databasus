package chains

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// walOwnership answers which archived WAL still belongs to a completed FULL: a segment is
// referenced when it overlaps a range of some completed FULL's timeline line. A segment on a
// timeline above 1 whose history is not published yet and that has no completed FULL is pending
// history: the history of a promoted timeline arrives separately from its WAL, so that WAL is kept
// until the history decides where the line goes.
type walOwnership struct {
	lines                 []TimelineLine
	historyTimelines      map[int]bool
	completedFullTimeline map[int]bool
}

func newWalOwnership(chainCatalog chainCatalog, excludedFullID uuid.UUID) walOwnership {
	ownership := walOwnership{
		lines:                 make([]TimelineLine, 0, len(chainCatalog.Fulls)),
		historyTimelines:      make(map[int]bool),
		completedFullTimeline: make(map[int]bool),
	}

	for _, history := range chainCatalog.Histories {
		ownership.historyTimelines[history.TimelineID] = true
	}

	for _, full := range chainCatalog.Fulls {
		if full.ID == excludedFullID || full.StartLSN == nil {
			continue
		}

		ownership.completedFullTimeline[full.TimelineID] = true
		ownership.lines = append(ownership.lines, BuildTimelineLine(chainCatalog.Histories, full))
	}

	return ownership
}

func (ownership walOwnership) isUnreferenced(timelineID int, segment catalog.LSNRange) bool {
	return !ownership.isPendingHistory(timelineID) && !ownership.isTimelineOnAnyLine(timelineID, segment)
}

func (ownership walOwnership) isPendingHistory(timelineID int) bool {
	return timelineID > 1 && !ownership.historyTimelines[timelineID] && !ownership.completedFullTimeline[timelineID]
}

func (ownership walOwnership) isTimelineOnAnyLine(timelineID int, segment catalog.LSNRange) bool {
	for _, line := range ownership.lines {
		for _, lineRange := range line.Ranges {
			if lineRange.TimelineID == timelineID && segment.Start < lineRange.EndLSN &&
				segment.End > lineRange.StartLSN {
				return true
			}
		}
	}

	return false
}

func (ownership walOwnership) hasTimelineOnAnyLine(timelineID int) bool {
	for _, line := range ownership.lines {
		for _, lineRange := range line.Ranges {
			if lineRange.TimelineID == timelineID {
				return true
			}
		}
	}

	return false
}

// getOwnedWalSpans returns the WAL a FULL's chain owns along its line: from its start up to the next
// completed FULL on the line.
func getOwnedWalSpans(chainCatalog chainCatalog, full *catalog.Backup) ([]catalog.TimelineSpan, error) {
	line := BuildTimelineLine(chainCatalog.Histories, full)

	span, err := getChainSpan(chainCatalog, full, line)
	if err != nil {
		return nil, err
	}

	return line.getSpans(span), nil
}

// hasPredecessorOnLine reports whether an older completed FULL can still replay the WAL after this
// one: its line holds this FULL's start.
func hasPredecessorOnLine(chainCatalog chainCatalog, full *catalog.Backup) bool {
	for _, candidate := range chainCatalog.Fulls {
		if candidate.ID == full.ID || candidate.StartLSN == nil || *candidate.StartLSN >= *full.StartLSN {
			continue
		}

		line := BuildTimelineLine(chainCatalog.Histories, candidate)
		if line.getRangeIndexContaining(full.TimelineID, *full.StartLSN) >= 0 {
			return true
		}
	}

	return false
}
