package walstreaming

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

type slotLagWindow struct {
	runningTime    time.Duration
	lagChangeBytes int64

	hasPreviousSample    bool
	previousRunStartedAt time.Time
	previousObservedAt   time.Time
	previousLagBytes     int64
}

func (w *slotLagWindow) recordSampleAndDetectUnreducedLag(sample slotBreakSample) bool {
	lagBytes := *sample.SlotState.LagBytes
	thresholdBytes := sample.WalLagThresholdBytes

	if thresholdBytes <= 0 || lagBytes <= thresholdBytes {
		w.clear()

		return false
	}

	if sample.ReceiverRunStartedAt.IsZero() {
		w.hasPreviousSample = false

		return false
	}

	if w.hasPreviousSample && w.previousRunStartedAt.Equal(sample.ReceiverRunStartedAt) {
		w.runningTime += sample.ObservedAt.Sub(w.previousObservedAt)
		w.lagChangeBytes += lagBytes - w.previousLagBytes
	}

	w.rememberPreviousSample(sample)

	if w.runningTime < streamhealth.WarningSlotStatusHoldPeriod {
		return false
	}

	minDrainBytes := thresholdBytes * streamhealth.SlotLagMinDrainPercentOfThreshold / 100

	if -w.lagChangeBytes < minDrainBytes {
		w.clear()

		return true
	}

	w.runningTime = 0
	w.lagChangeBytes = 0

	return false
}

func (w *slotLagWindow) rememberPreviousSample(sample slotBreakSample) {
	w.hasPreviousSample = true
	w.previousRunStartedAt = sample.ReceiverRunStartedAt
	w.previousObservedAt = sample.ObservedAt
	w.previousLagBytes = *sample.SlotState.LagBytes
}

func (w *slotLagWindow) clear() {
	*w = slotLagWindow{}
}
