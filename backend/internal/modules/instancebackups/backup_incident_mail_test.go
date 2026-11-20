package instancebackups_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/mailer"
	mailpit "databasus-backend/internal/lib/mailpittest"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // Each scenario moves its own application's clock between passes.
func TestMailInstanceBackupIncident(t *testing.T) {
	t.Parallel()

	t.Run("mails root admin once per incident across passes and restarts until the daily reminder", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.stopProtection(t, "Archive")

		owner.App.RunJobPass(t, incidentJob)
		owner.App.RunJobPass(t, incidentJob)
		owner.App.RestartModules(t)
		owner.App.Clock.Advance(23 * time.Hour)
		owner.App.RunJobPass(t, incidentJob)
		messagesWithinDay := owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email)
		owner.App.Clock.Advance(time.Hour)
		owner.App.RunJobPass(t, incidentJob)
		messagesAfterDay := owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email)

		require.Len(t, messagesWithinDay, 1)
		assert.Contains(t, messagesWithinDay[0].Subject, "the last destination was removed")
		assert.Len(t, messagesAfterDay, 2)
		status := getStatus(t, owner.RootAdmin)
		require.NotNil(t, status.Incident)
		assert.Equal(t, apiclient.InstanceBackupMailStatusSENT, status.Incident.MailStatus)
		assert.NotNil(t, status.Incident.LastMailedAt)
	})

	t.Run("mails once before the reminder while delivery retries, a kind change and a restart follow",
		func(t *testing.T) {
			owner := newInstanceOwner(t)
			failingNamespace := apptest.CreateStorageNamespace(t, owner.RootAdmin,
				apptest.CreateWorkspace(t, owner.RootAdmin, "Failing").Id)
			recoveringNamespace := apptest.CreateStorageNamespace(t, owner.RootAdmin,
				apptest.CreateWorkspace(t, owner.RootAdmin, "Recovering").Id)
			failingNamespace.MakeUnavailable(t, owner.RootAdmin)
			recoveringNamespace.MakeUnavailable(t, owner.RootAdmin)
			owner.selectDestinations(t, failingNamespace.StorageID, recoveringNamespace.StorageID)
			owner.backUp(t)
			owner.App.RunJobPass(t, incidentJob)
			failedStatus := getStatus(t, owner.RootAdmin)

			owner.App.Clock.Advance(time.Minute)
			owner.App.RunJobPass(t, deliveryJob)
			owner.App.RunJobPass(t, incidentJob)
			recoveringNamespace.MakeAvailable(t, owner.RootAdmin)
			owner.App.Clock.Advance(2 * time.Minute)
			owner.App.RunJobPass(t, deliveryJob)
			owner.App.RunJobPass(t, incidentJob)
			partialStatus := getStatus(t, owner.RootAdmin)
			owner.App.RestartModules(t)
			owner.App.Clock.Advance(20 * time.Hour)
			owner.App.RunJobPass(t, deliveryJob)
			owner.App.RunJobPass(t, incidentJob)
			restartedStatus := getStatus(t, owner.RootAdmin)

			require.NotNil(t, failedStatus.Incident)
			assert.Equal(t, apiclient.InstanceBackupIncidentKindFAILED, failedStatus.Incident.Kind)
			require.NotNil(t, partialStatus.Incident)
			assert.Equal(t, apiclient.InstanceBackupIncidentKindPARTIAL, partialStatus.Incident.Kind)
			assert.Equal(t, failedStatus.Incident.Id, partialStatus.Incident.Id)
			require.NotNil(t, restartedStatus.Incident)
			assert.Equal(t, failedStatus.Incident.Id, restartedStatus.Incident.Id)
			messages := owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email)
			require.Len(t, messages, 1)
			assert.Contains(t, messages[0].Subject, "no destination received the latest Databasus backup")
		})

	t.Run("keeps the stored copy successful and the incident visible when the mail server refuses", func(t *testing.T) {
		owner := newInstanceOwner(t)
		localStorageID := owner.createLocalStorage(t, "Local")
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.App.Mail.RefuseSends()
		owner.selectDestinations(t, localStorageID, namespace.StorageID)
		backup := owner.backUp(t)

		owner.App.RunJobPass(t, incidentJob)
		status := getStatus(t, owner.RootAdmin)

		assert.Len(t, owner.App.Mail.GetRefusedMessagesTo(owner.RootAdmin.Email), 1)
		storedCopy := getCopy(t, owner.getBackup(t, backup.Id), localStorageID)
		assert.Equal(t, apiclient.InstanceBackupCopyStatusSTORED, storedCopy.Status)
		assert.True(t, storedCopy.IsDownloadable)
		destinationHealths := map[uuid.UUID]apiclient.InstanceBackupHealth{}
		for _, destination := range status.Destinations {
			destinationHealths[destination.StorageId] = destination.Health
		}
		assert.Equal(t, apiclient.InstanceBackupHealthHEALTHY, destinationHealths[localStorageID])
		require.NotNil(t, status.Incident)
		assert.Equal(t, apiclient.InstanceBackupIncidentKindPARTIAL, status.Incident.Kind)
		assert.Equal(t, apiclient.InstanceBackupMailStatusRETRYING, status.Incident.MailStatus)
		require.NotNil(t, status.Incident.LastMailError)
		assert.NotEmpty(t, *status.Incident.LastMailError)
	})

	t.Run("mails a recurrence after recovery at once as a new incident", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.stopProtection(t, "Archive")
		owner.App.RunJobPass(t, incidentJob)
		firstStatus := getStatus(t, owner.RootAdmin)
		owner.selectDestinations(t, storageID)
		owner.backUp(t)
		owner.App.RunJobPass(t, incidentJob)
		recoveredStatus := getStatus(t, owner.RootAdmin)

		owner.selectDestinations(t)
		owner.App.RunJobPass(t, incidentJob)
		recurredStatus := getStatus(t, owner.RootAdmin)

		require.NotNil(t, firstStatus.Incident)
		assert.Nil(t, recoveredStatus.Incident)
		require.NotNil(t, recurredStatus.Incident)
		assert.NotEqual(t, firstStatus.Incident.Id, recurredStatus.Incident.Id)
		assert.Equal(t, apiclient.InstanceBackupMailStatusSENT, recurredStatus.Incident.MailStatus)
		assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 2)
	})

	t.Run("delivers the incident email through the system mail server", func(t *testing.T) {
		mailpitEndpoint := containers.StartMailpit(t)
		mailpitClient := mailpit.NewClient(fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port))
		owner := newInstanceOwner(t, apptest.WithInstanceMailServer(mailer.Settings{
			Host:                 mailpitEndpoint.SMTP.Host,
			Port:                 mailpitEndpoint.SMTP.Port,
			Security:             mailer.SecurityNone,
			Username:             "",
			Password:             "",
			From:                 "databasus@databasus.local",
			HeloName:             "",
			IsInsecureSkipVerify: false,
		}))
		owner.stopProtection(t, "Archive")

		owner.App.RunJobPass(t, incidentJob)

		messages, err := mailpitClient.FetchMessages(t.Context())
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0].Subject, "the last destination was removed")
		require.Len(t, messages[0].To, 1)
		assert.Equal(t, owner.RootAdmin.Email, messages[0].To[0].Address)
		assert.Equal(t, apiclient.InstanceBackupMailStatusSENT, getStatus(t, owner.RootAdmin).Incident.MailStatus)
	})

	t.Run("names the condition and destinations without secrets", func(t *testing.T) {
		owner := newInstanceOwner(t)
		namespace := apptest.CreateStorageNamespace(
			t,
			owner.RootAdmin,
			apptest.CreateWorkspace(t, owner.RootAdmin, "Remote").Id,
		)
		namespace.MakeUnavailable(t, owner.RootAdmin)
		owner.selectDestinations(t, namespace.StorageID)
		backup := owner.backUp(t)

		owner.App.RunJobPass(t, incidentJob)

		messages := owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email)
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0].Subject, "no destination received the latest Databasus backup")
		assert.Contains(t, messages[0].HTMLBody, getCopy(t, backup, namespace.StorageID).StorageName)
		assert.Contains(t, messages[0].HTMLBody, "Remote")
		assert.NotContains(t, messages[0].HTMLBody, namespace.Bucket)
		assert.NotContains(t, messages[0].HTMLBody, strings.TrimSpace(string(owner.App.ReadMasterKeyFile(t))))
	})

	t.Run("keeps the incident in-app without a mail server and mails it once one is configured", func(t *testing.T) {
		owner := newInstanceOwner(t, apptest.WithoutMailServer())
		owner.stopProtection(t, "Archive")

		owner.App.RunJobPass(t, incidentJob)
		statusWithoutMail := getStatus(t, owner.RootAdmin)
		owner.App.Mail.ConfigureMailServer()
		owner.App.RunJobPass(t, incidentJob)
		statusWithMail := getStatus(t, owner.RootAdmin)

		require.NotNil(t, statusWithoutMail.Incident)
		assert.Equal(t, apiclient.InstanceBackupMailStatusNOTELIGIBLE, statusWithoutMail.Incident.MailStatus)
		assert.Nil(t, statusWithoutMail.Incident.LastMailedAt)
		assert.False(t, statusWithoutMail.IsEmailConfigured)
		require.NotNil(t, statusWithMail.Incident)
		assert.Equal(t, apiclient.InstanceBackupMailStatusSENT, statusWithMail.Incident.MailStatus)
		assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
	})

	t.Run("records a refused mail and retries it only after its backoff", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.App.Mail.RefuseSends()
		owner.stopProtection(t, "Archive")

		owner.App.RunJobPass(t, incidentJob)
		owner.App.RunJobPass(t, incidentJob)
		statusAfterRefusal := getStatus(t, owner.RootAdmin)
		refusedMessages := owner.App.Mail.GetRefusedMessagesTo(owner.RootAdmin.Email)
		owner.App.Mail.AcceptSends()
		owner.App.Clock.Advance(time.Minute)
		owner.App.RunJobPass(t, incidentJob)

		assert.Len(t, refusedMessages, 1)
		require.NotNil(t, statusAfterRefusal.Incident)
		assert.Equal(t, apiclient.InstanceBackupMailStatusRETRYING, statusAfterRefusal.Incident.MailStatus)
		require.NotNil(t, statusAfterRefusal.Incident.LastMailError)
		assert.NotEmpty(t, *statusAfterRefusal.Incident.LastMailError)
		assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
		assert.Equal(t, apiclient.InstanceBackupMailStatusSENT, getStatus(t, owner.RootAdmin).Incident.MailStatus)
	})

	t.Run("mails the address root admin has when the mail is sent", func(t *testing.T) {
		owner := newInstanceOwner(t, apptest.WithoutMailServer())
		owner.stopProtection(t, "Archive")
		owner.App.RunJobPass(t, incidentJob)
		changedEmail := apptest.NewTestEmail("changed-root")
		renamed, err := owner.RootAdmin.Client.UpdateUserWithResponse(t.Context(), owner.RootAdmin.UserID,
			apiclient.UserUpdate{Name: "Owner", Email: changedEmail})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, renamed.StatusCode(), string(renamed.Body))

		owner.App.Mail.ConfigureMailServer()
		owner.App.RunJobPass(t, incidentJob)

		assert.Len(t, owner.App.Mail.GetMessagesTo(changedEmail), 1)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
	})

	t.Run("sends nothing for an incident that resolved before its mail went out", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.App.Mail.RefuseSends()
		storageID := owner.stopProtection(t, "Archive")
		owner.App.RunJobPass(t, incidentJob)

		owner.selectDestinations(t, storageID)
		owner.backUp(t)
		owner.App.Mail.AcceptSends()
		owner.App.Clock.Advance(time.Minute)
		owner.App.RunJobPass(t, incidentJob)

		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
		assert.Nil(t, getStatus(t, owner.RootAdmin).Incident)
	})
}

// stopProtection backs up to a new storage and then removes it, the last destination.
func (o instanceOwner) stopProtection(t *testing.T, storageName string) uuid.UUID {
	t.Helper()

	storageID := o.createLocalStorage(t, storageName)
	o.selectDestinations(t, storageID)
	o.backUp(t)
	o.selectDestinations(t)

	return storageID
}
