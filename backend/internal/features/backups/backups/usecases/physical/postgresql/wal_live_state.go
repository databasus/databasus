package usecases_physical_postgresql

import (
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// WalStreamLiveState is what the running streamer knows at one moment, read
// without storing anything. The source-side values are those of the newest slot
// probe, which the liveness watcher makes at least every 10 seconds.
type WalStreamLiveState struct {
	MeasuredAt time.Time

	// Nil until the session's first probe; the source-side fields below are nil
	// unless that probe found the source reachable with its slot.
	SourceProbedAt       *time.Time
	SourceState          *physical_enums.PhysicalWalStreamSourceState
	SourceCurrentLSN     *walmath.LSN
	SlotWalStatus        *string
	SlotLagBytes         *int64
	SlotSafeWalSizeBytes *int64

	// Nil while nothing has been archived for the database.
	ArchiveProgressLSN *walmath.LSN
	LastArchivedWalAt  *time.Time

	// Nil without a source position or an archive progress point; zero once the
	// progress point passes the probed position.
	UnarchivedWalBytes *int64

	LocalQueueBytes      int64
	LocalQueuePauseBytes int64
	IsReceiverRunning    bool
}

type timedSlotProbe struct {
	probe    slotProbe
	probedAt time.Time
}

// GetLiveState never takes the upload snapshot or the restart counters, which
// reset on read and belong to the next sample. The archive progress point and
// the unsaved amount are derived the way the sample derives them.
func (s *WalStreamSupervisor) GetLiveState(now time.Time) WalStreamLiveState {
	liveState := WalStreamLiveState{
		MeasuredAt:           now,
		LocalQueueBytes:      s.backlogBytes(),
		LocalQueuePauseBytes: s.highWatermarkBytes.Load(),
		IsReceiverRunning:    s.isReceiverRunning(),
	}

	newestCommitted, lastCommittedWalAt := s.getLastCommittedWal(s.logger)
	archiveProgressLSN := getArchiveProgressPoint(s.getLowestQueuedSegment(), newestCommitted)

	if archiveProgressLSN > 0 {
		liveState.ArchiveProgressLSN = &archiveProgressLSN
	}

	liveState.LastArchivedWalAt = lastCommittedWalAt

	latestProbe := s.latestProbe.Load()
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

	// The probe can be up to one watcher interval old while segments keep
	// landing, so the progress point may pass the probed position. Everything
	// the source had written by then is saved, which is zero, not unknown.
	if archiveProgressLSN > 0 {
		liveState.UnarchivedWalBytes = new(int64(0))

		if *slotState.CurrentLSN > archiveProgressLSN {
			liveState.UnarchivedWalBytes = new(int64(*slotState.CurrentLSN - archiveProgressLSN))
		}
	}

	return liveState
}
