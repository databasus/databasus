package physical

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // The protocol stages share the promoted cluster and provisioned role for each version.
func TestGetTimelineHistoryCancellation(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			_, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
			server := testServer{endpoint: standby, image: image}
			administrator := server.connectAs(t, getAdminRole())
			var isPromoted bool
			require.NoError(t, administrator.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
			require.True(t, isPromoted)
			user, err := server.newSource(getAdminRole()).CreateReplicationOnlyUser(t.Context(), newTestLogger(t))
			require.NoError(t, err)
			role := testRole{Name: user.Username, Password: user.Password}
			request := TimelineHistoryRequest{ExpectedSystemIdentifier: server.readSystemIdentifier(t), TimelineID: 2}
			privilegeProbe := createHistoryPrivilegeProbe(t, server)

			for _, command := range []string{"IDENTIFY_SYSTEM", "TIMELINE_HISTORY 2"} {
				t.Run(
					"releases the connection when cancellation interrupts a withheld "+command+" response",
					func(t *testing.T) {
						proxy := holdReplicationResponse(t, replicationProxySpec{Source: standby, Command: command})
						settings := server.getSettings(role)
						settings.Host = proxy.endpoint.Host
						settings.Port = proxy.endpoint.Port
						source := NewSource(settings)
						ctx, cancel := context.WithCancel(t.Context())
						defer cancel()
						acquisitionFinished := make(chan error, 1)
						privilegesBefore := getHistoryRolePrivileges(t, administrator, privilegeProbe.TableName)

						go func() {
							_, err := source.GetTimelineHistory(ctx, request)
							acquisitionFinished <- err
						}()

						testwait.Until(
							t,
							"the proxy withholds a real PostgreSQL acquisition response",
							func(context.Context) error {
								select {
								case <-proxy.responseHeld:
									return nil
								case <-proxy.done:
									t.Fatalf(
										"replication proxy ended before withholding a response: %v",
										<-proxy.failure,
									)
									return nil
								default:
									return fmt.Errorf("the acquisition response has not reached the barrier")
								}
							},
						)
						var senderCount int
						require.NoError(
							t,
							administrator.QueryRow(t.Context(), "SELECT count(*) FROM pg_stat_activity WHERE backend_type='walsender' AND usename=$1", user.Username).
								Scan(&senderCount),
						)
						require.Equal(t, 1, senderCount)

						cancel()

						testwait.Until(
							t,
							"the in-flight acquisition returns cancellation without receiving the withheld response",
							func(context.Context) error {
								select {
								case acquisitionErr := <-acquisitionFinished:
									require.ErrorIs(t, acquisitionErr, context.Canceled)
									return nil
								default:
									return fmt.Errorf("the canceled acquisition is still waiting")
								}
							},
						)

						testwait.Until(
							t,
							"the canceled acquisition closes its proxy connection",
							func(context.Context) error {
								select {
								case <-proxy.clientClosed:
									return nil
								default:
									return fmt.Errorf("the client connection is still open")
								}
							},
						)

						testwait.Until(
							t,
							"PostgreSQL releases the canceled acquisition WAL sender",
							func(ctx context.Context) error {
								if err := administrator.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE backend_type='walsender' AND usename=$1", user.Username).
									Scan(&senderCount); err != nil {
									return err
								}

								if senderCount != 0 {
									return fmt.Errorf("%d WAL senders remain", senderCount)
								}

								return nil
							},
						)

						testwait.Until(
							t,
							"the response proxy stops after client cancellation",
							func(context.Context) error {
								select {
								case <-proxy.done:
									return nil
								default:
									return fmt.Errorf("the proxy is still serving the canceled connection")
								}
							},
						)

						assertRoleHasOnlyReplication(t, server, user.Username)
						require.Equal(
							t,
							privilegesBefore,
							getHistoryRolePrivileges(t, administrator, privilegeProbe.TableName),
						)
					},
				)
			}
		})
	}
}
