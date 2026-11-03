package walstreaming_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestWalStreamShutdown(t *testing.T) {
	t.Parallel()

	t.Run("resumes the same slot within thirty seconds when the modules restart", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		team.App.WaitForPhysicalBackups(t)

		source := team.openSource(t)
		initialSlot := getActiveWalSlot(t, source)

		initialStatistics := team.statistics(t)
		require.NotNil(t, initialStatistics.Streamer.SessionStartedAt)

		team.App.RestartModules(t)
		resumeCtx, cancelResume := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancelResume()

		testwait.Until(
			t,
			"the same replication slot and its statistics resume before the old heartbeat becomes stale",
			func(context.Context) error {
				if err := resumeCtx.Err(); err != nil {
					t.Fatalf("the WAL stream did not resume within thirty seconds: %v", err)
				}
				statistics := team.statistics(t)
				if statistics.Streamer.State != "RUNNING" ||
					statistics.Streamer.SessionStartedAt == nil ||
					!statistics.Streamer.SessionStartedAt.After(*initialStatistics.Streamer.SessionStartedAt) {
					return fmt.Errorf("no new running WAL session")
				}
				var activeSlot string
				if err := source.QueryRow(resumeCtx, "SELECT slot_name FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%' AND active").
					Scan(&activeSlot); err != nil {
					return err
				}
				assert.Equal(t, initialSlot, activeSlot)
				var slotCount int
				require.NoError(
					t,
					source.QueryRow(resumeCtx, "SELECT count(*) FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%'").
						Scan(&slotCount),
				)
				assert.Equal(t, 1, slotCount)
				return nil
			},
		)
	})
	t.Run("keeps its replication slot when a manager pass follows disabling backups", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		team.App.WaitForPhysicalBackups(t)

		source := team.openSource(t)
		initialSlot := getActiveWalSlot(t, source)

		team.disable(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		testwait.Until(t, "the statistics report a stopped streamer", func(context.Context) error {
			if state := team.statistics(t).Streamer.State; state != "NOT_STARTED" {
				return fmt.Errorf("streamer state: %s", state)
			}
			return nil
		})

		var slotExists bool
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1)", initialSlot).
				Scan(&slotExists),
		)
		assert.True(t, slotExists)
	})
}
