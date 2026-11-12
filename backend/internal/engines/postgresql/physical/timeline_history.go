package physical

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

type TimelineHistoryRequest struct {
	ExpectedSystemIdentifier string
	TimelineID               int
}

type TimelineHistory struct {
	Filename string
	Body     []byte
}

func (s *Source) GetTimelineHistory(ctx context.Context, request TimelineHistoryRequest) (TimelineHistory, error) {
	if request.ExpectedSystemIdentifier == "" || request.TimelineID <= 1 ||
		uint64(request.TimelineID) > uint64(^uint32(0)) {
		return TimelineHistory{}, fmt.Errorf(
			"%w: cluster identity and timeline above 1 are required",
			ErrInvalidTimelineHistory,
		)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	connection, err := s.connectReplication(ctx, "")
	if err != nil {
		return TimelineHistory{}, fmt.Errorf("connect to obtain timeline history: %w", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer closeCancel()
		_ = connection.Close(closeCtx)
	}()

	identityResults, err := connection.Exec(ctx, "IDENTIFY_SYSTEM").ReadAll()
	if err != nil {
		return TimelineHistory{}, fmt.Errorf("identify history source: %w", err)
	}

	identity, err := getReplicationResultRow(identityResults, 4)
	if err != nil {
		return TimelineHistory{}, err
	}

	if string(identity[0]) != request.ExpectedSystemIdentifier {
		return TimelineHistory{}, &pgshared.ConnectionTestError{
			Code:    pgshared.ConnErrSystemIdentifierMismatch,
			Message: "",
		}
	}

	liveTimeline, err := strconv.ParseUint(string(identity[1]), 10, 32)
	if err != nil || liveTimeline < uint64(request.TimelineID) {
		return TimelineHistory{}, fmt.Errorf("%w: requested timeline is ahead of source", ErrInvalidTimelineHistory)
	}

	historyResults, err := connection.Exec(ctx, fmt.Sprintf("TIMELINE_HISTORY %d", request.TimelineID)).ReadAll()
	if err != nil {
		return TimelineHistory{}, fmt.Errorf("obtain history for timeline %d: %w", request.TimelineID, err)
	}

	historyRow, err := getReplicationResultRow(historyResults, 2)
	if err != nil {
		return TimelineHistory{}, err
	}

	history := TimelineHistory{Filename: string(historyRow[0]), Body: historyRow[1]}
	if err := validateTimelineHistory(request.TimelineID, history); err != nil {
		return TimelineHistory{}, err
	}

	return history, nil
}
