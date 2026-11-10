package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/testdb"
)

func TestTelemetryShutdown(t *testing.T) {
	t.Run("cancels delivery and starts no further sends when the application stops", func(t *testing.T) {
		started := make(chan struct{}, 1)
		canceled := make(chan struct{}, 1)
		release := make(chan struct{})
		var attempts atomic.Int64
		receiver := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			_, _ = io.Copy(io.Discard, request.Body)
			attempts.Add(1)
			started <- struct{}{}
			select {
			case <-request.Context().Done():
				canceled <- struct{}{}
			case <-release:
			}
		}))
		t.Cleanup(receiver.Close)
		defer close(release)

		environment := newTestEnvironment(t, testdb.Open(t))
		manualClock := clock.NewManual(time.Now().UTC())
		environment.Clock = manualClock
		environment.Config.IsDisableAnonymousTelemetry = false
		environment.Config.TelemetryInstancePath = filepath.Join(t.TempDir(), "instance.json")
		environment.ExternalEndpoints.TelemetryReceiverURL = receiver.URL
		application, err := New(t.Context(), environment)
		require.NoError(t, err)
		t.Cleanup(application.Close)
		application.clientToolsChecker = clienttools.NewChecker(nil)
		require.NoError(t, application.RunJobPass(t.Context(), "telemetry_ping"))
		manualClock.Advance(time.Minute)

		serveContext, stop := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)
		serveDone := make(chan struct{})
		go func() {
			serveErrors <- application.Serve(serveContext, "127.0.0.1:0")
			close(serveDone)
		}()
		t.Cleanup(func() {
			stop()
			select {
			case <-serveDone:
			case <-time.After(10 * time.Second):
				t.Error("application did not finish shutdown")
			}
		})
		select {
		case <-started:
		case <-time.After(10 * time.Second):
			t.Fatal("telemetry delivery did not start")
		}
		stop()

		select {
		case err := <-serveErrors:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("application did not stop during telemetry delivery")
		}

		waitContext, cancelWait := context.WithTimeout(t.Context(), time.Second)
		defer cancelWait()
		require.NoError(t, application.runner.Wait(waitContext))
		select {
		case <-canceled:
		case <-time.After(time.Second):
			t.Fatal("receiver did not observe delivery cancellation")
		}

		hasTelemetrySnapshot := false
		for _, job := range application.runner.GetSnapshots() {
			if job.Name == "telemetry_ping" {
				hasTelemetrySnapshot = true
				assert.False(t, job.IsRunning)
				assert.Equal(t, background.PassCanceled, job.LastOutcome)
				require.ErrorIs(t, job.LastError, context.Canceled)
			}
		}
		assert.True(t, hasTelemetrySnapshot)
		manualClock.Advance(48 * time.Hour)
		assert.Zero(t, application.backgroundGroup.GetRunningCount())
		assert.Equal(t, int64(1), attempts.Load())
	})
}
