package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

const liveStateQueuedStartLSN = walmath.LSN(0x40000000)

func TestBuildWalStreamLiveState(t *testing.T) {
	t.Run("reports a changed queue without source values when no probe has run", func(t *testing.T) {
		archivedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		inputs := walStreamLiveInputs{
			MeasuredAt: archivedAt.Add(time.Minute), LocalQueueBytes: 3 * 1024 * 1024,
			LastArchivedWalAt: &archivedAt,
		}

		firstReading := buildWalStreamLiveState(inputs)

		assert.Equal(t, inputs.MeasuredAt, firstReading.MeasuredAt)
		assert.Equal(t, int64(3*1024*1024), firstReading.LocalQueueBytes)
		require.NotNil(t, firstReading.LastArchivedWalAt)
		assert.Equal(t, archivedAt, *firstReading.LastArchivedWalAt)
		assert.Nil(t, firstReading.SourceProbedAt)
		assert.Nil(t, firstReading.SlotLagBytes)
		inputs.LocalQueueBytes = 0

		secondReading := buildWalStreamLiveState(inputs)

		assert.Zero(t, secondReading.LocalQueueBytes)
	})
	t.Run("reports the newest probe when the source is reachable", func(t *testing.T) {
		inputs := walStreamLiveInputs{ArchiveProgressLSN: liveStateQueuedStartLSN, MeasuredAt: time.Now().UTC()}
		probedAt := time.Now().UTC().Add(-7 * time.Second)

		inputs.LatestProbe = &timedSlotProbe{
			probe: slotProbe{Outcome: slotProbeOutcomeReachable, State: &pgphysical.Slot{
				WalStatus:  "reserved",
				LagBytes:   new(int64(1024 * 1024)),
				CurrentLSN: new(liveStateQueuedStartLSN + 5*1024*1024),
			}},
			probedAt: probedAt,
		}

		liveState := buildWalStreamLiveState(inputs)

		require.NotNil(t, liveState.SourceProbedAt)
		assert.Equal(t, probedAt, *liveState.SourceProbedAt, "source values carry the time of the probe they come from")
		assert.Equal(t, catalog.WalStreamSourceStateReachable, *liveState.SourceState)
		assert.Equal(t, int64(1024*1024), *liveState.SlotLagBytes)
		assert.Equal(t, "reserved", *liveState.SlotWalStatus)
		assert.Equal(t, liveStateQueuedStartLSN, *liveState.ArchiveProgressLSN,
			"the lowest queued segment holds the progress point back, as in a sample")
		assert.Equal(t, int64(5*1024*1024), *liveState.UnarchivedWalBytes)
	})
	t.Run("reports zero unsaved bytes when archiving passes the source probe", func(t *testing.T) {
		inputs := walStreamLiveInputs{ArchiveProgressLSN: liveStateQueuedStartLSN, MeasuredAt: time.Now().UTC()}
		inputs.LatestProbe = &timedSlotProbe{
			probe: slotProbe{Outcome: slotProbeOutcomeReachable, State: &pgphysical.Slot{
				LagBytes:   new(int64(0)),
				WalStatus:  "reserved",
				CurrentLSN: new(liveStateQueuedStartLSN - 1024),
			}},
			probedAt: time.Now().UTC().Add(-9 * time.Second),
		}

		liveState := buildWalStreamLiveState(inputs)

		require.NotNil(t, liveState.UnarchivedWalBytes,
			"segments saved after the probe must not make the unsaved amount unknown")
		assert.Equal(t, int64(0), *liveState.UnarchivedWalBytes)
	})
	t.Run("reports no source values when the source is unreachable", func(t *testing.T) {
		inputs := walStreamLiveInputs{ArchiveProgressLSN: liveStateQueuedStartLSN, MeasuredAt: time.Now().UTC()}
		inputs.LatestProbe = &timedSlotProbe{
			probe:    slotProbe{Outcome: slotProbeOutcomeUnreachable},
			probedAt: time.Now().UTC(),
		}

		liveState := buildWalStreamLiveState(inputs)

		assert.Equal(t, catalog.WalStreamSourceStateUnreachable, *liveState.SourceState)
		assert.Nil(t, liveState.SlotLagBytes)
		assert.Nil(t, liveState.SourceCurrentLSN)
		assert.Nil(t, liveState.UnarchivedWalBytes)
	})
}
