package physical

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // Each engine version owns a primary/standby fixture and its role grants.
func TestGetTimelineHistory(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			t.Run("reads verified history after promotion with provisioned credentials", func(t *testing.T) {
				primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
				primaryServer := testServer{endpoint: primary, image: image, version: pgshared.Version("")}
				standbyServer := testServer{endpoint: standby, image: image, version: pgshared.Version("")}
				user, err := primaryServer.newSource(getAdminRole()).
					CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
				require.NoError(t, err)
				role := testRole{Name: user.Username, Password: user.Password}
				primaryConn := primaryServer.connectAs(t, getAdminRole())
				standbyConn := standbyServer.connectAs(t, getAdminRole())

				privilegeProbe := createHistoryPrivilegeProbe(t, primaryServer)

				var roleLSN string
				require.NoError(
					t,
					primaryConn.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn()::text").Scan(&roleLSN),
				)
				testwait.Until(t, "the provisioned role replays on standby", func(ctx context.Context) error {
					var hasReplayed bool
					if err := standbyConn.QueryRow(ctx, "SELECT pg_last_wal_replay_lsn()>=$1::pg_lsn", roleLSN).
						Scan(&hasReplayed); err != nil {
						return err
					}

					if !hasReplayed {
						return fmt.Errorf("role WAL has not replayed")
					}

					return nil
				})

				var isPromoted bool
				require.NoError(t, standbyConn.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
				require.True(t, isPromoted)

				source := standbyServer.newSource(role)
				request := TimelineHistoryRequest{
					ExpectedSystemIdentifier: primaryServer.readSystemIdentifier(t),
					TimelineID:               2,
				}
				var sourceHistory []byte
				require.NoError(
					t,
					standbyConn.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/00000002.history')").
						Scan(&sourceHistory),
				)

				privilegesBefore := getHistoryRolePrivileges(t, standbyConn, privilegeProbe.TableName)
				require.Contains(t, privilegesBefore, role.Name)
				require.Empty(t, privilegesBefore[role.Name].Memberships)
				require.Contains(t, privilegesBefore[role.Name].FunctionPrivileges, "pg_read_binary_file(text)=false")
				require.Contains(t, privilegesBefore[role.Name].TablePrivileges, "SELECT=false")
				require.Contains(t, privilegesBefore, privilegeProbe.Bystander.Name)
				require.NotEmpty(t, privilegesBefore[privilegeProbe.Bystander.Name].Memberships)
				require.Contains(
					t,
					privilegesBefore[privilegeProbe.Bystander.Name].FunctionPrivileges,
					"pg_read_binary_file(text)=false",
				)
				require.Contains(t, privilegesBefore[privilegeProbe.Bystander.Name].TablePrivileges, "SELECT=true")

				assertPrivilegesUnchanged := func(t *testing.T) {
					t.Helper()
					require.Equal(
						t,
						privilegesBefore,
						getHistoryRolePrivileges(t, standbyConn, privilegeProbe.TableName),
					)
				}

				history, err := source.GetTimelineHistory(t.Context(), request)

				require.NoError(t, err)
				require.Equal(t, "00000002.history", history.Filename)
				require.Equal(t, sourceHistory, history.Body)
				assertRoleHasOnlyReplication(t, standbyServer, role.Name)
				require.NoError(t, validateTimelineHistory(2, history))
				assertPrivilegesUnchanged(t)

				anotherCluster := startTestServer(t, image)
				request.ExpectedSystemIdentifier = anotherCluster.readSystemIdentifier(t)
				_, err = source.GetTimelineHistory(t.Context(), request)
				require.Equal(t, pgshared.ConnErrSystemIdentifierMismatch, getConnectionTestErrorCode(t, err))
				assertPrivilegesUnchanged(t)

				request.ExpectedSystemIdentifier = ""
				_, err = source.GetTimelineHistory(t.Context(), request)
				require.ErrorIs(t, err, ErrInvalidTimelineHistory)
				assertPrivilegesUnchanged(t)

				request.ExpectedSystemIdentifier = primaryServer.readSystemIdentifier(t)
				for _, timeline := range []int{0, 1, 3, -1} {
					request.TimelineID = timeline
					_, err = source.GetTimelineHistory(t.Context(), request)
					require.ErrorIs(t, err, ErrInvalidTimelineHistory)
					assertPrivilegesUnchanged(t)
				}

				request.TimelineID = 2
				canceledCtx, cancel := context.WithCancel(t.Context())
				cancel()
				_, err = source.GetTimelineHistory(canceledCtx, request)
				require.ErrorIs(t, err, context.Canceled)
				assertRoleHasOnlyReplication(t, standbyServer, role.Name)
				assertPrivilegesUnchanged(t)

				var remainingSlots int
				require.NoError(
					t,
					standbyConn.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots").
						Scan(&remainingSlots),
				)
				require.Zero(t, remainingSlots)
			})
			t.Run("reads an earlier history after a cascading standby promotes again", func(t *testing.T) {
				_, standby, cascading := containers.StartPhysicalCascadingStandbys(t, image)
				standbyServer := testServer{endpoint: standby, image: image}
				cascadingServer := testServer{endpoint: cascading, image: image}
				standbyConn := standbyServer.connectAs(t, getAdminRole())
				cascadingConn := cascadingServer.connectAs(t, getAdminRole())

				var isPromoted bool
				require.NoError(t, standbyConn.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
				require.True(t, isPromoted)
				user, err := standbyServer.newSource(getAdminRole()).
					CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
				require.NoError(t, err)

				var roleLSN string
				require.NoError(
					t,
					standbyConn.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn()::text").Scan(&roleLSN),
				)
				testwait.Until(
					t,
					"the cascading standby follows the promoted timeline",
					func(ctx context.Context) error {
						var hasReplayed bool
						if err := cascadingConn.QueryRow(ctx, "SELECT pg_last_wal_replay_lsn()>=$1::pg_lsn", roleLSN).
							Scan(&hasReplayed); err != nil {
							return err
						}

						if !hasReplayed {
							return fmt.Errorf("the new timeline has not replayed")
						}

						return nil
					},
				)
				require.NoError(t, cascadingConn.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
				require.True(t, isPromoted)
				source := cascadingServer.newSource(testRole{Name: user.Username, Password: user.Password})
				var sourceHistory []byte
				require.NoError(
					t,
					cascadingConn.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/00000002.history')").
						Scan(&sourceHistory),
				)

				history, err := source.GetTimelineHistory(
					t.Context(),
					TimelineHistoryRequest{
						ExpectedSystemIdentifier: standbyServer.readSystemIdentifier(t),
						TimelineID:               2,
					},
				)

				require.NoError(t, err)
				require.Equal(t, sourceHistory, history.Body)

				current, err := source.GetTimelineHistory(
					t.Context(),
					TimelineHistoryRequest{
						ExpectedSystemIdentifier: standbyServer.readSystemIdentifier(t),
						TimelineID:               3,
					},
				)
				require.NoError(t, err)
				require.NoError(t, validateTimelineHistory(3, current))

				assertRoleHasOnlyReplication(t, cascadingServer, user.Username)
			})

			t.Run("returns an acquisition error when wal senders are exhausted", func(t *testing.T) {
				server := startTestServer(t, image, containers.WithMaxWalSenders(1))
				connection, err := server.openReplicationConn(t, getAdminRole())
				require.NoError(t, err)
				_, err = connection.Exec(t.Context(), "IDENTIFY_SYSTEM").ReadAll()
				require.NoError(t, err)

				_, err = server.newSource(getAdminRole()).GetTimelineHistory(t.Context(), TimelineHistoryRequest{
					ExpectedSystemIdentifier: server.readSystemIdentifier(t), TimelineID: 2,
				})

				var postgresError *pgconn.PgError
				require.ErrorAs(t, err, &postgresError)
				require.Equal(t, "53300", postgresError.Code)

				var slots int
				require.NoError(
					t,
					server.connectAs(t, getAdminRole()).
						QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots").
						Scan(&slots),
				)
				require.Zero(t, slots)
			})
		})
	}
}
