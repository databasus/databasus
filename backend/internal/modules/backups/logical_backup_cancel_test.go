package backups_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

var errBastionNotReached = errors.New("the backup has not reached the SSH bastion")

//nolint:tparallel // The subtests share one source server.
func TestCancelLogicalBackup(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 202, records the running backup as canceled and deletes its file", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)
		cancelledFileName := getBackupFileName(t, tm, runningBackup.Id)

		response := requestLogicalBackupCancel(t, tm.owner, runningBackup.Id)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		assert.Equal(t, runningBackup.Id, response.JSON202.Id)
		waitForRestorableBackupStatus(t, tm.owner, runningBackup.Id, apiclient.LogicalBackupStatusCANCELED)
		assert.Zero(t, countRunningLogicalBackups(t, tm.owner, source.databaseID))

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, cancelledFileName))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, cancelledFileName+".metadata"))
		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM storages_pending_file_deletions WHERE file_name IN (?, ?)",
			cancelledFileName, cancelledFileName+".metadata"))
	})

	t.Run("keeps the canceled status and size 0 after a cancel while the dump streams", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup := startQuietDumpBackup(t, tm, server, source)
		moveClockUntilProgressRefresh(t, tm, runningBackup.Id)

		response := requestLogicalBackupCancel(t, tm.owner, runningBackup.Id)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		cancelledBackup := waitForRestorableBackupStatus(t, tm.owner, runningBackup.Id,
			apiclient.LogicalBackupStatusCANCELED)
		assert.Zero(t, cancelledBackup.SizeMb)
	})

	t.Run("records canceled when the user cancels during setup", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		failureWebhook := attachFailureWebhook(t, tm, server, source)
		silentBastion := routeThroughSilentBastion(t, tm, source)
		settingUpBackup := startRestorableBackup(t, tm.owner, source.databaseID)
		silentBastion.waitForConnection(t)

		response := requestLogicalBackupCancel(t, tm.owner, settingUpBackup.Id)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		waitForRestorableBackupStatus(t, tm.owner, settingUpBackup.Id, apiclient.LogicalBackupStatusCANCELED)
		tm.app.WaitForNotificationDeliveries(t)
		assert.Zero(t, failureWebhook.getRequestCount())

		readDatabase, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), source.databaseID)
		require.NoError(t, err)
		require.NotNil(t, readDatabase.JSON200, string(readDatabase.Body))
		assert.Nil(t, readDatabase.JSON200.LastBackupErrorMessage)
	})

	t.Run("records the shutdown message when the application stops during setup", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		failureWebhook := attachFailureWebhook(t, tm, server, source)
		silentBastion := routeThroughSilentBastion(t, tm, source)
		settingUpBackup := startRestorableBackup(t, tm.owner, source.databaseID)
		silentBastion.waitForConnection(t)

		tm.app.RestartModules(t)

		failedBackup := waitForRestorableBackupStatus(
			t,
			tm.owner,
			settingUpBackup.Id,
			apiclient.LogicalBackupStatusFAILED,
		)
		require.NotNil(t, failedBackup.FailMessage)
		assert.Equal(t, "backup interrupted by application shutdown", *failedBackup.FailMessage)
		tm.app.WaitForNotificationDeliveries(t)
		assert.Equal(t, int64(1), failureWebhook.getRequestCount())
	})

	t.Run("records the connection error when the bastion refuses the connection", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		failureWebhook := attachFailureWebhook(t, tm, server, source)
		routeThroughBastionPort(t, tm, source, getClosedPort(t))

		startedBackup := startRestorableBackup(t, tm.owner, source.databaseID)

		failedBackup := waitForRestorableBackupStatus(
			t,
			tm.owner,
			startedBackup.Id,
			apiclient.LogicalBackupStatusFAILED,
		)
		require.NotNil(t, failedBackup.FailMessage)
		assert.Contains(t, *failedBackup.FailMessage, "connection refused")
		tm.app.WaitForNotificationDeliveries(t)
		assert.Equal(t, int64(1), failureWebhook.getRequestCount())
	})

	t.Run("returns 409 not_running when the backup has completed and keeps it completed", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		response := requestLogicalBackupCancel(t, tm.owner, database.backup.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeNotRunning)
		assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, getLogicalBackupStatus(t, tm.owner, database.backup.Id))
	})

	t.Run("returns 404 not_found for a backup that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestLogicalBackupCancel(t, tm.owner, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

// silentBastion completes the TCP connect and then says nothing, as a blackholing middlebox does.
type silentBastion struct {
	connectionCount *atomic.Int64
}

func (b silentBastion) waitForConnection(t *testing.T) {
	t.Helper()

	testwait.Until(t, "the backup connects to the SSH bastion", func(context.Context) error {
		if b.connectionCount.Load() == 0 {
			return errBastionNotReached
		}

		return nil
	})
}

func routeThroughSilentBastion(t *testing.T, tm team, source backupSourceDatabase) silentBastion {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	bastion := silentBastion{connectionCount: &atomic.Int64{}}

	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}

			t.Cleanup(func() { _ = connection.Close() })
			bastion.connectionCount.Add(1)
		}
	}()

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)
	routeThroughBastionPort(t, tm, source, listenerAddress.Port)

	return bastion
}

// routeThroughBastionPort turns the tunnel on by SQL, because saving it through the API would itself
// connect to the bastion.
func routeThroughBastionPort(t *testing.T, tm team, source backupSourceDatabase, bastionPort int) {
	t.Helper()

	tm.app.ExecuteSQL(t, `UPDATE databases_postgresql_logical
		SET ssh_is_enabled = TRUE, ssh_host = '127.0.0.1', ssh_port = ?, ssh_username = 'backup', ssh_auth_type = 'PASSWORD'
		WHERE database_id = ?`, bastionPort, source.databaseID)
}

func getClosedPort(t *testing.T) int {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)
	require.NoError(t, listener.Close())

	return listenerAddress.Port
}

type failureWebhook struct {
	requestCount *atomic.Int64
}

func (w failureWebhook) getRequestCount() int64 {
	return w.requestCount.Load()
}

func attachFailureWebhook(t *testing.T, tm team, server restoreServer, source backupSourceDatabase) failureWebhook {
	t.Helper()

	webhook := failureWebhook{requestCount: &atomic.Int64{}}
	endpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		webhook.requestCount.Add(1)
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(endpoint.Close)

	var notifierSettings apiclient.NotifierSettings
	require.NoError(t, notifierSettings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         endpoint.URL,
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		BodyTemplate:       nil,
		Headers:            []apiclient.WebhookHeader{},
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
	}))
	notifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, notifierSettings)

	response, err := tm.owner.Client.UpdateDatabaseWithResponse(
		t.Context(),
		source.databaseID,
		apiclient.DatabaseUpdate{
			Name:        "Database with a failure webhook",
			NotifierIds: []uuid.UUID{notifier.Id},
			Settings:    server.newDatabaseSettings(t, source.sourceName),
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return webhook
}
