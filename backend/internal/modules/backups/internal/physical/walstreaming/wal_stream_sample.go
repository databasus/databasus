package walstreaming

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type walStreamSampleInput struct {
	DatabaseID       uuid.UUID
	SampledAt        time.Time
	SessionStartedAt time.Time

	Probe                  slotProbe
	SourceUnreachableSince *time.Time

	PreviousSourceLSN *walmath.LSN

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

func buildWalStreamSample(input walStreamSampleInput) *catalog.WalStreamSample {
	sample := &catalog.WalStreamSample{
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

func fillSourceSideValues(sample *catalog.WalStreamSample, input walStreamSampleInput) {
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

func getSampleSourceState(probe slotProbe) catalog.WalStreamSourceState {
	switch probe.Outcome {
	case slotProbeOutcomeReachable:
		return catalog.WalStreamSourceStateReachable

	case slotProbeOutcomeSlotMissing:
		return catalog.WalStreamSourceStateSlotMissing

	case slotProbeOutcomeUnreachable:
		return catalog.WalStreamSourceStateUnreachable
	}
	return catalog.WalStreamSourceStateUnreachable
}

func getTimeIfSet(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	return new(value)
}
