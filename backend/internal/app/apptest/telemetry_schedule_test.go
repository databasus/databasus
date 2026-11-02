package apptest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestTelemetrySchedule(t *testing.T) {
	t.Parallel()

	t.Run("retries after backoff when the receiver redirects to a successful page", func(t *testing.T) {
		t.Parallel()

		var attempts, redirectedRequests atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/accepted" {
				redirectedRequests.Add(1)
				writer.WriteHeader(http.StatusOK)
				return
			}
			attempts.Add(1)
			http.Redirect(writer, request, "/accepted", http.StatusFound)
		}))
		t.Cleanup(server.Close)

		application := apptest.New(t, apptest.WithTelemetry(apptest.TelemetrySettings{
			ReceiverURL: server.URL, IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
		}))

		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
		assert.Equal(t, int64(1), attempts.Load())
		assert.Zero(t, redirectedRequests.Load())
		application.Clock.Advance(time.Minute - time.Nanosecond)
		application.RunJobPass(t, "telemetry_ping")
		assert.Equal(t, int64(1), attempts.Load())
		application.Clock.Advance(time.Nanosecond)
		require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
		assert.Equal(t, int64(2), attempts.Load())
		assert.Zero(t, redirectedRequests.Load())
	})
	t.Run("doubles retries to the daily cap and resets them when a send succeeds", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusServiceUnavailable}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		application := apptest.New(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  server.URL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		delay := time.Minute

		for attempt := 1; attempt <= 14; attempt++ {
			require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
			require.Len(t, receiver.getReports(), attempt)
			application.Clock.Advance(delay - time.Nanosecond)
			application.RunJobPass(t, "telemetry_ping")
			require.Len(t, receiver.getReports(), attempt)
			application.Clock.Advance(time.Nanosecond)
			delay = min(delay*2, 24*time.Hour)
		}

		receiver.mutex.Lock()
		receiver.status = http.StatusCreated
		receiver.mutex.Unlock()
		application.RunJobPass(t, "telemetry_ping")
		require.Len(t, receiver.getReports(), 15)
		application.Clock.Advance(24*time.Hour - 10*time.Minute - time.Nanosecond)
		application.RunJobPass(t, "telemetry_ping")
		require.Len(t, receiver.getReports(), 15)

		receiver.mutex.Lock()
		receiver.status = http.StatusBadGateway
		receiver.mutex.Unlock()
		application.Clock.Advance(20*time.Minute + time.Nanosecond)
		require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
		application.Clock.Advance(time.Minute - time.Nanosecond)
		application.RunJobPass(t, "telemetry_ping")
		require.Len(t, receiver.getReports(), 16)
		application.Clock.Advance(time.Nanosecond)
		require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
		require.Len(t, receiver.getReports(), 17)
	})
	t.Run("waits another daily opportunity when identity cannot be loaded", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusOK}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		identityPath := filepath.Join(t.TempDir(), "identity.json")
		require.NoError(t, os.WriteFile(identityPath, []byte(`{}`), 0o600))
		application := apptest.New(
			t,
			apptest.WithTelemetry(apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath}),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		application.RunJobPass(t, "telemetry_ping")
		require.NoError(
			t,
			os.WriteFile(identityPath, []byte(`{"instanceID":"persisted","installedAt":"2020-01-01"}`), 0o600),
		)
		application.Clock.Advance(24*time.Hour - 10*time.Minute - time.Nanosecond)
		application.RunJobPass(t, "telemetry_ping")
		assert.Empty(t, receiver.getReports())
		application.Clock.Advance(20*time.Minute + time.Nanosecond)
		application.RunJobPass(t, "telemetry_ping")
		require.Len(t, receiver.getReports(), 1)
		assert.Equal(t, "persisted", receiver.getReports()[0]["instanceID"])
	})
	t.Run("records a transport failure when the receiver cannot be reached", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.NotFoundHandler())
		receiverURL := server.URL
		server.Close()
		application := apptest.New(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  receiverURL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		require.Error(t, application.RunJobPassWithContext(t.Context(), "telemetry_ping"))
	})
	t.Run("ends an in-flight send when its application context is canceled", func(t *testing.T) {
		t.Parallel()

		started := make(chan struct{})
		receiverDone := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(started)
			select {
			case <-request.Context().Done():
			case <-receiverDone:
			}
		}))
		t.Cleanup(server.Close)
		application := apptest.New(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  server.URL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)

		passContext, cancel := context.WithCancel(t.Context())
		completed := make(chan error, 1)
		go func() { completed <- application.RunJobPassWithContext(passContext, "telemetry_ping") }()
		<-started
		cancel()
		require.ErrorIs(t, <-completed, context.Canceled)
		close(receiverDone)
	})
	t.Run("bounds an unanswered HTTP send to five seconds", func(t *testing.T) {
		t.Parallel()

		receiverDone := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-receiverDone }))
		t.Cleanup(server.Close)
		defer close(receiverDone)
		application := apptest.New(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  server.URL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		startedAt := time.Now()
		err := application.RunJobPassWithContext(t.Context(), "telemetry_ping")
		require.Error(t, err)
		assert.GreaterOrEqual(t, time.Since(startedAt), 5*time.Second)
		assert.Less(t, time.Since(startedAt), 10*time.Second)
	})
}
