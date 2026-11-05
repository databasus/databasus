package healthcheck_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func holdProbe(t *testing.T, team healthcheckTeam) (<-chan net.Conn, chan<- struct{}) {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan net.Conn, 1)
	release := make(chan struct{})
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = connection.Close() }()
		accepted <- connection
		select {
		case <-release:
		case <-t.Context().Done():
		}
	}()

	address, isTCP := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCP)
	team.App.ExecuteSQL(
		t,
		"UPDATE databases_postgresql_logical SET port = ? WHERE database_id = ?",
		address.Port,
		team.Database.Id,
	)
	return accepted, release
}

func TestHealthcheckInterruption(t *testing.T) {
	t.Parallel()

	t.Run("discards a pending probe after checks are disabled", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		config := configInput(team.getConfig(t))
		config.FailedAttemptsBeforeUnavailable = 1
		team.saveConfig(t, config)

		accepted, release := holdProbe(t, team)
		finished := make(chan error, 1)
		go func() { finished <- team.App.RunJobPassWithContext(t.Context(), "healthcheck_probe") }()
		select {
		case <-accepted:
		case <-time.After(10 * time.Second):
			t.Fatal("probe did not connect")
		}

		config.IsHealthcheckEnabled = false
		team.saveConfig(t, config)
		require.Nil(t, team.getHealthStatus(t))

		close(release)
		select {
		case err := <-finished:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("disabled database probe did not finish")
		}

		team.App.WaitForNotificationDeliveries(t)
		assert.Empty(t, team.getAttempts(t))
		assert.Nil(t, team.getHealthStatus(t))
		assert.Empty(t, messages)
	})
	t.Run("waits for an interrupted probe and records nothing after cancellation", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		accepted, release := holdProbe(t, team)
		defer close(release)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		finished := make(chan error, 1)
		go func() { finished <- team.App.RunJobPassWithContext(ctx, "healthcheck_probe") }()
		select {
		case <-accepted:
		case <-time.After(10 * time.Second):
			t.Fatal("probe did not connect")
		}

		cancel()
		select {
		case err := <-finished:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("cancelled pass did not wait for its probes to end")
		}

		team.App.WaitForNotificationDeliveries(t)
		assert.Empty(t, team.getAttempts(t))
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
		assert.Empty(t, messages)
	})
	t.Run("deletes a database and records nothing when a probe fails after deletion", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		accepted, release := holdProbe(t, team)

		finished := make(chan error, 1)
		go func() { finished <- team.App.RunJobPassWithContext(t.Context(), "healthcheck_probe") }()
		select {
		case <-accepted:
		case <-time.After(10 * time.Second):
			t.Fatal("probe did not connect")
		}

		response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

		close(release)
		select {
		case err := <-finished:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("deleted database probe did not finish")
		}

		team.App.WaitForNotificationDeliveries(t)
		for _, table := range []string{"healthcheck_configs", "healthcheck_attempts"} {
			var count int64
			team.App.QuerySQL(t, &count, "SELECT count(*) FROM "+table+" WHERE database_id = ?", team.Database.Id)
			assert.Zero(t, count)
		}
		assert.Empty(t, messages)
	})
}
