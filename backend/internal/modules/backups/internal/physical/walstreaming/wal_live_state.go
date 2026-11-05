package walstreaming

import (
	"context"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type WalStreamLiveState struct {
	MeasuredAt time.Time

	SourceProbedAt       *time.Time
	SourceState          *catalog.WalStreamSourceState
	SourceCurrentLSN     *walmath.LSN
	SlotWalStatus        *string
	SlotLagBytes         *int64
	SlotSafeWalSizeBytes *int64

	ArchiveProgressLSN *walmath.LSN
	LastArchivedWalAt  *time.Time

	UnarchivedWalBytes *int64

	LocalQueueBytes      int64
	LocalQueuePauseBytes int64
	IsReceiverRunning    bool
}

type timedSlotProbe struct {
	probe    slotProbe
	probedAt time.Time
}

func (s *Streamer) GetLiveState(ctx context.Context, now time.Time) WalStreamLiveState {
	newestCommitted, lastCommittedWalAt := s.getLastCommittedWal(ctx, s.logger)
	return s.getLiveStateForArchive(
		walLiveArchive{MeasuredAt: now, NewestCommitted: newestCommitted, LastCommittedWalAt: lastCommittedWalAt},
	)
}

type walLiveArchive struct {
	MeasuredAt         time.Time
	NewestCommitted    *walPosition
	LastCommittedWalAt *time.Time
}

func (s *Streamer) getLiveStateForArchive(archive walLiveArchive) WalStreamLiveState {
	return buildWalStreamLiveState(walStreamLiveInputs{
		MeasuredAt:           archive.MeasuredAt,
		LocalQueueBytes:      s.backlogBytes(),
		LocalQueuePauseBytes: s.highWatermarkBytes.Load(),
		IsReceiverRunning:    s.isReceiverRunning(),
		ArchiveProgressLSN:   getArchiveProgressPoint(s.getLowestQueuedSegment(), archive.NewestCommitted),
		LastArchivedWalAt:    archive.LastCommittedWalAt,
		LatestProbe:          s.latestProbe.Load(),
	})
}

type walStreamLiveInputs struct {
	MeasuredAt           time.Time
	LocalQueueBytes      int64
	LocalQueuePauseBytes int64
	IsReceiverRunning    bool
	ArchiveProgressLSN   walmath.LSN
	LastArchivedWalAt    *time.Time
	LatestProbe          *timedSlotProbe
}

func buildWalStreamLiveState(inputs walStreamLiveInputs) WalStreamLiveState {
	liveState := WalStreamLiveState{
		MeasuredAt:           inputs.MeasuredAt,
		LocalQueueBytes:      inputs.LocalQueueBytes,
		LocalQueuePauseBytes: inputs.LocalQueuePauseBytes,
		IsReceiverRunning:    inputs.IsReceiverRunning,
	}
	archiveProgressLSN := inputs.ArchiveProgressLSN

	if archiveProgressLSN > 0 {
		liveState.ArchiveProgressLSN = &archiveProgressLSN
	}

	liveState.LastArchivedWalAt = inputs.LastArchivedWalAt

	latestProbe := inputs.LatestProbe
	if latestProbe == nil {
		return liveState
	}

	sourceState := getSampleSourceState(latestProbe.probe)
	liveState.SourceProbedAt = &latestProbe.probedAt
	liveState.SourceState = &sourceState

	if !latestProbe.probe.IsReachableWithSlot() {
		return liveState
	}

	slotState := latestProbe.probe.State
	liveState.SourceCurrentLSN = slotState.CurrentLSN
	liveState.SlotWalStatus = &slotState.WalStatus
	liveState.SlotLagBytes = slotState.LagBytes
	liveState.SlotSafeWalSizeBytes = slotState.SafeWalSizeBytes

	if archiveProgressLSN > 0 {
		liveState.UnarchivedWalBytes = new(int64(0))

		if *slotState.CurrentLSN > archiveProgressLSN {
			liveState.UnarchivedWalBytes = new(int64(*slotState.CurrentLSN - archiveProgressLSN))
		}
	}

	return liveState
}
