package streamhealth

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type sampleValueReader func(sample *catalog.WalStreamSample) *int64

func readUnarchivedWalBytes(sample *catalog.WalStreamSample) *int64 {
	return sample.UnarchivedWalBytes
}

func readSlotLagBytes(sample *catalog.WalStreamSample) *int64 {
	return sample.SlotLagBytes
}

// getSamplesInWindow returns the samples of the keeping-up window that ends at
// windowEnd. Each sample stands for the interval before it, so the window holds
// the samples after its start, not at it.
func getSamplesInWindow(
	samples []*catalog.WalStreamSample,
	windowEnd time.Time,
) []*catalog.WalStreamSample {
	windowStart := windowEnd.Add(-keepingUpWindow)

	windowSamples := make([]*catalog.WalStreamSample, 0, len(samples))

	for _, sample := range samples {
		if sample.SampledAt.After(windowStart) && !sample.SampledAt.After(windowEnd) {
			windowSamples = append(windowSamples, sample)
		}
	}

	return windowSamples
}

// getChangeBytesPerSecond measures how a value moved over the keeping-up window
// ending at windowEnd, from the lowest value in the window's first edge to the
// lowest in its last edge. The edges are at least five minutes apart, so the
// rate never divides by a short span.
func getChangeBytesPerSecond(
	samples []*catalog.WalStreamSample,
	windowEnd time.Time,
	readValue sampleValueReader,
) (changeBytesPerSecond float64, isKnown bool) {
	windowStart := windowEnd.Add(-keepingUpWindow)
	firstEdgeEnd := windowStart.Add(keepingUpWindowEdge)
	lastEdgeStart := windowEnd.Add(-keepingUpWindowEdge)

	var firstLowest, lastLowest *catalog.WalStreamSample

	for _, sample := range getSamplesInWindow(samples, windowEnd) {
		value := readValue(sample)
		if value == nil {
			continue
		}

		if !sample.SampledAt.After(firstEdgeEnd) &&
			(firstLowest == nil || *value < *readValue(firstLowest)) {
			firstLowest = sample
		}

		if !sample.SampledAt.Before(lastEdgeStart) &&
			(lastLowest == nil || *value <= *readValue(lastLowest)) {
			lastLowest = sample
		}
	}

	if firstLowest == nil || lastLowest == nil {
		return 0, false
	}

	spanSeconds := lastLowest.SampledAt.Sub(firstLowest.SampledAt).Seconds()
	if spanSeconds <= 0 {
		return 0, false
	}

	return float64(*readValue(lastLowest)-*readValue(firstLowest)) / spanSeconds, true
}

func getGrowthOverWindowBytes(changeBytesPerSecond float64) int64 {
	return int64(changeBytesPerSecond * keepingUpWindow.Seconds())
}

// The unarchived WAL normally holds the uploads in flight plus the open
// segment, so growth below four segments is noise on a cluster with large
// segments.
func getBacklogThresholdBytes(sample *catalog.WalStreamSample) int64 {
	return max(minBacklogThresholdBytes, backlogThresholdSegments*sample.WalSegmentSizeBytes)
}
