package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/testwait"
)

func TestServe(t *testing.T) {
	t.Run("answers requests until its context ends and then stops every loop in time", func(t *testing.T) {
		testApp := newTestApp(t)
		address := reserveFreeAddress(t)
		serveCtx, stopServing := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)

		go func() { serveErrors <- testApp.Serve(serveCtx, address) }()

		requireStatus(t, "http://"+address+"/api/v1/system/version", http.StatusOK)
		requireStatus(t, "http://"+address+"/api/v1/no-such-route", http.StatusNotFound)

		stopServing()
		stoppedAt := time.Now()

		select {
		case err := <-serveErrors:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("Serve did not return after its context ended")
		}

		assert.Less(t, time.Since(stoppedAt), 8*time.Second)
		assert.Zero(t, testApp.backgroundGroup.GetRunningCount())
	})

	t.Run("returns after the running job pass ends when its context ends", func(t *testing.T) {
		testApp := newTestApp(t)
		var isPassStarted, isPassFinished atomic.Bool
		testApp.runner = background.NewRunner(
			background.RunnerDeps{Logger: testApp.logger, Clock: testApp.environment.Clock},
			background.Job{
				Name:     "slow_cleanup",
				Interval: time.Hour,
				Run: func(ctx context.Context, _ time.Time) error {
					isPassStarted.Store(true)
					<-ctx.Done()
					<-time.After(200 * time.Millisecond)
					isPassFinished.Store(true)

					return nil
				},
			},
		)
		serveCtx, stopServing := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)

		address := reserveFreeAddress(t)

		go func() { serveErrors <- testApp.Serve(serveCtx, address) }()

		testwait.Until(t, "the job pass to start", func(context.Context) error {
			if !isPassStarted.Load() {
				return errors.New("the pass has not started")
			}

			return nil
		})

		stopServing()

		require.NoError(t, <-serveErrors)
		assert.True(t, isPassFinished.Load())
	})

	t.Run("returns an error before any loop starts when the port is busy", func(t *testing.T) {
		testApp := newTestApp(t)

		var listenConfig net.ListenConfig
		busyListener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = busyListener.Close() })

		err = testApp.Serve(t.Context(), busyListener.Addr().String())

		require.Error(t, err)
		assert.Nil(t, testApp.backgroundGroup)
	})
}

func TestRouter(t *testing.T) {
	swaggerPaths := []string{"/api/v1/docs/swagger/index.html", "/api/v1/docs/swagger/doc.json"}

	for _, envMode := range []config.EnvMode{config.EnvModeProduction, config.EnvModeDevelopment} {
		t.Run("answers Problem 404 for the API documentation in "+string(envMode)+" mode", func(t *testing.T) {
			testApp := newTestApp(t)
			testApp.environment.Config.EnvMode = envMode
			handler, err := testApp.NewHandler()
			require.NoError(t, err)

			for _, swaggerPath := range swaggerPaths {
				response := httptest.NewRecorder()
				handler.ServeHTTP(
					response,
					httptest.NewRequestWithContext(t.Context(), http.MethodGet, swaggerPath, nil),
				)

				assert.Equal(t, http.StatusNotFound, response.Code, swaggerPath)
				assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"), swaggerPath)

				var problemBody struct {
					Code string `json:"code"`
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problemBody), swaggerPath)
				assert.Equal(t, "not_found", problemBody.Code, swaggerPath)
			}
		})
	}
}

func reserveFreeAddress(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

// requireStatus waits until the server answers, since Serve opens its port in the test's goroutine.
func requireStatus(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	var answeredStatus int

	testwait.Until(t, "GET "+url+" to answer", func(ctx context.Context) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}

		_ = response.Body.Close()
		answeredStatus = response.StatusCode

		return nil
	})

	require.Equal(t, expectedStatus, answeredStatus, url)
}
