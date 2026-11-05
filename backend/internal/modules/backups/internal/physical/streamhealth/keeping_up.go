package streamhealth

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// GetKeepingUp summarizes the keeping-up window that ends at now: how fast the
// source writes against how fast Databasus archives, how the backlog moves, and
// when it clears or when the source or Databasus acts on it.
func GetKeepingUp(
	recentSamples []*catalog.WalStreamSample,
	slotRebuildLagBytes int64,
	now time.Time,
) KeepingUp {
	keepingUp := KeepingUp{State: KeepingUpStateUnknown, SlotRebuildLagBytes: slotRebuildLagBytes}

	windowSamples := getSamplesInWindow(recentSamples, now)
	if len(windowSamples) == 0 {
		return keepingUp
	}

	newest := windowSamples[len(windowSamples)-1]
	keepingUp.UnarchivedWalBytes = newest.UnarchivedWalBytes
	keepingUp.SlotLagBytes = newest.SlotLagBytes

	writtenBytes, hasWriteValues := fillRates(&keepingUp, windowSamples)

	if changeBytesPerSecond, isKnown := getChangeBytesPerSecond(recentSamples, now, readUnarchivedWalBytes); isKnown {
		keepingUp.UnarchivedWalChangeBytesPerSecond = new(changeBytesPerSecond)
	}

	if changeBytesPerSecond, isKnown := getChangeBytesPerSecond(recentSamples, now, readSlotLagBytes); isKnown {
		keepingUp.SlotLagChangeBytesPerSecond = new(changeBytesPerSecond)
	}

	fillEstimates(&keepingUp, newest)

	if hasWriteValues {
		keepingUp.State = getKeepingUpState(keepingUp, newest, writtenBytes)
	}

	return keepingUp
}

// Both rates divide by the seconds covered by samples that hold a write value,
// so a hole in the window lowers neither rate alone.
func fillRates(
	keepingUp *KeepingUp,
	windowSamples []*catalog.WalStreamSample,
) (writtenBytes int64, hasWriteValues bool) {
	var archivedBytes int64

	writeSampleCount := 0

	for _, sample := range windowSamples {
		if sample.SourceWrittenBytes == nil {
			continue
		}

		writtenBytes += *sample.SourceWrittenBytes
		archivedBytes += sample.ArchivedWalBytes
		writeSampleCount++
	}

	if writeSampleCount == 0 {
		return 0, false
	}

	coveredSeconds := float64(writeSampleCount) * SampleInterval.Seconds()

	keepingUp.SourceWriteBytesPerSecond = new(float64(writtenBytes) / coveredSeconds)
	keepingUp.ArchiveBytesPerSecond = new(float64(archivedBytes) / coveredSeconds)

	return writtenBytes, true
}

// A slot held in extended or unreserved is never rebuilt for its lag, so it
// gets no rebuild estimate.
func fillEstimates(keepingUp *KeepingUp, newest *catalog.WalStreamSample) {
	if keepingUp.UnarchivedWalBytes != nil &&
		keepingUp.UnarchivedWalChangeBytesPerSecond != nil &&
		*keepingUp.UnarchivedWalChangeBytesPerSecond < 0 {
		keepingUp.EstimatedCatchUpSeconds = new(
			int64(float64(*keepingUp.UnarchivedWalBytes) / -*keepingUp.UnarchivedWalChangeBytesPerSecond),
		)
	}

	if keepingUp.SlotLagBytes == nil ||
		keepingUp.SlotLagChangeBytesPerSecond == nil ||
		*keepingUp.SlotLagChangeBytesPerSecond <= 0 {
		return
	}

	slotLagChangeBytesPerSecond := *keepingUp.SlotLagChangeBytesPerSecond

	if keepingUp.SlotRebuildLagBytes > *keepingUp.SlotLagBytes &&
		!hasSlotWalStatus(newest, slotWalStatusExtended) &&
		!hasSlotWalStatus(newest, slotWalStatusUnreserved) {
		keepingUp.EstimatedSlotRebuildSeconds = new(
			int64(float64(keepingUp.SlotRebuildLagBytes-*keepingUp.SlotLagBytes) / slotLagChangeBytesPerSecond),
		)
	}

	if newest.SlotSafeWalSizeBytes != nil {
		keepingUp.EstimatedSlotLimitSeconds = new(
			int64(float64(*newest.SlotSafeWalSizeBytes) / slotLagChangeBytesPerSecond),
		)
	}
}

// First match wins, so a shrinking backlog on a quiet source reads as catching
// up rather than idle.
func getKeepingUpState(
	keepingUp KeepingUp,
	newest *catalog.WalStreamSample,
	writtenBytes int64,
) KeepingUpState {
	thresholdBytes := getBacklogThresholdBytes(newest)
	unarchivedChange := keepingUp.UnarchivedWalChangeBytesPerSecond

	if newest.IsSourceReachable() &&
		unarchivedChange != nil &&
		getGrowthOverWindowBytes(*unarchivedChange) >= thresholdBytes {
		return KeepingUpStateFallingBehind
	}

	if keepingUp.UnarchivedWalBytes != nil &&
		*keepingUp.UnarchivedWalBytes >= thresholdBytes &&
		unarchivedChange != nil &&
		*unarchivedChange < 0 {
		return KeepingUpStateCatchingUp
	}

	if writtenBytes < idleSourceWrittenBytes &&
		keepingUp.UnarchivedWalBytes != nil &&
		*keepingUp.UnarchivedWalBytes <= newest.WalSegmentSizeBytes {
		return KeepingUpStateIdle
	}

	return KeepingUpStateKeepingUp
}
