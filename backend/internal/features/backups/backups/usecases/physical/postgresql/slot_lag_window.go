package usecases_physical_postgresql

import (
	"time"

	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
)

// Adds up the lag change over the time the receiver spends running, so the lag
// threshold judges only what a running receiver fails to reduce. A pair of
// samples counts only inside one receiver run: the lag that builds while no
// receiver runs, or across a restart, is lag we caused by holding the receiver
// down. Restarts do not clear the sums, so a receiver the slot-LSN watcher or
// back pressure keeps restarting still reaches the rule when its runs do not
// reduce the lag.
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

	if w.runningTime < wal_health.WarningSlotStatusHoldPeriod {
		return false
	}

	minDrainBytes := thresholdBytes * wal_health.SlotLagMinDrainPercentOfThreshold / 100

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
