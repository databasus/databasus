package walstreaming_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

func TestWalForcedRotation(t *testing.T) {
	t.Parallel()

	t.Run(
		"archives a rarely written segment without a manual switch when forced rotation is enabled",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			team.enable(t)
			team.waitForReceiver(t)
			generateWal(t, team.openSource(t), 4096)
			team.waitForWal(t, 1)
		},
	)
	t.Run(
		"reports refused rotation through the webhook and statistics while explicit source rotations still archive WAL",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			roleName := team.useReplicationRole(t)

			inbox := &webhookInbox{}
			team.attachWebhook(t, inbox)
			team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
			team.waitForReceiver(t)

			source := team.openSource(t)
			_, err := source.Exec(
				t.Context(),
				"REVOKE EXECUTE ON FUNCTION pg_switch_wal() FROM "+pgx.Identifier{roleName}.Sanitize(),
			)
			require.NoError(t, err)
			generateWal(t, source, 4096)

			testwait.Until(t, "a chain risk webhook reports refused rotation", func(context.Context) error {
				for _, message := range inbox.getMessages() {
					if strings.Contains(message, "WAL_ROTATION_DENIED") {
						return nil
					}
				}
				return fmt.Errorf("no rotation refusal notification")
			})
			testwait.Until(t, "the stored sample reports the permanent rotation refusal", func(context.Context) error {
				statistics := team.statistics(t)
				if statistics.Current == nil || !statistics.Current.IsForcedRotationDenied {
					return fmt.Errorf("no rotation refusal sample")
				}
				return nil
			})

			rotateWal(t, source)
			team.waitForWal(t, 1)
			assert.True(t, team.live(t).IsStreamerRunning)
		},
	)
}

func TestWalArchiveStaleness(t *testing.T) {
	t.Parallel()

	t.Run("alerts while a writing source cannot archive and resumes after S3 recovers", func(t *testing.T) {
		t.Parallel()

		timings := backups.ProductionWalStreamTimings()
		timings.ForcedRotationInterval = time.Second
		timings.ArchiveStalenessThreshold = 2 * time.Second

		team := newWalTeam(t, walTeamSpec{Timings: &timings})
		team.useUnavailableStorage(t)

		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox)
		team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
		team.waitForReceiver(t)
		source := team.openSource(t)
		testwait.Until(t, "the stale archive sends its chain risk notification", func(context.Context) error {
			generateWal(t, source, 4096)
			for _, message := range inbox.getMessages() {
				if strings.Contains(message, "WAL_ARCHIVE_STALE") {
					return nil
				}
			}
			return fmt.Errorf("no stale archive notification")
		})

		statistics := team.statistics(t)
		require.NotNil(t, statistics.Current)
		assert.NotNil(t, statistics.Current.ArchiveStalledSince)

		team.recoverStorage(t)
		team.waitForWal(t, 1)
	})
}
