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

// The scenario must watch 150 seconds of real samples, which is longer than the default wait.
const idleSamplesLimit = 5 * time.Minute

func TestIdlePhysicalWalSource(t *testing.T) {
	t.Parallel()

	t.Run(
		"keeps the same receiver and slot across 150 seconds of samples when the source stays idle",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
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
			var receiverPID int
			testwait.Until(t, "pg_receivewal attaches to its slot", func(ctx context.Context) error {
				return source.QueryRow(ctx, "SELECT active_pid FROM pg_replication_slots WHERE slot_name=$1 AND active_pid IN (SELECT pid FROM pg_stat_activity WHERE application_name LIKE 'databasus%')", slot).
					Scan(&receiverPID)
			})

			startedAt := time.Now().UTC()
			testwait.Within(idleSamplesLimit).Until(
				t,
				"samples cover 150 seconds of an idle source",
				func(ctx context.Context) error {
					var activePID int
					require.NoError(
						t,
						source.QueryRow(ctx, "SELECT active_pid FROM pg_replication_slots WHERE slot_name=$1", slot).
							Scan(&activePID),
					)
					require.Equal(t, receiverPID, activePID)

					statistics := team.statistics(t)
					if statistics.Current == nil ||
						statistics.Current.SampledAt.Before(startedAt.Add(150*time.Second)) {
						return fmt.Errorf("the idle observation window is not complete")
					}

					assert.Zero(t, statistics.Current.ReceiverRestarts)
					assert.Zero(t, statistics.Current.ReceiverPauses)

					return nil
				},
			)
		},
	)
}
