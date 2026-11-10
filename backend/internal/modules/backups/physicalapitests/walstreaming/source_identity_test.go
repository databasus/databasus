package walstreaming_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func openIdentitySource(t *testing.T, endpoint containers.Endpoint) *pgx.Conn {
	t.Helper()
	connection, err := pgx.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=postgres sslmode=disable",
		endpoint.Host, endpoint.Port,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })
	return connection
}

func TestWalSourceIdentity(t *testing.T) {
	t.Parallel()
	for _, isReconnect := range []bool{false, true} {
		name := "refuses a foreign WAL session when SQL preflight reaches the original cluster"
		if isReconnect {
			name = "refuses a foreign WAL session after an established original stream disconnects"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			original := containers.StartPhysicalPostgres(t, "postgres:17")
			foreign := containers.StartPhysicalPostgres(t, "postgres:17")
			proxy := containers.StartPostgresProxy(t, original)
			team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
			foreignConnection := openIdentitySource(t, foreign)
			var foreignIdentity, savedIdentity, slotName string
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
					Scan(&foreignIdentity),
			)
			team.App.QuerySQL(
				t,
				&savedIdentity,
				"SELECT system_identifier FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			require.NotEqual(t, savedIdentity, foreignIdentity)
			originalIdentity := savedIdentity
			originalConnection := openIdentitySource(t, original)
			var originalTimeline, foreignTimeline int
			require.NoError(
				t,
				originalConnection.QueryRow(t.Context(), "SELECT timeline_id FROM pg_control_checkpoint()").
					Scan(&originalTimeline),
			)
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT timeline_id FROM pg_control_checkpoint()").
					Scan(&foreignTimeline),
			)
			require.Equal(t, originalTimeline, foreignTimeline)
			var originalWalFilename string
			require.NoError(
				t,
				originalConnection.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").
					Scan(&originalWalFilename),
			)
			team.App.QuerySQL(
				t,
				&slotName,
				"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			_, err := foreignConnection.Exec(
				t.Context(),
				"SELECT pg_create_physical_replication_slot($1,true)",
				slotName,
			)
			require.NoError(t, err)
			generateWal(t, foreignConnection, 32*1024*1024)
			var hasCollidingWalFilename bool
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_ls_waldir() WHERE name=$1)", originalWalFilename).
					Scan(&hasCollidingWalFilename),
			)
			require.True(t, hasCollidingWalFilename)
			team.enable(t)
			if isReconnect {
				team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
				team.waitForReceiver(t)
				testwait.Until(t, "the original source enters replication", func(context.Context) error {
					for _, event := range proxy.GetEvents() {
						if event.Kind == "streaming" && event.Destination == original {
							return nil
						}
					}
					return fmt.Errorf("original source has not entered replication")
				})
			}
			proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: foreign})
			proxy.CutConnections()
			team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
			var refusedConnectionID int
			testwait.Until(t, "the actual WAL connection reaches the independent cluster", func(context.Context) error {
				for _, event := range proxy.GetEvents() {
					if event.Kind == "identified" && event.IsReplication && event.Destination == foreign {
						require.Equal(t, foreignIdentity, event.SystemIdentifier)
						refusedConnectionID = event.ConnectionID
						return nil
					}
				}
				return fmt.Errorf("foreign source has not answered IDENTIFY_SYSTEM")
			})

			testwait.Until(t, "the identified foreign connection is refused", func(context.Context) error {
				hasRequestedReplication := false
				for _, event := range proxy.GetEvents() {
					if event.ConnectionID != refusedConnectionID {
						continue
					}
					if strings.HasPrefix(event.Command, "START_REPLICATION") {
						hasRequestedReplication = true
					}
					require.NotEqual(
						t,
						"streaming",
						event.Kind,
						"foreign server entered CopyBoth replication on connection %d despite SQL preflight reaching the original cluster",
						refusedConnectionID,
					)
					if event.Kind == "closed" {
						require.False(
							t,
							hasRequestedReplication,
							"a refused source must receive no START_REPLICATION command",
						)
						return nil
					}
				}
				return fmt.Errorf("identified foreign connection remains open")
			})

			team.waitForIdentityRefusal(t)
			team.App.QuerySQL(
				t,
				&savedIdentity,
				"SELECT system_identifier FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			require.Equal(t, originalIdentity, savedIdentity)
			hasAuthenticatedForeignSession := false
			for _, event := range proxy.GetEvents() {
				if event.ConnectionID == refusedConnectionID && event.Kind == "authenticated" {
					hasAuthenticatedForeignSession = true
				}
			}
			require.True(t, hasAuthenticatedForeignSession)
			var isForeignSlotActive bool
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT active FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&isForeignSlotActive),
			)
			require.False(t, isForeignSlotActive)
			var foreignPublications int
			team.App.QuerySQL(
				t,
				&foreignPublications,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
				team.Database.Id,
			)
			if !isReconnect {
				require.Zero(t, foreignPublications)
			}
			team.App.RestartModules(t)
			team.waitForIdentityRefusal(t)
			team.disable(t)
			team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
			require.Equal(t, "NOT_STREAMING", team.statistics(t).Health.Level)
			team.requirePersistedIdentityRefusal(t)
			proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: original})
			proxy.CutConnections()
			team.enable(t)
			team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
			team.waitForReceiver(t)
			var refusal *string
			team.App.QuerySQL(
				t,
				&refusal,
				"SELECT error_reason FROM backups_physical_wal_streamers WHERE database_id=?",
				team.Database.Id,
			)
			require.Nil(t, refusal)
			rotateWal(t, openIdentitySource(t, original))
			team.waitForWal(t, 1)
		})
	}
}

func (team *walTeam) requirePersistedIdentityRefusal(t *testing.T) {
	t.Helper()
	var reason string
	team.App.QuerySQL(
		t,
		&reason,
		"SELECT COALESCE(error_reason,'') FROM backups_physical_wal_streamers WHERE database_id=?",
		team.Database.Id,
	)
	require.Equal(t, "SYSTEM_IDENTIFIER_MISMATCH", reason)
}

func (team *walTeam) waitForIdentityRefusal(t *testing.T) {
	t.Helper()
	testwait.Until(t, "monitoring reports the persisted source identity refusal", func(context.Context) error {
		statistics := team.statistics(t)
		if statistics.Health.Level != "BROKEN" || len(statistics.Health.Reasons) != 1 ||
			statistics.Health.Reasons[0].Code != "SYSTEM_IDENTIFIER_MISMATCH" {
			return fmt.Errorf("health: %+v", statistics.Health)
		}
		reading := team.live(t)
		if reading.Current != nil && reading.Current.IsReceiverRunning {
			return fmt.Errorf("receiver still running")
		}
		return nil
	})
	team.requirePersistedIdentityRefusal(t)
}

func TestWalSourceInspectionIdentity(t *testing.T) {
	t.Parallel()
	for _, hasForeignSlot := range []bool{false, true} {
		name := "preserves a foreign slot when SQL observations reach another cluster"
		if !hasForeignSlot {
			name = "refuses slot creation when SQL reaches another cluster without the slot"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			original := containers.StartPhysicalPostgres(t, "postgres:17")
			foreign := containers.StartPhysicalPostgres(t, "postgres:17")
			proxy := containers.StartPostgresProxy(t, original)
			team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
			var slotName string
			team.App.QuerySQL(
				t,
				&slotName,
				"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			foreignConnection := openIdentitySource(t, foreign)
			if hasForeignSlot {
				_, err := foreignConnection.Exec(
					t.Context(),
					"SELECT pg_create_physical_replication_slot($1,true)",
					slotName,
				)
				require.NoError(t, err)
			}
			proxy.Route(containers.PostgresProxyRoute{SQLDestination: foreign, ReplicationDestination: foreign})
			team.enable(t)
			team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
			team.waitForIdentityRefusal(t)
			var slots int
			require.NoError(
				t,
				foreignConnection.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&slots),
			)
			if hasForeignSlot {
				require.Equal(t, 1, slots)
			} else {
				require.Zero(t, slots)
			}
			var rebuildRequests int
			team.App.QuerySQL(
				t,
				&rebuildRequests,
				"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_full_requested_at IS NOT NULL",
				team.Database.Id,
			)
			require.Zero(t, rebuildRequests)
		})
	}
}

func TestWalSourceQueuedRestart(t *testing.T) {
	t.Parallel()
	t.Run("preserves completed original WAL when restarting against a foreign cluster", func(t *testing.T) {
		t.Parallel()
		original := containers.StartPhysicalPostgres(t, "postgres:17")
		foreign := containers.StartPhysicalPostgres(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, original)
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
		team.useUnavailableStorage(t)
		team.enable(t)
		team.waitForReceiver(t)
		source := openIdentitySource(t, original)
		rotateWal(t, source)
		generateWal(t, source, 32*1024*1024)
		rotateWal(t, source)
		queueRoot := team.App.GetWalQueueFilePath(team.Database.Id, "")
		var completedFiles map[string][32]byte
		testwait.Until(
			t,
			"completed original segments remain queued after failed uploads",
			func(context.Context) error {
				completedFiles = readIdentityQueue(t, queueRoot)
				if len(completedFiles) < 2 {
					return fmt.Errorf("only %d completed segments", len(completedFiles))
				}
				return nil
			},
		)
		proxy.Route(containers.PostgresProxyRoute{SQLDestination: foreign, ReplicationDestination: foreign})
		proxy.CutConnections()
		team.App.RestartModules(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForIdentityRefusal(t)
		retainedFiles := readIdentityQueue(t, queueRoot)
		for name, checksum := range completedFiles {
			require.Equal(t, checksum, retainedFiles[name], name)
		}
		// The queue layout is an internal crash state which no HTTP operation can prepare.
		for name := range retainedFiles {
			if filepath.Dir(name) == "." {
				require.NoError(
					t,
					os.Rename(filepath.Join(queueRoot, name), filepath.Join(queueRoot, "pending-upload", name)),
				)
				break
			}
		}
		completedFiles = readIdentityQueue(t, queueRoot)
		team.App.RestartModules(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForIdentityRefusal(t)
		require.Equal(t, completedFiles, readIdentityQueue(t, queueRoot))
		team.recoverStorage(t)
		proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: original})
		proxy.CutConnections()
		barrier := proxy.HoldCommand("START_REPLICATION")
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		testwait.Until(t, "the recovered source reaches verified replication setup", func(context.Context) error {
			select {
			case <-barrier.Reached:
				return nil
			default:
				return fmt.Errorf("setup not reached")
			}
		})
		team.waitForWal(t, len(completedFiles))
		testwait.Until(
			t,
			"a new sample and queued uploads retain the unresolved identity refusal",
			func(context.Context) error {
				statistics := team.statistics(t)
				if statistics.Current == nil || statistics.Streamer.SessionStartedAt == nil ||
					statistics.Current.SampledAt.Before(*statistics.Streamer.SessionStartedAt) {
					return fmt.Errorf("no sample in the new claim")
				}
				require.Equal(t, "BROKEN", statistics.Health.Level)
				team.requirePersistedIdentityRefusal(t)
				return nil
			},
		)
		close(barrier.Release)
		team.waitForReceiver(t)
		team.waitForWal(t, len(completedFiles))
		require.NotEqual(t, "BROKEN", team.statistics(t).Health.Level)
		var reason *string
		team.App.QuerySQL(
			t,
			&reason,
			"SELECT error_reason FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		require.Nil(t, reason)
		team.call(t, apptest.PhysicalListRequest(team.Database.Id, "?limit=1000"), http.StatusOK, nil)
		assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))
	})
}

func readIdentityQueue(t *testing.T, directory string) map[string][32]byte {
	t.Helper()
	files := make(map[string][32]byte)
	require.NoError(t, filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || len(entry.Name()) != 24 {
			return nil
		}
		contents, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		files[relative] = sha256.Sum256(contents)
		return nil
	}))
	return files
}

func TestWalSourceStaleAttempt(t *testing.T) {
	t.Parallel()
	t.Run("retains a newer refusal when an older verified attempt enters replication", func(t *testing.T) {
		t.Parallel()
		original := containers.StartPhysicalPostgres(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, original)
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
		barrier := proxy.HoldCommand("START_REPLICATION")
		team.enable(t)
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		var connectionID int
		testwait.Until(t, "the older verified attempt reaches replication setup", func(context.Context) error {
			select {
			case event := <-barrier.Reached:
				connectionID = event.ConnectionID
				return nil
			default:
				return fmt.Errorf("replication setup not reached")
			}
		})
		// Claim replacement cannot be requested through HTTP; it models ownership lost to another manager.
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_streamers SET claim_id=gen_random_uuid(),error_reason='SYSTEM_IDENTIFIER_MISMATCH',failing_since=started_at WHERE database_id=?",
			team.Database.Id,
		)
		close(barrier.Release)
		testwait.Until(t, "the stale attempt receives a verified replication response", func(context.Context) error {
			for _, event := range proxy.GetEvents() {
				if event.ConnectionID == connectionID && event.Kind == "streaming" {
					return nil
				}
			}
			return fmt.Errorf("not streaming")
		})
		team.waitForReceiver(t)
		team.requirePersistedIdentityRefusal(t)
		require.Equal(t, "BROKEN", team.statistics(t).Health.Level)
	})
}

func TestWalSourceIdentityNotifications(t *testing.T) {
	t.Parallel()
	for _, isEnabled := range []bool{true, false} {
		scenario := "sends recovery guidance once when the chain failure trigger is enabled"
		if !isEnabled {
			scenario = "sends no notification when the chain failure trigger is disabled"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			original := containers.StartPhysicalPostgres(t, "postgres:17")
			foreign := containers.StartPhysicalPostgres(t, "postgres:17")
			proxy := containers.StartPostgresProxy(t, original)
			team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
			inbox := &webhookInbox{}
			team.attachWebhook(t, inbox)
			if isEnabled {
				team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
			} else {
				team.setNotifications(t)
			}
			proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: foreign})
			team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
			team.waitForIdentityRefusal(t)
			if isEnabled {
				testwait.Until(t, "the existing notifier sends source recovery guidance", func(context.Context) error {
					for _, message := range inbox.getMessages() {
						if strings.Contains(message, "SYSTEM_IDENTIFIER_MISMATCH") {
							require.Contains(t, message, "restore routing to the original cluster")
							require.NotContains(t, message, "testpassword")
							return nil
						}
					}
					return fmt.Errorf("no identity refusal notification")
				})
			}
			lastForeignConnectionID := 0
			for _, event := range proxy.GetEvents() {
				if event.Destination == foreign && event.ConnectionID > lastForeignConnectionID {
					lastForeignConnectionID = event.ConnectionID
				}
			}
			var firstClaim string
			team.App.QuerySQL(
				t,
				&firstClaim,
				"SELECT claim_id::text FROM backups_physical_wal_streamers WHERE database_id=?",
				team.Database.Id,
			)
			testwait.Until(t, "the manager retries the refused source with a new claim", func(context.Context) error {
				team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
				var nextClaim string
				team.App.QuerySQL(
					t,
					&nextClaim,
					"SELECT claim_id::text FROM backups_physical_wal_streamers WHERE database_id=?",
					team.Database.Id,
				)
				if nextClaim == firstClaim {
					return fmt.Errorf("old claim still owns the run")
				}
				return nil
			})
			testwait.Until(
				t,
				"the retry refuses B and completes its notification decision",
				func(context.Context) error {
					hasRefusedRetry := false
					for _, event := range proxy.GetEvents() {
						if event.ConnectionID > lastForeignConnectionID && event.Destination == foreign &&
							event.Kind == "identified" {
							hasRefusedRetry = true
						}
					}
					if !hasRefusedRetry || team.live(t).IsStreamerRunning {
						return fmt.Errorf("the retry has not finished")
					}
					return nil
				},
			)
			team.waitForIdentityRefusal(t)
			team.App.WaitForNotificationDeliveries(t)
			messages := inbox.getMessages()
			if isEnabled {
				require.Len(t, messages, 1)
			} else {
				require.Empty(t, messages)
			}
		})
	}
}

func TestWalSourceBackupPause(t *testing.T) {
	t.Parallel()
	t.Run("keeps backups paused until a manual full succeeds after WAL resumes", func(t *testing.T) {
		t.Parallel()
		original := containers.StartPhysicalPostgres(t, "postgres:17")
		foreign := containers.StartPhysicalPostgres(t, "postgres:17")
		proxy := containers.StartPostgresProxy(t, original)
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &proxy.Endpoint})
		team.enable(t)
		team.waitForReceiver(t)
		waitForCompletedFullBackups(t, team, 1)
		proxy.Route(containers.PostgresProxyRoute{SQLDestination: foreign, ReplicationDestination: foreign})
		proxy.CutConnections()
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForIdentityRefusal(t)
		team.call(t, apptest.PhysicalStartRequest(team.Database.Id, "FULL"), http.StatusAccepted, nil)
		team.App.RunJobPass(t, "backups_physical_request_scheduler")
		testwait.Until(t, "the backup independently records its source identity failure", func(context.Context) error {
			var failures int
			team.App.QuerySQL(
				t,
				&failures,
				"SELECT count(*) FROM backups_physical WHERE database_id=? AND kind='FULL' AND status='CHAIN_BROKEN' AND error_reason='SYSTEM_IDENTIFIER_MISMATCH'",
				team.Database.Id,
			)
			if failures == 0 {
				return fmt.Errorf("no backup identity failure")
			}
			return nil
		})
		// Historical scheduling timestamps cannot be prepared through HTTP while live jobs use the system clock.
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=created_at-interval '48 hours',completed_at=completed_at-interval '48 hours' WHERE database_id=? AND status='COMPLETED' AND kind='FULL'",
			team.Database.Id,
		)
		proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: original})
		proxy.CutConnections()
		team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
		team.waitForReceiver(t)
		var refusal *string
		team.App.QuerySQL(
			t,
			&refusal,
			"SELECT error_reason FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		require.Nil(t, refusal)
		team.App.RunJobPass(t, "backups_physical_scheduler")
		var fullCount int
		team.App.QuerySQL(
			t,
			&fullCount,
			"SELECT count(*) FROM backups_physical WHERE database_id=? AND kind='FULL'",
			team.Database.Id,
		)
		require.Equal(t, 2, fullCount, "WAL recovery must not authorize automatic backup recovery")
		team.call(t, apptest.PhysicalStartRequest(team.Database.Id, "FULL"), http.StatusAccepted, nil)
		team.App.RunJobPass(t, "backups_physical_request_scheduler")
		testwait.Until(
			t,
			"the operator-requested full completes after the original source returns",
			func(context.Context) error {
				var completedFulls int
				team.App.QuerySQL(
					t,
					&completedFulls,
					"SELECT count(*) FROM backups_physical WHERE database_id=? AND kind='FULL' AND status='COMPLETED'",
					team.Database.Id,
				)
				if completedFulls != 2 {
					return fmt.Errorf("%d completed full backups", completedFulls)
				}
				return nil
			},
		)
	})
}
