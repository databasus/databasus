package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
)

func TestBuildSlotBreakSample(t *testing.T) {
	t.Run("keeps the receiver start time while it runs and clears it after exit", func(t *testing.T) {
		supervisor := &Streamer{spec: WalStreamSpec{WalLagThresholdBytes: 1024}}
		startedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		observedAt := startedAt.Add(time.Minute)
		slot := &pgphysical.Slot{}
		supervisor.receiverRunStartedAtUnixNano.Store(startedAt.UnixNano())

		runningSample := supervisor.buildSlotBreakSample(slot, observedAt)
		assert.Same(t, slot, runningSample.SlotState)
		assert.Equal(t, int64(1024), runningSample.WalLagThresholdBytes)
		assert.Equal(t, startedAt, runningSample.ReceiverRunStartedAt)
		assert.Equal(t, observedAt, runningSample.ObservedAt)

		supervisor.receiverRunStartedAtUnixNano.Store(0)
		stoppedSample := supervisor.buildSlotBreakSample(slot, observedAt)
		assert.True(t, stoppedSample.ReceiverRunStartedAt.IsZero())
	})
}
