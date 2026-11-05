package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

func TestPhysicalWalBackpressure(t *testing.T) {
	t.Parallel()

	t.Run(
		"counts a pause and then a restart after the queue drains and the receiver loses its source connection",
		func(t *testing.T) {
			t.Parallel()

			timings := backups.ProductionWalStreamTimings()
			timings.ForcedRotationInterval = 0
			timings.LocalQueuePauseBytes = 3 * 16 * 1024 * 1024

			team := newWalTeam(t, walTeamSpec{Timings: &timings})
			team.useUnavailableStorage(t)
			team.enable(t)
			team.waitForReceiver(t)
			source := team.openSource(t)

			var slot string
			team.App.QuerySQL(
				t,
				&slot,
				"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)

			waitForSourceFlush := func(targetLSN string, canStopAtPause bool) {
				testwait.Until(
					t,
					"the receiver flushes the source position or reaches its queue limit",
					func(context.Context) error {
						if canStopAtPause && team.live(t).Current.LocalQueueBytes >= timings.LocalQueuePauseBytes {
							return nil
						}
						var isFlushed bool
						err := source.QueryRow(t.Context(),
							`SELECT coalesce((SELECT flush_lsn >= $1::pg_lsn
						 FROM pg_stat_replication WHERE pid = (SELECT active_pid FROM pg_replication_slots WHERE slot_name=$2)), false)`,
							targetLSN, slot).Scan(&isFlushed)
						if err != nil {
							return err
						}
						if !isFlushed {
							return fmt.Errorf("the source still has WAL the receiver has not flushed")
						}
						return nil
					},
				)
			}

			for {
				queuedBytes := team.live(t).Current.LocalQueueBytes
				if queuedBytes >= timings.LocalQueuePauseBytes {
					break
				}
				rotateWal(t, source)
				var rotatedLSN string
				require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&rotatedLSN))
				waitForSourceFlush(rotatedLSN, true)
				testwait.Until(t, "the receiver queues the rotated segment", func(context.Context) error {
					if team.live(t).Current.LocalQueueBytes <= queuedBytes {
						return fmt.Errorf("the latest segment has not reached the local queue")
					}
					return nil
				})
			}

			testwait.Until(t, "back pressure pauses the receiver while S3 is unavailable", func(context.Context) error {
				reading := team.live(t)
				if reading.Current == nil || reading.Current.IsReceiverRunning ||
					reading.Current.LocalQueueBytes < timings.LocalQueuePauseBytes {
					return fmt.Errorf("receiver has not paused")
				}
				return nil
			})

			var pauseSample streamStatistics
			generateWal(t, source, 1024*1024)
			var sourceLSN string
			require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&sourceLSN))
			var lagBeforeRecovery int64
			testwait.Until(t, "the live source probe reports the paused receiver lag", func(context.Context) error {
				reading := team.live(t)
				if reading.Current == nil || reading.Current.SlotLagBytes == nil || *reading.Current.SlotLagBytes <= 0 {
					return fmt.Errorf("no positive source slot lag")
				}
				lagBeforeRecovery = *reading.Current.SlotLagBytes
				return nil
			})

			team.recoverStorage(t)
			team.waitForReceiver(t)
			waitForSourceFlush(sourceLSN, false)
			testwait.Until(
				t,
				"the archive drains the recovered queue before the next rotation",
				func(context.Context) error {
					if team.live(t).Current.LocalQueueBytes != 0 {
						return fmt.Errorf("queue is not empty")
					}
					return nil
				},
			)

			rotateWal(t, source)
			testwait.Until(t, "the source slot lag drains after the receiver resumes", func(context.Context) error {
				reading := team.live(t)
				if reading.Current == nil || reading.Current.SlotLagBytes == nil ||
					*reading.Current.SlotLagBytes > lagBeforeRecovery/2 {
					return fmt.Errorf("slot lag has not drained")
				}
				return nil
			})
			testwait.Until(t, "a persisted sample counts the queue pause", func(context.Context) error {
				pauseSample = team.statistics(t)
				// Catching up can refill the queue faster than a loaded machine archives it, which is a
				// further real pause, so the count can be above one.
				if pauseSample.Current == nil || pauseSample.Current.ReceiverPauses == 0 {
					return fmt.Errorf("no pause sample")
				}
				return nil
			})

			assert.Zero(t, pauseSample.Current.ReceiverRestarts)

			var terminated int
			require.NoError(
				t,
				source.QueryRow(t.Context(), "SELECT count(pg_terminate_backend(active_pid)) FROM pg_replication_slots WHERE slot_name=$1 AND active_pid IS NOT NULL", slot).
					Scan(&terminated),
			)
			require.Equal(t, 1, terminated)
			testwait.Until(
				t,
				"a later persisted sample counts the failed receiver restart",
				func(context.Context) error {
					sample := team.statistics(t)
					if sample.Current == nil || sample.Current.ReceiverRestarts == 0 {
						return fmt.Errorf("no receiver restart sample")
					}
					assert.Equal(t, 1, sample.Current.ReceiverRestarts)
					return nil
				},
			)

			team.waitForReceiver(t)
			team.waitForWal(t, 1)
			assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))
		},
	)
}
