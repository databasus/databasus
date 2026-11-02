package physical

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestResolveLiveTimelineID(t *testing.T) {
	t.Parallel()
	promotedPrimaryWalFilename := "000000020000000000000005"

	tests := []struct {
		name               string
		sources            clusterIdentitySources
		resolvedTimelineID int
	}{
		{
			name: "prefers the wal filename when promotion is ahead of the control file",
			sources: clusterIdentitySources{
				CurrentWalFilename:    &promotedPrimaryWalFilename,
				ControlFileTimelineID: 1,
			},
			resolvedTimelineID: 2,
		},
		{
			name:               "uses the control file when the cluster is in recovery",
			sources:            clusterIdentitySources{ControlFileTimelineID: 3},
			resolvedTimelineID: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			timelineID, err := resolveLiveTimelineID(test.sources)

			require.NoError(t, err)
			require.Equal(t, test.resolvedTimelineID, timelineID)
		})
	}
}

func TestResolveLiveTimelineIDUnreadable(t *testing.T) {
	t.Parallel()
	t.Run("returns an error when the wal filename is unreadable", func(t *testing.T) {
		t.Parallel()

		unreadableWalFilename := "not-a-wal-file"

		_, err := resolveLiveTimelineID(clusterIdentitySources{
			CurrentWalFilename:    &unreadableWalFilename,
			ControlFileTimelineID: 1,
		})

		require.Error(
			t,
			err,
			"a source naming its WAL file in a shape we cannot read must not fall back to a stale timeline",
		)
	})
}

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestReadLiveTimeline(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			t.Run("reports the live timeline after promotion before the control file catches up", func(t *testing.T) {
				primary, standby := containers.StartPhysicalPrimaryWithStandby(
					t,
					image,
					containers.WithCheckpointsSpreadOverAnHour(),
				)
				primaryServer := testServer{endpoint: primary, image: image, version: ""}
				standbyServer := testServer{endpoint: standby, image: image, version: ""}

				primaryConn := primaryServer.connectAs(t, getAdminRole())
				standbyConn := standbyServer.connectAs(t, getAdminRole())
				execStatements(
					t,
					primaryConn,
					"CREATE TABLE promotion_ballast AS SELECT generate_series(1, 300000) AS n",
				)
				testwait.Until(t, "ballast is replayed on standby", func(ctx context.Context) error {
					var relation *string
					if err := standbyConn.QueryRow(ctx, "SELECT to_regclass('promotion_ballast')::text").
						Scan(&relation); err != nil {
						return err
					}
					if relation == nil {
						return fmt.Errorf("ballast is not replayed")
					}
					return nil
				})

				inspection := standbyServer.openInspection(t, getAdminRole())
				slotName := newSlotName()
				require.NoError(t, inspection.CreateSlot(t.Context(), slotName))
				standbySlot, err := inspection.GetSlot(t.Context(), slotName)
				require.NoError(t, err)
				require.NotNil(t, standbySlot)
				require.Equal(t, "physical", standbySlot.Type)
				require.Nil(t, standbySlot.CurrentLSN)
				require.Nil(t, standbySlot.LagBytes)
				require.Nil(t, standbySlot.SafeWalSizeBytes)
				require.NoError(
					t,
					standbyServer.newSource(getAdminRole()).DropSlot(t.Context(), newTestLogger(t), slotName),
				)

				var isPromoted bool
				require.NoError(
					t,
					standbyConn.QueryRow(t.Context(), "SELECT pg_promote(wait => true, wait_seconds => 60)").
						Scan(&isPromoted),
				)
				require.True(t, isPromoted)
				var controlFileTimelineID int
				require.NoError(
					t,
					standbyConn.QueryRow(t.Context(), "SELECT timeline_id FROM pg_control_checkpoint()").
						Scan(&controlFileTimelineID),
				)
				require.Equal(t, 1, controlFileTimelineID)

				liveTimeline, systemIdentifier, err := inspection.ReadLiveTimeline(t.Context())
				require.NoError(t, err)
				require.Equal(t, 2, liveTimeline)
				require.Equal(t, primaryServer.readSystemIdentifier(t), systemIdentifier)
				history, err := inspection.GetTimelineHistory(t.Context(), "00000002.history")
				require.NoError(t, err)
				require.Contains(t, string(history), "no recovery target specified")
			})
		})
	}
	t.Run("reads identity without rejecting an unsupported server version", func(t *testing.T) {
		server := testServer{
			endpoint: containers.StartPostgres(t, "postgres:16"),
			image:    "postgres:16",
			version:  "",
		}

		inspection := server.openInspection(t, getAdminRole())
		_, err := inspection.InspectCluster(t.Context())
		require.ErrorIs(t, err, ErrUnsupportedVersion)

		timeline, systemIdentifier, err := inspection.ReadLiveTimeline(t.Context())
		require.NoError(t, err)
		require.Equal(t, 1, timeline)
		require.Equal(t, server.readSystemIdentifier(t), systemIdentifier)
	})
}
