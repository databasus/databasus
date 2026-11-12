package walstreaming_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestWalSourceRebuildMutations(t *testing.T) {
	t.Parallel()

	for _, mutation := range []string{"drop", "recreate", "terminate"} {
		t.Run(
			"refuses the foreign "+mutation+" session after release probes reach the original cluster",
			func(t *testing.T) {
				t.Parallel()

				original := containers.StartPhysicalPostgres(t, "postgres:17")
				foreign := containers.StartPhysicalPostgres(t, "postgres:17")
				proxy := containers.StartPostgresProxy(t, original)
				rebuildReached := make(chan struct{})
				releaseRebuild := make(chan struct{})
				mutationReached := make(chan int, 1)
				var routeOnce sync.Once
				observer := sourceRebuildObserver{
					Handler: slog.DiscardHandler, once: &sync.Once{},
					onRebuild: func() { close(rebuildReached); <-releaseRebuild },
					onRecord: func(message string) {
						targetMessage := "rebuilding released replication slot"
						switch mutation {
						case "recreate":
							targetMessage = "replication slot dropped"
						case "terminate":
							targetMessage = "wal receiver did not release its slot; checking backend ownership"
						}

						if message == targetMessage {
							routeOnce.Do(func() {
								lastConnectionID := getLastProxyConnectionID(proxy)
								proxy.RouteNext(foreign)
								mutationReached <- lastConnectionID
							})
						}
					},
				}
				fixture := prepareSourceRebuild(t, sourceRebuildSpec{
					proxy: proxy, observer: observer, original: original, foreign: foreign,
				})
				var releaseOnce sync.Once
				release := func() { releaseOnce.Do(func() { close(releaseRebuild) }) }
				t.Cleanup(release)

				switch mutation {
				case "recreate":
					_, err := fixture.foreignConnection.Exec(
						t.Context(),
						"SELECT pg_drop_replication_slot($1)",
						fixture.slotName,
					)
					require.NoError(t, err)
				case "terminate":
					consumer, err := pgconn.Connect(t.Context(), fmt.Sprintf(
						"host=%s port=%d user=testuser password=testpassword dbname=postgres replication=true sslmode=disable application_name=databasus_wal_receiver_foreign options='-c wal_sender_timeout=0'",
						foreign.Host,
						foreign.Port,
					))
					require.NoError(t, err)
					t.Cleanup(func() { _ = consumer.Close(context.WithoutCancel(t.Context())) })
					consumer.Frontend().Send(&pgproto3.Query{String: fmt.Sprintf(
						"START_REPLICATION SLOT %s PHYSICAL %s", fixture.slotName, fixture.foreignRestartLSN,
					)})
					require.NoError(t, consumer.Frontend().Flush())
					response, err := consumer.ReceiveMessage(t.Context())
					require.NoError(t, err)
					require.IsType(t, &pgproto3.CopyBothResponse{}, response)
				}

				var foreignStateBefore string
				require.NoError(t, fixture.foreignConnection.QueryRow(
					t.Context(),
					"SELECT COALESCE((SELECT concat(restart_lsn,':',COALESCE(active_pid::text,'')) FROM pg_replication_slots WHERE slot_name=$1),'absent')",
					fixture.slotName,
				).Scan(&foreignStateBefore))

				testwait.Until(t, "the original slot observation starts a rebuild", func(context.Context) error {
					select {
					case <-rebuildReached:
						return nil
					default:
						return fmt.Errorf("rebuild not reached")
					}
				})

				if mutation != "terminate" {
					require.NoError(t, fixture.consumer.Close(t.Context()))
					testwait.Until(
						t,
						"the original consumer releases the slot before rebuild probing",
						func(ctx context.Context) error {
							var isActive bool
							if err := fixture.originalConnection.QueryRow(
								ctx,
								"SELECT active FROM pg_replication_slots WHERE slot_name=$1",
								fixture.slotName,
							).Scan(&isActive); err != nil {
								return err
							}

							if isActive {
								return fmt.Errorf("original slot remains active")
							}

							return nil
						},
					)
				}

				firstRebuildConnectionID := getLastProxyConnectionID(proxy)
				release()

				lastProbeConnectionID := 0
				testwait.Until(t, "the rebuild reaches the selected mutation connection", func(context.Context) error {
					select {
					case lastProbeConnectionID = <-mutationReached:
						return nil
					default:
						return fmt.Errorf("mutation stage not reached")
					}
				})
				testwait.Until(t, "the selected foreign mutation session closes", func(context.Context) error {
					for _, event := range proxy.GetEvents() {
						if event.ConnectionID > lastProbeConnectionID && event.Destination == foreign &&
							event.Kind == "closed" {
							return nil
						}
					}

					return fmt.Errorf("foreign mutation session has not closed")
				})

				hasOriginalReleaseProbe := false
				hasForeignIdentity := false
				hasOriginalDrop := false
				for _, event := range proxy.GetEvents() {
					if event.ConnectionID > firstRebuildConnectionID && event.ConnectionID <= lastProbeConnectionID &&
						event.Destination == original && strings.Contains(event.Command, "FROM pg_replication_slots s") {
						hasOriginalReleaseProbe = true
					}

					if event.ConnectionID > firstRebuildConnectionID && event.Destination == original &&
						strings.Contains(event.Command, "pg_drop_replication_slot") {
						hasOriginalDrop = true
					}

					if event.ConnectionID > lastProbeConnectionID && event.Destination == foreign {
						if event.Kind == "identified" && !event.IsReplication {
							hasForeignIdentity = true
						}

						for _, forbidden := range []string{"pg_drop_replication_slot", "pg_create_physical_replication_slot", "pg_terminate_backend"} {
							require.NotContains(t, event.Command, forbidden)
						}
					}
				}

				require.True(t, hasOriginalReleaseProbe)
				require.True(t, hasForeignIdentity)
				require.Equal(t, mutation == "recreate", hasOriginalDrop)

				fixture.team.waitForIdentityRefusal(t)
				var foreignStateAfter string
				require.NoError(t, fixture.foreignConnection.QueryRow(
					t.Context(),
					"SELECT COALESCE((SELECT concat(restart_lsn,':',COALESCE(active_pid::text,'')) FROM pg_replication_slots WHERE slot_name=$1),'absent')",
					fixture.slotName,
				).Scan(&foreignStateAfter))
				require.Equal(t, foreignStateBefore, foreignStateAfter)

				var hasFullRequest bool
				fixture.team.App.QuerySQL(
					t,
					&hasFullRequest,
					"SELECT force_full_requested_at IS NOT NULL FROM backups_configs_physical WHERE database_id=?",
					fixture.team.Database.Id,
				)
				require.False(t, hasFullRequest)
			},
		)
	}
}

func getLastProxyConnectionID(proxy *containers.PostgresProxy) int {
	lastConnectionID := 0
	for _, event := range proxy.GetEvents() {
		if event.ConnectionID > lastConnectionID {
			lastConnectionID = event.ConnectionID
		}
	}

	return lastConnectionID
}
