package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/util/testing/containers"
)

func TestPhysicalWalSlotRetentionSample(t *testing.T) {
	t.Run("records the start of extended slot retention while a paused receiver cannot advance", func(t *testing.T) {
		timings := backups.ProductionWalStreamTimings()
		timings.ForcedRotationInterval = 0
		timings.LocalQueuePauseBytes = 16 * 1024 * 1024
		team := newWalTeam(
			t,
			walTeamSpec{
				Timings:       &timings,
				SourceOptions: []containers.PhysicalPostgresOption{containers.WithWalKeepSize(0)},
			},
		)
		team.useUnavailableStorage(t)
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		_, err := source.Exec(t.Context(), "ALTER SYSTEM SET max_wal_size='32MB'")
		require.NoError(t, err)
		_, err = source.Exec(t.Context(), "SELECT pg_reload_conf()")
		require.NoError(t, err)
		generateWal(t, source, 2*16*1024*1024)
		rotateWal(t, source)
		testwait.Until(t, "the storage outage pauses the receiver", func(context.Context) error {
			reading := team.live(t)
			if reading.Current == nil || reading.Current.IsReceiverRunning ||
				reading.Current.LocalQueueBytes < timings.LocalQueuePauseBytes {
				return fmt.Errorf("receiver has not paused")
			}
			return nil
		})
		generateWal(t, source, 4*16*1024*1024)
		_, err = source.Exec(t.Context(), "CHECKPOINT")
		require.NoError(t, err)
		var first streamStatistics
		testwait.Until(t, "statistics expose extended slot retention with its start time", func(context.Context) error {
			first = team.statistics(t)
			if first.Current == nil || first.Current.SlotExtraWalSince == nil || first.Current.SlotWalStatus == nil ||
				*first.Current.SlotWalStatus != "extended" {
				return fmt.Errorf("no extended retention sample")
			}
			return nil
		})
		assert.True(
			t,
			first.Current.SlotExtraWalSince.Before(first.Current.SampledAt) ||
				first.Current.SlotExtraWalSince.Equal(first.Current.SampledAt),
		)
		testwait.Until(t, "the next sample preserves the same retention episode", func(context.Context) error {
			next := team.statistics(t)
			if next.Current == nil || !next.Current.SampledAt.After(first.Current.SampledAt) ||
				next.Current.SlotExtraWalSince == nil {
				return fmt.Errorf("no later extended retention sample")
			}
			assert.True(t, first.Current.SlotExtraWalSince.Equal(*next.Current.SlotExtraWalSince))
			return nil
		})
	})
}
