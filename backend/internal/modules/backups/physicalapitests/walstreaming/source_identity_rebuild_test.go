package walstreaming_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

type sourceRebuildObserver struct {
	slog.Handler
	onRebuild func()
	onRecord  func(string)
	once      *sync.Once
}

func (observer sourceRebuildObserver) Handle(_ context.Context, record slog.Record) error {
	if observer.onRecord != nil {
		observer.onRecord(record.Message)
	}
	if record.Message == "starting slot rebuild" && observer.onRebuild != nil {
		observer.once.Do(observer.onRebuild)
	}
	return nil
}

func (observer sourceRebuildObserver) WithAttrs(attributes []slog.Attr) slog.Handler {
	observer.Handler = observer.Handler.WithAttrs(attributes)
	return observer
}

func (observer sourceRebuildObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}

func TestWalSourceRebuildIdentity(t *testing.T) {
	t.Parallel()
	t.Run(
		"retains the identity cause when a verified probe is followed by a foreign rebuild session",
		func(t *testing.T) {
			t.Parallel()
			original := containers.StartPhysicalPostgres(t, "postgres:17")
			foreign := containers.StartPhysicalPostgres(t, "postgres:17")
			proxy := containers.StartPostgresProxy(t, original)
			observer := sourceRebuildObserver{Handler: slog.DiscardHandler, once: &sync.Once{}, onRebuild: func() {
				proxy.Route(containers.PostgresProxyRoute{SQLDestination: foreign, ReplicationDestination: foreign})
			}}
			fixture := prepareSourceRebuild(
				t,
				sourceRebuildSpec{proxy: proxy, observer: observer, original: original, foreign: foreign},
			)
			team := fixture.team
			originalConnection := fixture.originalConnection
			foreignConnection := fixture.foreignConnection
			slotName := fixture.slotName
			foreignRestartLSN := fixture.foreignRestartLSN
			team.waitForIdentityRefusal(t)
			var retainedRestartLSN string
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&retainedRestartLSN),
			)
			require.Equal(t, foreignRestartLSN, retainedRestartLSN)
			var isOriginalConsumerActive bool
			require.NoError(
				t,
				originalConnection.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='another_consumer')").
					Scan(&isOriginalConsumerActive),
			)
			require.True(t, isOriginalConsumerActive)
			var hasFullRequest bool
			team.App.QuerySQL(
				t,
				&hasFullRequest,
				"SELECT force_full_requested_at IS NOT NULL FROM backups_configs_physical WHERE database_id=?",
				team.Database.Id,
			)
			require.False(t, hasFullRequest)
			require.NotEmpty(t, readIdentityQueue(t, team.App.GetWalQueueFilePath(team.Database.Id, "")))
		},
	)
}

func TestWalSourceForeignSlotState(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"lost", "stolen"} {
		t.Run("preserves a foreign "+state+" slot when a running stream probes another cluster", func(t *testing.T) {
			t.Parallel()
			original := containers.StartPhysicalPostgres(t, "postgres:17")
			foreign := containers.StartPhysicalPostgres(t, "postgres:17", containers.WithWalKeepSize(0))
			proxy := containers.StartPostgresProxy(t, original)
			team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
			team.enable(t)
			team.waitForReceiver(t)
			var slotName string
			team.App.QuerySQL(
				t,
				&slotName,
				"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			foreignConnection := openIdentitySource(t, foreign)
			_, err := foreignConnection.Exec(
				t.Context(),
				"SELECT pg_create_physical_replication_slot($1,true)",
				slotName,
			)
			require.NoError(t, err)
			if state == "lost" {
				_, err = foreignConnection.Exec(t.Context(), "ALTER SYSTEM SET max_slot_wal_keep_size='0'")
				require.NoError(t, err)
				_, err = foreignConnection.Exec(t.Context(), "ALTER SYSTEM SET max_wal_size='32MB'")
				require.NoError(t, err)
				_, err = foreignConnection.Exec(t.Context(), "SELECT pg_reload_conf()")
				require.NoError(t, err)
				generateWal(t, foreignConnection, 64*1024*1024)
				testwait.Until(
					t,
					"the independent source reports a genuinely lost slot",
					func(ctx context.Context) error {
						_, err := foreignConnection.Exec(ctx, "CHECKPOINT")
						if err != nil {
							return err
						}
						var status string
						if err := foreignConnection.QueryRow(ctx, "SELECT wal_status FROM pg_replication_slots WHERE slot_name=$1", slotName).
							Scan(&status); err != nil {
							return err
						}
						if status != "lost" {
							generateWal(t, foreignConnection, 32*1024*1024)
							return fmt.Errorf("slot status: %s", status)
						}
						return nil
					},
				)
			} else {
				var restartLSN string
				require.NoError(
					t,
					foreignConnection.QueryRow(t.Context(), "SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1", slotName).
						Scan(&restartLSN),
				)
				consumer, err := pgconn.Connect(
					t.Context(),
					fmt.Sprintf(
						"host=%s port=%d user=testuser password=testpassword dbname=postgres replication=true sslmode=disable application_name=databasus_wal_receiver_foreign",
						foreign.Host,
						foreign.Port,
					),
				)
				require.NoError(t, err)
				t.Cleanup(func() { _ = consumer.Close(context.WithoutCancel(t.Context())) })
				consumer.Frontend().
					Send(&pgproto3.Query{String: fmt.Sprintf("START_REPLICATION SLOT %s PHYSICAL %s", slotName, restartLSN)})
				require.NoError(t, consumer.Frontend().Flush())
				response, err := consumer.ReceiveMessage(t.Context())
				require.NoError(t, err)
				require.IsType(t, &pgproto3.CopyBothResponse{}, response)
			}
			var beforeState string
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT concat(wal_status,':',COALESCE(active_pid::text,''),':',COALESCE(restart_lsn::text,'')) FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&beforeState),
			)
			proxy.Route(containers.PostgresProxyRoute{SQLDestination: foreign, ReplicationDestination: original})
			team.waitForIdentityRefusal(t)
			var afterState string
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT concat(wal_status,':',COALESCE(active_pid::text,''),':',COALESCE(restart_lsn::text,'')) FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&afterState),
			)
			require.Equal(t, beforeState, afterState)
			var hasFullRequest bool
			team.App.QuerySQL(
				t,
				&hasFullRequest,
				"SELECT force_full_requested_at IS NOT NULL FROM backups_configs_physical WHERE database_id=?",
				team.Database.Id,
			)
			require.False(t, hasFullRequest)
		})
	}
}

type sourceRebuildTeam struct {
	team               *walTeam
	originalConnection *pgx.Conn
	foreignConnection  *pgx.Conn
	slotName           string
	foreignRestartLSN  string
	consumer           *pgconn.PgConn
}

type sourceRebuildSpec struct {
	proxy    *containers.PostgresProxy
	observer slog.Handler
	original containers.Endpoint
	foreign  containers.Endpoint
}

func prepareSourceRebuild(t *testing.T, spec sourceRebuildSpec) sourceRebuildTeam {
	t.Helper()
	proxy, observer := spec.proxy, spec.observer
	original, foreign := spec.original, spec.foreign
	timings := backups.ProductionWalStreamTimings()
	timings.LocalQueuePauseBytes = 16 * 1024 * 1024
	timings.ForcedRotationInterval = 0
	team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint, Timings: &timings, LogSink: observer})
	team.useUnavailableStorage(t)
	team.enable(t)
	originalConnection := openIdentitySource(t, original)
	var slotName string
	team.App.QuerySQL(
		t,
		&slotName,
		"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
		team.Database.Id,
	)
	testwait.Until(t, "the original source reserves its WAL slot", func(ctx context.Context) error {
		var count int
		if err := originalConnection.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
			Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("slot not yet created")
		}
		return nil
	})
	generateWal(t, originalConnection, 32*1024*1024)
	rotateWal(t, originalConnection)
	testwait.Until(
		t,
		"storage backpressure releases the original receiver slot",
		func(ctx context.Context) error {
			reading := team.live(t)
			if reading.Current == nil || reading.Current.IsReceiverRunning ||
				reading.Current.LocalQueueBytes < timings.LocalQueuePauseBytes {
				return fmt.Errorf("receiver not yet paused")
			}
			var isActive bool
			if err := originalConnection.QueryRow(ctx, "SELECT active FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&isActive); err != nil {
				return err
			}
			if isActive {
				return fmt.Errorf("slot not yet released")
			}
			return nil
		},
	)
	foreignConnection := openIdentitySource(t, foreign)
	var foreignRestartLSN string
	require.NoError(
		t,
		foreignConnection.QueryRow(t.Context(), "SELECT lsn::text FROM pg_create_physical_replication_slot($1,true)", slotName).
			Scan(&foreignRestartLSN),
	)
	var originalRestartLSN string
	require.NoError(
		t,
		originalConnection.QueryRow(t.Context(), "SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1", slotName).
			Scan(&originalRestartLSN),
	)
	consumer, err := pgconn.Connect(
		t.Context(),
		fmt.Sprintf(
			"host=%s port=%d user=testuser password=testpassword dbname=postgres replication=true sslmode=disable application_name=another_consumer options='-c wal_sender_timeout=0'",
			original.Host,
			original.Port,
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close(context.WithoutCancel(t.Context())) })
	consumer.Frontend().
		Send(&pgproto3.Query{String: fmt.Sprintf("START_REPLICATION SLOT %s PHYSICAL %s", slotName, originalRestartLSN)})
	require.NoError(t, consumer.Frontend().Flush())
	response, err := consumer.ReceiveMessage(t.Context())
	require.NoError(t, err)
	require.IsType(t, &pgproto3.CopyBothResponse{}, response)
	return sourceRebuildTeam{
		team: team, originalConnection: originalConnection, foreignConnection: foreignConnection,
		slotName: slotName, foreignRestartLSN: foreignRestartLSN, consumer: consumer,
	}
}
