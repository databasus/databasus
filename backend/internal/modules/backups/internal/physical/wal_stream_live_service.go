package physical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/walstreaming"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) GetWalStreamLive(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*GetWalStreamLiveResponse, error) {
	_, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}

	liveState, isRunning := s.streams.GetLiveState(ctx, databaseID, s.clock.Now())
	if !isRunning {
		return &GetWalStreamLiveResponse{}, nil
	}

	return &GetWalStreamLiveResponse{
		IsStreamerRunning: true,
		Current:           toWalStreamLive(liveState),
	}, nil
}

func toWalStreamLive(
	liveState *walstreaming.WalStreamLiveState,
) *WalStreamLive {
	live := &WalStreamLive{
		MeasuredAt:           liveState.MeasuredAt.UTC(),
		SourceProbedAt:       getTimeInUTC(liveState.SourceProbedAt),
		SourceState:          liveState.SourceState,
		SlotWalStatus:        liveState.SlotWalStatus,
		SlotLagBytes:         liveState.SlotLagBytes,
		SlotSafeWalSizeBytes: liveState.SlotSafeWalSizeBytes,
		LastArchivedWalAt:    getTimeInUTC(liveState.LastArchivedWalAt),
		UnarchivedWalBytes:   liveState.UnarchivedWalBytes,
		LocalQueueBytes:      liveState.LocalQueueBytes,
		LocalQueuePauseBytes: liveState.LocalQueuePauseBytes,
		IsReceiverRunning:    liveState.IsReceiverRunning,
	}

	if liveState.SourceCurrentLSN != nil {
		live.SourceCurrentLsn = new(liveState.SourceCurrentLSN.String())
	}

	if liveState.ArchiveProgressLSN != nil {
		live.ArchiveProgressLsn = new(liveState.ArchiveProgressLSN.String())
	}

	return live
}
