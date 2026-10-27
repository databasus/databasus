package wal_health

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
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

var testNow = time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)

type reasonExpectation struct {
	Code  ReasonCode
	Level ReasonLevel
}

func newHealthyTestSample(sampledAt time.Time) *physical_models.PhysicalWalStreamSample {
	return &physical_models.PhysicalWalStreamSample{
		ID:                       uuid.New(),
		DatabaseID:               uuid.Nil,
		SampledAt:                sampledAt,
		SessionStartedAt:         testNow.Add(-3 * time.Hour),
		SourceState:              physical_enums.PhysicalWalStreamSourceStateReachable,
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
// the newest taken newestAge before testNow. modify sees each sample with its
// index, oldest first, so a test can shape a trend across the hour.
func buildTestSamples(
	newestAge time.Duration,
	modify func(index int, sample *physical_models.PhysicalWalStreamSample),
) []*physical_models.PhysicalWalStreamSample {
	samples := make([]*physical_models.PhysicalWalStreamSample, 0, testSamplesPerHour)
	newestAt := testNow.Add(-newestAge)

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
	modify func(sample *physical_models.PhysicalWalStreamSample),
) func(index int, sample *physical_models.PhysicalWalStreamSample) {
	return func(index int, sample *physical_models.PhysicalWalStreamSample) {
		if index >= testSamplesPerHour-2 {
			modify(sample)
		}
	}
}

func modifyNewestOnly(
	modify func(sample *physical_models.PhysicalWalStreamSample),
) func(index int, sample *physical_models.PhysicalWalStreamSample) {
	return func(index int, sample *physical_models.PhysicalWalStreamSample) {
		if index == testSamplesPerHour-1 {
			modify(sample)
		}
	}
}

func newRunningTestStreamer() *physical_models.PhysicalWalStreamer {
	return &physical_models.PhysicalWalStreamer{
		DatabaseID:      uuid.Nil,
		StartedAt:       testNow.Add(-3 * time.Hour),
		LastHeartbeatAt: testNow.Add(-10 * time.Second),
		Status:          physical_enums.PhysicalWalStreamerStatusRunning,
	}
}

func newFailedTestStreamer(startedAgo, failingFor time.Duration) *physical_models.PhysicalWalStreamer {
	return &physical_models.PhysicalWalStreamer{
		DatabaseID:      uuid.Nil,
		StartedAt:       testNow.Add(-startedAgo),
		LastHeartbeatAt: testNow.Add(-startedAgo),
		Status:          physical_enums.PhysicalWalStreamerStatusFailed,
		FailingSince:    new(testNow.Add(-failingFor)),
	}
}

func newTestVerdictInput(samples []*physical_models.PhysicalWalStreamSample) VerdictInput {
	input := VerdictInput{
		RecentSamples:          samples,
		Streamer:               newRunningTestStreamer(),
		SessionStartsLastHour:  0,
		SlotRebuildLagBytes:    testRebuildLagBytes,
		IsStreamingExpected:    true,
		HasCompletedFullBackup: true,
		Now:                    testNow,
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
