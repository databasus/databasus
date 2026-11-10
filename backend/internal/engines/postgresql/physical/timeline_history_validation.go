package physical

import (
	"fmt"
	"strconv"
	"strings"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func validateTimelineHistory(timelineID int, history TimelineHistory) error {
	if history.Filename != walmath.FormatHistoryFilename(uint32(timelineID)) {
		return fmt.Errorf("%w: history filename does not match requested timeline", ErrInvalidTimelineHistory)
	}

	var previousTimeline uint64
	var previousPosition uint64
	for line := range strings.SplitSeq(string(history.Body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("%w: missing ancestor position", ErrInvalidTimelineHistory)
		}

		ancestorTimeline, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil || ancestorTimeline <= previousTimeline || ancestorTimeline >= uint64(timelineID) {
			return fmt.Errorf("%w: invalid ancestor timeline", ErrInvalidTimelineHistory)
		}

		positionParts := strings.Split(fields[1], "/")
		if len(positionParts) != 2 {
			return fmt.Errorf("%w: invalid ancestor position", ErrInvalidTimelineHistory)
		}

		highPosition, highErr := strconv.ParseUint(positionParts[0], 16, 32)
		lowPosition, lowErr := strconv.ParseUint(positionParts[1], 16, 32)
		position := highPosition<<32 | lowPosition
		if highErr != nil || lowErr != nil || position < previousPosition {
			return fmt.Errorf("%w: invalid ancestor position", ErrInvalidTimelineHistory)
		}

		previousTimeline = ancestorTimeline
		previousPosition = position
	}

	if previousTimeline == 0 {
		return fmt.Errorf("%w: no ancestor entries", ErrInvalidTimelineHistory)
	}

	return nil
}
