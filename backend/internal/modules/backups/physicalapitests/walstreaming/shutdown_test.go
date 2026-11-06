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

	t.Run("releases fresh ownership on restart and resumes the same slot", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		team.App.WaitForPhysicalBackups(t)

		source := team.openSource(t)
		initialSlot := getActiveWalSlot(t, source)

		initialStatistics := team.statistics(t)
		require.NotNil(t, initialStatistics.Streamer.SessionStartedAt)

		team.App.PausePhysicalJobs(t)
		// Future freshness prevents heartbeat expiry from disguising a missing shutdown release.
		heartbeat := time.Now().UTC().Add(time.Hour)
		team.App.ExecuteSQL(t,
			"UPDATE backups_physical_wal_streamers SET last_heartbeat_at=? WHERE database_id=?",
			heartbeat, team.Database.Id,
		)

		team.App.RestartModules(t)
		var releasedStatus string
		team.App.QuerySQL(t, &releasedStatus,
			"SELECT status FROM backups_physical_wal_streamers WHERE database_id=? AND last_heartbeat_at>?",
			team.Database.Id, time.Now().UTC(),
		)
		require.Equal(t, "FAILED", releasedStatus)

		team.App.ResumePhysicalJobs(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")

		testwait.Until(
			t,
			"the released claim resumes the same replication slot and records a new session",
			func(context.Context) error {
				statistics := team.statistics(t)
				if statistics.Streamer.State != "RUNNING" ||
					statistics.Streamer.SessionStartedAt == nil ||
					!statistics.Streamer.SessionStartedAt.After(*initialStatistics.Streamer.SessionStartedAt) {
					return fmt.Errorf("no new running WAL session")
				}
				var activeSlot string
				if err := source.QueryRow(t.Context(), "SELECT slot_name FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%' AND active").
					Scan(&activeSlot); err != nil {
					return err
				}
				assert.Equal(t, initialSlot, activeSlot)
				var slotCount int
				require.NoError(
					t,
					source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%'").
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
