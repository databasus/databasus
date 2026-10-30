package streamhealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

const testMB int64 = 1000 * 1000

func perSample(bytesPerSecond int64) int64 {
	return bytesPerSecond * int64(SampleInterval.Seconds())
}

func writingAndArchiving(
	writeBytesPerSecond, archiveBytesPerSecond int64,
) func(index int, sample *catalog.WalStreamSample) {
	return func(_ int, sample *catalog.WalStreamSample) {
		sample.SourceWrittenBytes = new(perSample(writeBytesPerSecond))
		sample.ArchivedWalBytes = perSample(archiveBytesPerSecond)
	}
}

func combineModifiers(
	modifiers ...func(index int, sample *catalog.WalStreamSample),
) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		for _, modify := range modifiers {
			modify(index, sample)
		}
	}
}

func unarchivedWalOf(
	getBytes func(index int) int64,
) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		sample.UnarchivedWalBytes = new(getBytes(index))
	}
}

// 200 KiB over the 20 samples of the window.
func quietSource() func(index int, sample *catalog.WalStreamSample) {
	return func(_ int, sample *catalog.WalStreamSample) {
		sample.SourceWrittenBytes = new(int64(10 * 1024))
		sample.ArchivedWalBytes = 0
	}
}

func getTestKeepingUp(modify func(index int, sample *catalog.WalStreamSample)) KeepingUp {
	return GetKeepingUp(buildTestSamples(0, modify), testRebuildLagBytes, getTestNow())
}

func Test_GetKeepingUp_WithSpecScenarios_ReturnsStateRatesAndEstimates(t *testing.T) {
	samplesFromNewest := func(index int) int64 { return int64(testSamplesPerHour - 1 - index) }

	t.Run("Archiving keeps up", func(t *testing.T) {
		keepingUp := getTestKeepingUp(writingAndArchiving(10*testMB, 10*testMB))

		assert.Equal(t, KeepingUpStateKeepingUp, keepingUp.State)
		assert.InDelta(t, 10*testMB, *keepingUp.SourceWriteBytesPerSecond, 1)
		assert.InDelta(t, 10*testMB, *keepingUp.ArchiveBytesPerSecond, 1)
	})

	t.Run("Archiving falls behind", func(t *testing.T) {
		keepingUp := getTestKeepingUp(combineModifiers(
			writingAndArchiving(21*testMB, 12*testMB),
			unarchivedWalOf(func(index int) int64 { return int64(index) * perSample(9*testMB) }),
		))

		assert.Equal(t, KeepingUpStateFallingBehind, keepingUp.State)
		require.NotNil(t, keepingUp.UnarchivedWalChangeBytesPerSecond)
		assert.InDelta(t, 9*testMB, *keepingUp.UnarchivedWalChangeBytesPerSecond, 1)
	})

	t.Run("Catching up after a burst", func(t *testing.T) {
		shrinkPerSample := 600 * testMiB / 20

		keepingUp := getTestKeepingUp(combineModifiers(
			writingAndArchiving(testMB, 2*testMB),
			unarchivedWalOf(func(index int) int64 { return 3*testGiB + samplesFromNewest(index)*shrinkPerSample }),
		))

		assert.Equal(t, KeepingUpStateCatchingUp, keepingUp.State)
		require.NotNil(t, keepingUp.EstimatedCatchUpSeconds)
		assert.InDelta(t, 50*60, *keepingUp.EstimatedCatchUpSeconds, 3*60)
	})

	t.Run("Running out of allowance", func(t *testing.T) {
		keepingUp := getTestKeepingUp(func(index int, sample *catalog.WalStreamSample) {
			sample.SlotLagBytes = new(int64(index) * testGiB / 20)
			sample.SlotSafeWalSizeBytes = new(5 * testGiB)
		})

		require.NotNil(t, keepingUp.EstimatedSlotLimitSeconds)
		assert.InDelta(t, 50*60, *keepingUp.EstimatedSlotLimitSeconds, 60)
	})

	t.Run("Quiet database", func(t *testing.T) {
		keepingUp := getTestKeepingUp(combineModifiers(
			quietSource(),
			unarchivedWalOf(func(int) int64 { return 100 * 1024 }),
		))

		assert.Equal(t, KeepingUpStateIdle, keepingUp.State)
	})

	t.Run("Quiet database with a stuck backlog", func(t *testing.T) {
		keepingUp := getTestKeepingUp(combineModifiers(
			quietSource(),
			unarchivedWalOf(func(int) int64 { return 2 * testGiB }),
		))

		assert.Equal(t, KeepingUpStateKeepingUp, keepingUp.State)
	})
}

func Test_GetKeepingUp_WhenWindowHasNoWriteValues_ReportsUnknown(t *testing.T) {
	keepingUp := getTestKeepingUp(func(_ int, sample *catalog.WalStreamSample) {
		sample.SourceWrittenBytes = nil
	})

	assert.Equal(t, KeepingUpStateUnknown, keepingUp.State)
	assert.Nil(t, keepingUp.SourceWriteBytesPerSecond)
	assert.Nil(t, keepingUp.ArchiveBytesPerSecond)
}

func Test_GetKeepingUp_WhenQuietSourceShrinksBacklog_ReportsCatchingUpBeforeIdle(t *testing.T) {
	keepingUp := getTestKeepingUp(combineModifiers(
		quietSource(),
		unarchivedWalOf(func(index int) int64 { return 2*testGiB - int64(index)*testMiB }),
	))

	assert.Equal(t, KeepingUpStateCatchingUp, keepingUp.State)
}

func Test_GetKeepingUp_WhenWindowHasHole_BothRatesShareOneDenominator(t *testing.T) {
	keepingUp := getTestKeepingUp(func(index int, sample *catalog.WalStreamSample) {
		writingAndArchiving(10*testMB, 10*testMB)(index, sample)

		if index%2 == 0 {
			sample.SourceWrittenBytes = nil
		}
	})

	assert.InDelta(t, 10*testMB, *keepingUp.SourceWriteBytesPerSecond, 1)
	assert.InDelta(t, 10*testMB, *keepingUp.ArchiveBytesPerSecond, 1)
}

func Test_GetKeepingUp_WhenSlotLagGrows_EstimatesRebuildOnlyOutsideExtendedStatus(t *testing.T) {
	growingLag := func(walStatus string) func(index int, sample *catalog.WalStreamSample) {
		return func(index int, sample *catalog.WalStreamSample) {
			growingSlotLag(testMiB)(index, sample)
			sample.SlotWalStatus = new(walStatus)
		}
	}

	reservedKeepingUp := getTestKeepingUp(growingLag("reserved"))
	require.NotNil(t, reservedKeepingUp.EstimatedSlotRebuildSeconds)
	assert.Positive(t, *reservedKeepingUp.EstimatedSlotRebuildSeconds)

	extendedKeepingUp := getTestKeepingUp(growingLag("extended"))
	assert.Nil(t, extendedKeepingUp.EstimatedSlotRebuildSeconds)
}

func Test_GetKeepingUp_WhenNewestSampleIsStale_ReportsUnknown(t *testing.T) {
	keepingUp := GetKeepingUp(buildTestSamples(15*time.Minute, nil), testRebuildLagBytes, getTestNow())

	assert.Equal(t, KeepingUpStateUnknown, keepingUp.State)
	assert.Equal(t, testRebuildLagBytes, keepingUp.SlotRebuildLagBytes)
}
