package walstreaming_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type sourceFailureBarrier struct {
	slog.Handler
	reached chan struct{}
	release chan struct{}
	once    *sync.Once
}

func (barrier sourceFailureBarrier) Handle(_ context.Context, record slog.Record) error {
	if record.Message == "wal streamer stopping with fatal error" {
		barrier.once.Do(func() {
			close(barrier.reached)
			<-barrier.release
		})
	}

	return nil
}

func (barrier sourceFailureBarrier) WithAttrs(attributes []slog.Attr) slog.Handler {
	barrier.Handler = barrier.Handler.WithAttrs(attributes)
	return barrier
}

func (barrier sourceFailureBarrier) WithGroup(name string) slog.Handler {
	barrier.Handler = barrier.Handler.WithGroup(name)
	return barrier
}

func TestWalSourceConcurrentDisable(t *testing.T) {
	t.Parallel()

	t.Run("retains the terminal refusal when streaming is disabled before persistence", func(t *testing.T) {
		t.Parallel()

		original := containers.StartPhysicalPostgres(t, "postgres:17")
		foreign := containers.StartPhysicalPostgres(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, original)
		barrier := sourceFailureBarrier{
			Handler: slog.DiscardHandler,
			reached: make(chan struct{}),
			release: make(chan struct{}),
			once:    &sync.Once{},
		}
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint, LogSink: barrier})
		var releaseOnce sync.Once
		release := func() { releaseOnce.Do(func() { close(barrier.release) }) }
		t.Cleanup(release)

		proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: foreign})
		team.enable(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		testwait.Until(
			t,
			"the receiver has refused B before the manager persists the cause",
			func(context.Context) error {
				select {
				case <-barrier.reached:
					return nil
				default:
					return fmt.Errorf("terminal refusal not reached")
				}
			},
		)

		team.disable(t)

		var retainedClaims int
		team.App.QuerySQL(t, &retainedClaims,
			"SELECT count(*) FROM backups_physical_wal_streamers WHERE database_id=? AND error_reason IS NULL",
			team.Database.Id)
		require.Equal(t, 1, retainedClaims, "disabling must retain a claim whose terminal cause is not yet persisted")

		release()
		testwait.Until(t, "the disabled stream retains the completed identity refusal", func(context.Context) error {
			var reason string
			team.App.QuerySQL(
				t,
				&reason,
				"SELECT COALESCE(error_reason,'') FROM backups_physical_wal_streamers WHERE database_id=?",
				team.Database.Id,
			)

			if reason != "SYSTEM_IDENTIFIER_MISMATCH" || team.live(t).IsStreamerRunning {
				return fmt.Errorf("refusal completion pending")
			}

			return nil
		})

		require.Equal(t, "NOT_STREAMING", team.statistics(t).Health.Level)

		team.App.RestartModules(t)
		team.requirePersistedIdentityRefusal(t)

		team.enable(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForIdentityRefusal(t)

		proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: original})
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForReceiver(t)
	})
}
