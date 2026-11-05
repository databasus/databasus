package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

func TestPhysicalWalSlotRetentionSample(t *testing.T) {
	t.Parallel()

	t.Run("records the start of extended slot retention while a paused receiver cannot advance", func(t *testing.T) {
		t.Parallel()

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
		source := team.openSource(t)

		// The queue limit is one segment, so the receiver can pause before any poll sees it running.
		testwait.Until(t, "the streamer creates its replication slot", func(ctx context.Context) error {
			var slotCount int
			if err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE starts_with(slot_name,'databasus_slot')").
				Scan(&slotCount); err != nil {
				return err
			}

			if slotCount == 0 {
				return fmt.Errorf("the streamer has no replication slot yet")
			}

			return nil
		})

		_, err := source.Exec(t.Context(), "ALTER SYSTEM SET max_wal_size='32MB'")
		require.NoError(t, err)
		_, err = source.Exec(t.Context(), "SELECT pg_reload_conf()")
		require.NoError(t, err)

		testwait.Until(
			t,
			"PostgreSQL applies the WAL reservation limit before generating WAL",
			func(ctx context.Context) error {
				var maxWalSizeBytes int64
				if err := source.QueryRow(ctx, "SELECT pg_size_bytes(current_setting('max_wal_size'))").
					Scan(&maxWalSizeBytes); err != nil {
					return err
				}
				if maxWalSizeBytes != 32*1024*1024 {
					return fmt.Errorf("max_wal_size is still %d bytes", maxWalSizeBytes)
				}
				return nil
			},
		)

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
				var slotStatus, maxWalSize string
				var slotLag int64
				if err := source.QueryRow(t.Context(), "SELECT COALESCE(wal_status,''), COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn),0)::bigint, current_setting('max_wal_size') FROM pg_replication_slots WHERE slot_type='physical' LIMIT 1").
					Scan(&slotStatus, &slotLag, &maxWalSize); err != nil {
					return err
				}
				return fmt.Errorf(
					"no extended retention sample: source status=%s lag=%d max_wal_size=%s",
					slotStatus,
					slotLag,
					maxWalSize,
				)
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
