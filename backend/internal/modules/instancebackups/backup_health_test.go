package instancebackups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const incidentJob = "instancebackups_incidents"

//nolint:tparallel // Each scenario moves its own application's clock between passes.
func TestTrackInstanceBackupHealth(t *testing.T) {
	t.Parallel()

	t.Run("keeps the first overdue deadline of a five-minute schedule that never backs up", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.setClockAt(t, 11, 58)
		owner.selectScheduledDestinations(t, everyFiveMinutes(), storageID)
		owner.backUp(t)

		owner.setClockAt(t, 12, 0)
		owner.App.RunJobPass(t, incidentJob)
		owner.setClockAt(t, 12, 7)
		owner.App.RunJobPass(t, incidentJob)
		owner.selectScheduledDestinations(t, apiclient.Schedule{Type: apiclient.ScheduleTypeHOURLY}, storageID)
		owner.setClockAt(t, 12, 14)
		owner.App.RunJobPass(t, incidentJob)
		statusBeforeDeadline := getStatus(t, owner.RootAdmin)
		owner.setClockAt(t, 12, 15)
		owner.App.RunJobPass(t, incidentJob)
		statusAtDeadline := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, statusBeforeDeadline.Health)
		require.Len(t, statusBeforeDeadline.Destinations, 1)
		require.NotNil(t, statusBeforeDeadline.Destinations[0].OverdueAt)
		assert.Equal(t, owner.getClockAt(12, 15), statusBeforeDeadline.Destinations[0].OverdueAt.UTC())
		assert.Nil(t, statusBeforeDeadline.Incident)
		assert.Equal(t, apiclient.InstanceBackupHealthOVERDUE, statusAtDeadline.Health)
		require.NotNil(t, statusAtDeadline.Incident)
		assert.Equal(t, apiclient.InstanceBackupIncidentKindOVERDUE, statusAtDeadline.Incident.Kind)
	})

	t.Run("reports overdue when the first copy misses 15 minutes after the first destination", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.setClockAt(t, 12, 0)
		owner.selectDestinations(t, storageID)

		owner.setClockAt(t, 12, 14)
		owner.App.RunJobPass(t, incidentJob)
		statusBeforeDeadline := getStatus(t, owner.RootAdmin)
		owner.setClockAt(t, 12, 15)
		owner.App.RunJobPass(t, incidentJob)
		statusAtDeadline := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupHealthINITIALIZING, statusBeforeDeadline.Health)
		assert.Nil(t, statusBeforeDeadline.Incident)
		assert.Equal(t, apiclient.InstanceBackupHealthOVERDUE, statusAtDeadline.Health)
		require.Len(t, statusAtDeadline.Destinations, 1)
		require.NotNil(t, statusAtDeadline.Destinations[0].OverdueAt)
		assert.Equal(t, owner.getClockAt(12, 15), statusAtDeadline.Destinations[0].OverdueAt.UTC())
		require.NotNil(t, statusAtDeadline.Incident)
		assert.Equal(t, apiclient.InstanceBackupIncidentKindOVERDUE, statusAtDeadline.Incident.Kind)
	})

	t.Run("covers several occurrences with a fresh snapshot but not with a stale late copy", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		owner.setClockAt(t, 11, 50)
		owner.selectScheduledDestinations(t, everyFiveMinutes(), namespace.StorageID)
		owner.backUp(t)
		owner.setClockAt(t, 11, 52)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		staleBackup := owner.startBackup(t)
		owner.App.RunJobPass(t, productionJob)
		owner.App.RunJobPass(t, deliveryJob)

		owner.setClockAt(t, 12, 0)
		owner.App.RunJobPass(t, incidentJob)
		namespace.MakeAvailable(t, owner.RootAdmin)
		owner.setClockAt(t, 12, 1)
		owner.App.RunJobPass(t, deliveryJob)
		statusAfterStaleCopy := getStatus(t, owner.RootAdmin)
		owner.setClockAt(t, 12, 12)
		owner.startBackup(t)
		owner.App.RunJobPass(t, productionJob)
		owner.App.RunJobPass(t, deliveryJob)
		owner.setClockAt(t, 12, 16)
		owner.App.RunJobPass(t, incidentJob)
		statusAfterFreshCopy := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupStatusValueCOMPLETED, owner.getBackup(t, staleBackup.Id).Status)
		require.Len(t, statusAfterStaleCopy.Destinations, 1)
		require.NotNil(t, statusAfterStaleCopy.Destinations[0].OverdueAt)
		assert.Equal(t, owner.getClockAt(12, 10), statusAfterStaleCopy.Destinations[0].OverdueAt.UTC())
		require.Len(t, statusAfterFreshCopy.Destinations, 1)
		require.NotNil(t, statusAfterFreshCopy.Destinations[0].OverdueAt)
		assert.Equal(t, owner.getClockAt(12, 30), statusAfterFreshCopy.Destinations[0].OverdueAt.UTC())
		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, statusAfterFreshCopy.Health)
		assert.Nil(t, statusAfterFreshCopy.Incident)
	})

	t.Run("reports partial protection when one of two destinations fails", func(t *testing.T) {
		owner := newInstanceOwner(t)
		localStorageID := owner.createLocalStorage(t, "Local")
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.selectDestinations(t, localStorageID, namespace.StorageID)
		owner.backUp(t)

		owner.App.RunJobPass(t, incidentJob)
		status := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupHealthPARTIAL, status.Health)
		require.NotNil(t, status.Incident)
		assert.Equal(t, apiclient.InstanceBackupIncidentKindPARTIAL, status.Incident.Kind)
		destinationHealths := map[uuid.UUID]apiclient.InstanceBackupHealth{}
		for _, destination := range status.Destinations {
			destinationHealths[destination.StorageId] = destination.Health
		}
		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, destinationHealths[localStorageID])
		assert.Equal(t, apiclient.InstanceBackupHealthFAILED, destinationHealths[namespace.StorageID])
	})

	t.Run("reports stopped protection at once and keeps it while resuming until a fresh copy", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		owner.backUp(t)
		owner.deferReminder(t)

		owner.selectDestinations(t)
		stoppedStatus := getStatus(t, owner.RootAdmin)
		owner.selectDestinations(t, storageID)
		owner.App.RunJobPass(t, incidentJob)
		resumingStatus := getStatus(t, owner.RootAdmin)
		owner.backUp(t)
		owner.App.RunJobPass(t, incidentJob)
		protectedStatus := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupHealthSTOPPED, stoppedStatus.Health)
		require.NotNil(t, stoppedStatus.Incident)
		assert.Equal(t, apiclient.InstanceBackupIncidentKindSTOPPED, stoppedStatus.Incident.Kind)
		assert.Equal(t, apiclient.InstanceBackupHealthRESUMING, resumingStatus.Health)
		require.NotNil(t, resumingStatus.Incident)
		assert.Equal(t, stoppedStatus.Incident.Id, resumingStatus.Incident.Id)
		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, protectedStatus.Health)
		assert.Nil(t, protectedStatus.Incident)
	})

	t.Run("opens no incident when removing the last destination cannot be saved", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		settings := owner.selectDestinations(t, storageID)
		owner.backUp(t)
		settings.StorageIds = []uuid.UUID{}
		settings.RetentionDays = 0

		response, err := owner.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
		require.NoError(t, err)
		owner.App.RunJobPass(t, incidentJob)
		status := getStatus(t, owner.RootAdmin)

		assert.Equal(t, http.StatusBadRequest, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, status.Health)
		assert.Nil(t, status.Incident)
		assert.Equal(t, []uuid.UUID{storageID}, owner.getSettings(t).StorageIds)
	})

	t.Run("opens no incident on an installation that never selected a destination", func(t *testing.T) {
		owner := newInstanceOwner(t)

		owner.App.Clock.Advance(2 * time.Hour)
		owner.App.RunJobPass(t, incidentJob)
		status := getStatus(t, owner.RootAdmin)

		assert.Equal(t, apiclient.InstanceBackupHealthNOTCONFIGURED, status.Health)
		assert.Nil(t, status.Incident)
	})
}

func everyFiveMinutes() apiclient.Schedule {
	return apiclient.Schedule{
		Type: apiclient.ScheduleTypeCRON, CronExpression: new("*/5 * * * *"),
		DayOfMonth: nil, TimeOfDay: nil, Weekday: nil,
	}
}

// getClockAt is the time of day on the day after the test started, so moving the clock there
// never goes back before the rows the test created.
func (o instanceOwner) getClockAt(hour, minute int) time.Time {
	return o.testDay.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

func (o instanceOwner) setClockAt(t *testing.T, hour, minute int) {
	t.Helper()

	o.App.Clock.Set(o.getClockAt(hour, minute))
}

func (o instanceOwner) selectScheduledDestinations(
	t *testing.T,
	destinationSchedule apiclient.Schedule,
	storageIDs ...uuid.UUID,
) {
	t.Helper()

	settings := o.getSettings(t)
	settings.Schedule = destinationSchedule
	settings.StorageIds = append([]uuid.UUID{}, storageIDs...)

	response, err := o.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
}

func (o instanceOwner) deferReminder(t *testing.T) {
	t.Helper()

	response, err := o.RootAdmin.Client.UpdateInstanceBackupReminderWithResponse(t.Context(),
		apiclient.InstanceBackupReminderUpdate{IsDeferred: true})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}
