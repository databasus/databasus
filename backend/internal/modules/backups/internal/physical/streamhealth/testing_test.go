package streamhealth

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

const (
	testMiB int64 = 1024 * 1024
	testGiB int64 = 1024 * testMiB

	testSegmentSizeBytes  = 16 * testMiB
	testRebuildLagBytes   = 256 * testMiB
	testPauseBytes        = 512 * testMiB
	testSamplesPerHour    = 120
	testArchiveProgressAt = walmath.LSN(0x40000000)
)

func getTestNow() time.Time { return time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC) }

type reasonExpectation struct {
	Code  ReasonCode
	Level ReasonLevel
}

func newHealthyTestSample(sampledAt time.Time) *catalog.WalStreamSample {
	return &catalog.WalStreamSample{
		ID:                       uuid.New(),
		DatabaseID:               uuid.Nil,
		SampledAt:                sampledAt,
		SessionStartedAt:         getTestNow().Add(-3 * time.Hour),
		SourceState:              catalog.WalStreamSourceStateReachable,
		SourceCurrentLSN:         new(testArchiveProgressAt + walmath.LSN(8*testMiB)),
		ArchiveProgressLSN:       new(testArchiveProgressAt),
		SourceWrittenBytes:       new(10 * testMiB),
		ArchivedSegments:         1,
		ArchivedWalBytes:         10 * testMiB,
		ArchivedStoredBytes:      4 * testMiB,
		WalSegmentSizeBytes:      testSegmentSizeBytes,
		HighestConcurrentUploads: 1,
		SlotWalStatus:            new("reserved"),
		SlotLagBytes:             new(2 * testMiB),
		SlotSafeWalSizeBytes:     new(10 * testGiB),
		UnarchivedWalBytes:       new(8 * testMiB),
		LocalQueueBytes:          testSegmentSizeBytes,
		LocalQueuePauseBytes:     testPauseBytes,
		IsReceiverRunning:        true,
	}
}

// buildTestSamples returns an hour of healthy samples, one every 30 seconds,
// the newest taken newestAge before getTestNow(). modify sees each sample with its
// index, oldest first, so a test can shape a trend across the hour.
func buildTestSamples(
	newestAge time.Duration,
	modify func(index int, sample *catalog.WalStreamSample),
) []*catalog.WalStreamSample {
	samples := make([]*catalog.WalStreamSample, 0, testSamplesPerHour)
	newestAt := getTestNow().Add(-newestAge)

	for index := range testSamplesPerHour {
		sampledAt := newestAt.Add(-time.Duration(testSamplesPerHour-1-index) * SampleInterval)
		sample := newHealthyTestSample(sampledAt)

		if modify != nil {
			modify(index, sample)
		}

		samples = append(samples, sample)
	}

	return samples
}

func modifyNewestTwo(
	modify func(sample *catalog.WalStreamSample),
) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		if index >= testSamplesPerHour-2 {
			modify(sample)
		}
	}
}

func modifyNewestOnly(
	modify func(sample *catalog.WalStreamSample),
) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		if index == testSamplesPerHour-1 {
			modify(sample)
		}
	}
}

func newRunningTestStreamer() *catalog.WalStreamer {
	return &catalog.WalStreamer{
		DatabaseID:      uuid.Nil,
		StartedAt:       getTestNow().Add(-3 * time.Hour),
		LastHeartbeatAt: getTestNow().Add(-10 * time.Second),
		Status:          catalog.WalStreamerStatusRunning,
	}
}

func newFailedTestStreamer(startedAgo, failingFor time.Duration) *catalog.WalStreamer {
	return &catalog.WalStreamer{
		DatabaseID:      uuid.Nil,
		StartedAt:       getTestNow().Add(-startedAgo),
		LastHeartbeatAt: getTestNow().Add(-startedAgo),
		Status:          catalog.WalStreamerStatusFailed,
		FailingSince:    new(getTestNow().Add(-failingFor)),
	}
}

func newTestVerdictInput(samples []*catalog.WalStreamSample) VerdictInput {
	input := VerdictInput{
		RecentSamples:          samples,
		Streamer:               newRunningTestStreamer(),
		SessionStartsLastHour:  0,
		SlotRebuildLagBytes:    testRebuildLagBytes,
		IsStreamingExpected:    true,
		HasCompletedFullBackup: true,
		Now:                    getTestNow(),
	}

	if len(samples) > 0 {
		input.NewestSample = samples[len(samples)-1]
	}

	return input
}

func getReasonExpectations(reasons []Reason) []reasonExpectation {
	expectations := make([]reasonExpectation, 0, len(reasons))

	for _, reason := range reasons {
		expectations = append(expectations, reasonExpectation{Code: reason.Code, Level: reason.Level})
	}

	return expectations
}

func getNoticeCodes(notices []Notice) []NoticeCode {
	codes := make([]NoticeCode, 0, len(notices))

	for _, notice := range notices {
		codes = append(codes, notice.Code)
	}

	return codes
}

func findReason(reasons []Reason, code ReasonCode) *Reason {
	for _, reason := range reasons {
		if reason.Code == code {
			return &reason
		}
	}

	return nil
}
