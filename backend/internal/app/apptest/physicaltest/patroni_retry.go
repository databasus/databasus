package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

type patroniRetrySpec struct {
	AttemptID uuid.UUID
	Events    *patroniScheduleEvents
}

func (source patroniSource) requireScheduledRetry(t *testing.T, spec patroniRetrySpec) uuid.UUID {
	t.Helper()

	testwait.Until(t, "the failed backup worker releases its database reservation", func(context.Context) error {
		if !spec.Events.IsRunFinished(spec.AttemptID) {
			return fmt.Errorf("backup worker %s has not finished", spec.AttemptID)
		}

		return nil
	})

	precedingEvents := len(spec.Events.GetBackups())
	source.Proxy.SetUnavailable(false)
	due := source.setRetrySchedule(t)
	connection := source.connect(t)
	testwait.Within(3*time.Minute).
		Until(t, "the recorded full-backup schedule becomes due", func(context.Context) error {
			if !sourceTime(t, connection).Before(due) {
				return nil
			}

			source.Team.App.RunJobPass(t, "backups_physical_scheduler")
			source.Team.App.RunJobPass(t, "backups_physical_request_scheduler")

			if !sourceTime(t, connection).Before(due) {
				return nil
			}

			var laterCount int
			source.Team.App.QuerySQL(
				t,
				&laterCount,
				"SELECT count(*) FROM backups_physical WHERE database_id=? AND created_at > (SELECT created_at FROM backups_physical WHERE id=?)",
				source.Team.Database.Id,
				spec.AttemptID,
			)
			require.Zero(t, laterCount, "a finished generic failure must not be retried before its schedule")
			require.Len(t, spec.Events.GetBackups(), precedingEvents)
			var hasForcedRequest bool
			source.Team.App.QuerySQL(
				t,
				&hasForcedRequest,
				"SELECT force_full_requested_at IS NOT NULL OR force_incremental_requested_at IS NOT NULL FROM backups_configs_physical WHERE database_id=?",
				source.Team.Database.Id,
			)
			require.False(t, hasForcedRequest, "slot rebuilds and manual requests must not satisfy the cadence check")

			return fmt.Errorf("schedule is due at %s", due)
		})

	source.Team.App.RunJobPass(t, "backups_physical_scheduler")
	var retry patroniScheduledBackup
	testwait.Until(t, "the scheduler dispatches the due full backup", func(context.Context) error {
		backups := spec.Events.GetBackups()
		if len(backups) == precedingEvents {
			return fmt.Errorf("no scheduled full after %s", due)
		}

		retry = backups[precedingEvents]
		require.Equal(t, "full cadence is due", retry.Reason)
		require.False(t, retry.ScheduledAt.Before(due))
		require.NotEqual(t, uuid.Nil, retry.ID)

		return nil
	})

	replacement := source.waitForCompletedFull(t, retry.ID)
	require.False(t, replacement.CreatedAt.Before(due))
	t.Logf("scheduled retry %s was due at %s and dispatched at %s", retry.ID, due, retry.ScheduledAt)

	return retry.ID
}

func (source patroniSource) setRetrySchedule(t *testing.T) time.Time {
	t.Helper()

	response, err := source.Team.Owner.Client.GetBackupConfigWithResponse(t.Context(), source.Team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	configuration := response.JSON200
	settings, err := configuration.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	connection := source.connect(t)
	now := sourceTime(t, connection)
	due := now.UTC().Truncate(time.Minute).Add(2 * time.Minute)
	expression := fmt.Sprintf("%d %d %d %d *", due.Minute(), due.Hour(), due.Day(), int(due.Month()))
	settings.FullBackupSchedule = apiclient.Schedule{Type: apiclient.ScheduleTypeCRON, CronExpression: &expression}
	require.NoError(t, configuration.Settings.FromPhysicalBackupSettings(settings))
	updated, err := source.Team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		source.Team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			Encryption:       configuration.Encryption,
			StorageId:        configuration.StorageId,
			Settings:         configuration.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updated.StatusCode(), "%s", updated.Body)

	return due
}
