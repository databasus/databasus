package usecases_physical_postgresql

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

// Everything one lag-monitor tick knows when it records its sample. The trackers'
// times carry over ticks without source-side values, so their zero value, not a
// missing probe, means "no condition".
type walStreamSampleInput struct {
	DatabaseID       uuid.UUID
	SampledAt        time.Time
	SessionStartedAt time.Time

	Probe                  slotProbe
	SourceUnreachableSince *time.Time

	// The source position of the previous tick, nil after a tick without
	// source-side values and on the first tick of a session.
	PreviousSourceLSN *walmath.LSN

	// Zero while nothing has been archived for the database.
	ArchiveProgressLSN walmath.LSN

	Snapshot            ArchivingSnapshot
	WalSegmentSizeBytes int64

	SlotExtraWalSince   time.Time
	ArchiveStalledSince time.Time

	LocalQueueBytes      int64
	LocalQueuePauseBytes int64

	ReceiverRestarts       int
	ReceiverPauses         int
	IsReceiverRunning      bool
	IsForcedRotationDenied bool
}

func buildWalStreamSample(input walStreamSampleInput) *physical_models.PhysicalWalStreamSample {
	sample := &physical_models.PhysicalWalStreamSample{
		DatabaseID:               input.DatabaseID,
		SampledAt:                input.SampledAt,
		SessionStartedAt:         input.SessionStartedAt,
		SourceState:              getSampleSourceState(input.Probe),
		ArchivedSegments:         int(input.Snapshot.ArchivedSegments),
		ArchivedWalBytes:         input.Snapshot.ArchivedSegments * input.WalSegmentSizeBytes,
		ArchivedStoredBytes:      input.Snapshot.ArchivedBytes,
		WalSegmentSizeBytes:      input.WalSegmentSizeBytes,
		HighestConcurrentUploads: input.Snapshot.HighestConcurrentUploads,
		SlotExtraWalSince:        getTimeIfSet(input.SlotExtraWalSince),
		ArchiveStalledSince:      getTimeIfSet(input.ArchiveStalledSince),
		LocalQueueBytes:          input.LocalQueueBytes,
		LocalQueuePauseBytes:     input.LocalQueuePauseBytes,
		ReceiverRestarts:         input.ReceiverRestarts,
		ReceiverPauses:           input.ReceiverPauses,
		IsReceiverRunning:        input.IsReceiverRunning,
		IsForcedRotationDenied:   input.IsForcedRotationDenied,
	}

	if input.Probe.Outcome == slotProbeOutcomeUnreachable {
		sample.SourceUnreachableSince = input.SourceUnreachableSince
	}

	if input.ArchiveProgressLSN > 0 {
		sample.ArchiveProgressLSN = new(input.ArchiveProgressLSN)
	}

	if input.Snapshot.ArchivedSegments > 0 {
		sample.SegmentWriteAvgMillis = new(
			input.Snapshot.SegmentWrite.GetAverage(input.Snapshot.ArchivedSegments).Milliseconds(),
		)
		sample.SegmentWriteMaxMillis = new(input.Snapshot.SegmentWrite.Longest.Milliseconds())
	}

	if input.Probe.IsReachableWithSlot() {
		fillSourceSideValues(sample, input)
	}

	return sample
}

// The current position comes from the source itself rather than from the slot,
// so a lost slot, whose restart_lsn is gone, still yields a real position. A
// position behind the previous one means the source switched to a new
// timeline, and that tick's write volume is unknown rather than negative.
func fillSourceSideValues(sample *physical_models.PhysicalWalStreamSample, input walStreamSampleInput) {
	state := input.Probe.State
	currentLSN := *state.CurrentLSN

	sample.SourceCurrentLSN = new(currentLSN)
	sample.SlotWalStatus = new(state.WalStatus)
	sample.SlotLagBytes = state.LagBytes
	sample.SlotSafeWalSizeBytes = state.SafeWalSizeBytes

	if input.PreviousSourceLSN != nil && currentLSN >= *input.PreviousSourceLSN {
		sample.SourceWrittenBytes = new(int64(currentLSN - *input.PreviousSourceLSN))
	}

	if input.ArchiveProgressLSN > 0 && currentLSN >= input.ArchiveProgressLSN {
		sample.UnarchivedWalBytes = new(int64(currentLSN - input.ArchiveProgressLSN))
	}
}

func getSampleSourceState(probe slotProbe) physical_enums.PhysicalWalStreamSourceState {
	switch probe.Outcome {
	case slotProbeOutcomeReachable:
		return physical_enums.PhysicalWalStreamSourceStateReachable

	case slotProbeOutcomeSlotMissing:
		return physical_enums.PhysicalWalStreamSourceStateSlotMissing

	default:
		return physical_enums.PhysicalWalStreamSourceStateUnreachable
	}
}

func getTimeIfSet(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	return new(value)
}
