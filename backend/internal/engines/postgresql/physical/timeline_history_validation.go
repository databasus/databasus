package physical

import (
	"fmt"
	"strconv"
	"strings"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// TimelineFork is one entry of a timeline's history: the ancestor timeline and the fork point's
// LSN, where the history leaves that ancestor. The last entry names the direct parent.
type TimelineFork struct {
	ParentTimelineID int
	ForkLSN          walmath.LSN
}

func validateTimelineHistory(timelineID int, history TimelineHistory) (TimelineFork, error) {
	forks, err := parseTimelineHistory(timelineID, history)
	if err != nil {
		return TimelineFork{}, err
	}

	return forks[len(forks)-1], nil
}

// parseTimelineHistory returns the history's entries oldest first. A valid history names at least
// one ancestor, its ancestors increase and stay below timelineID, and its fork points never move
// backwards.
func parseTimelineHistory(timelineID int, history TimelineHistory) ([]TimelineFork, error) {
	if history.Filename != walmath.FormatHistoryFilename(uint32(timelineID)) {
		return nil, fmt.Errorf("%w: history filename does not match requested timeline", ErrInvalidTimelineHistory)
	}

	var forks []TimelineFork
	var previousTimeline uint64
	var previousPosition uint64
	for line := range strings.SplitSeq(string(history.Body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("%w: missing ancestor position", ErrInvalidTimelineHistory)
		}

		ancestorTimeline, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil || ancestorTimeline <= previousTimeline || ancestorTimeline >= uint64(timelineID) {
			return nil, fmt.Errorf("%w: invalid ancestor timeline", ErrInvalidTimelineHistory)
		}

		positionParts := strings.Split(fields[1], "/")
		if len(positionParts) != 2 {
			return nil, fmt.Errorf("%w: invalid ancestor position", ErrInvalidTimelineHistory)
		}

		highPosition, highErr := strconv.ParseUint(positionParts[0], 16, 32)
		lowPosition, lowErr := strconv.ParseUint(positionParts[1], 16, 32)
		position := highPosition<<32 | lowPosition
		if highErr != nil || lowErr != nil || position < previousPosition {
			return nil, fmt.Errorf("%w: invalid ancestor position", ErrInvalidTimelineHistory)
		}

		previousTimeline = ancestorTimeline
		previousPosition = position
		forks = append(forks, TimelineFork{ParentTimelineID: int(ancestorTimeline), ForkLSN: walmath.LSN(position)})
	}

	if len(forks) == 0 {
		return nil, fmt.Errorf("%w: no ancestor entries", ErrInvalidTimelineHistory)
	}

	return forks, nil
}
