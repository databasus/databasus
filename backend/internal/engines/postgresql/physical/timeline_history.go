package physical

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

type TimelineHistoryRequest struct {
	ExpectedSystemIdentifier string
	TimelineID               int
}

type TimelineHistory struct {
	Filename string
	Body     []byte
	Fork     TimelineFork
}

func (s *Source) GetTimelineHistory(ctx context.Context, request TimelineHistoryRequest) (TimelineHistory, error) {
	if request.ExpectedSystemIdentifier == "" || request.TimelineID <= 1 ||
		uint64(request.TimelineID) > uint64(^uint32(0)) {
		return TimelineHistory{}, fmt.Errorf(
			"%w: cluster identity and timeline above 1 are required",
			ErrInvalidTimelineHistory,
		)
	}

	history, forks, err := s.readVerifiedTimelineHistory(ctx, request.ExpectedSystemIdentifier, request.TimelineID)
	if err != nil {
		return TimelineHistory{}, err
	}
	history.Fork = forks[len(forks)-1]

	return history, nil
}

// getLiveForkLSN returns the fork point's LSN at which the history of the server's live timeline
// leaves leftTimelineID, read through an identity-checked replication connection.
func (s *Source) getLiveForkLSN(
	ctx context.Context,
	expectedSystemIdentifier string,
	leftTimelineID int,
) (walmath.LSN, error) {
	if expectedSystemIdentifier == "" {
		return 0, fmt.Errorf("%w: cluster identity is required", ErrInvalidTimelineHistory)
	}

	_, forks, err := s.readVerifiedTimelineHistory(ctx, expectedSystemIdentifier, 0)
	if err != nil {
		return 0, err
	}

	for _, fork := range forks {
		if fork.ParentTimelineID == leftTimelineID {
			return fork.ForkLSN, nil
		}
	}

	return 0, fmt.Errorf("%w: live history does not leave timeline %d", ErrInvalidTimelineHistory, leftTimelineID)
}

// readVerifiedTimelineHistory reads a timeline's history from a server whose system identifier
// matches. A zero timelineID reads the history of the server's live timeline.
func (s *Source) readVerifiedTimelineHistory(
	ctx context.Context,
	expectedSystemIdentifier string,
	timelineID int,
) (TimelineHistory, []TimelineFork, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	connection, err := s.connectReplication(ctx, "")
	if err != nil {
		return TimelineHistory{}, nil, fmt.Errorf("connect to obtain timeline history: %w", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer closeCancel()
		_ = connection.Close(closeCtx)
	}()

	identityResults, err := connection.Exec(ctx, "IDENTIFY_SYSTEM").ReadAll()
	if err != nil {
		return TimelineHistory{}, nil, fmt.Errorf("identify history source: %w", err)
	}

	identity, err := getReplicationResultRow(identityResults, 4)
	if err != nil {
		return TimelineHistory{}, nil, err
	}

	if string(identity[0]) != expectedSystemIdentifier {
		return TimelineHistory{}, nil, &pgshared.ConnectionTestError{
			Code:    pgshared.ConnErrSystemIdentifierMismatch,
			Message: "",
		}
	}

	liveTimeline, err := strconv.ParseUint(string(identity[1]), 10, 32)
	if err != nil || liveTimeline < uint64(timelineID) {
		return TimelineHistory{}, nil, fmt.Errorf(
			"%w: requested timeline is ahead of source",
			ErrInvalidTimelineHistory,
		)
	}

	if timelineID == 0 {
		timelineID = int(liveTimeline)
	}

	if timelineID <= 1 {
		return TimelineHistory{}, nil, fmt.Errorf("%w: timeline 1 has no history", ErrInvalidTimelineHistory)
	}

	historyResults, err := connection.Exec(ctx, fmt.Sprintf("TIMELINE_HISTORY %d", timelineID)).ReadAll()
	if err != nil {
		return TimelineHistory{}, nil, fmt.Errorf("obtain history for timeline %d: %w", timelineID, err)
	}

	historyRow, err := getReplicationResultRow(historyResults, 2)
	if err != nil {
		return TimelineHistory{}, nil, err
	}

	history := TimelineHistory{Filename: string(historyRow[0]), Body: historyRow[1], Fork: TimelineFork{}}
	forks, err := parseTimelineHistory(timelineID, history)
	if err != nil {
		return TimelineHistory{}, nil, err
	}

	return history, forks, nil
}
