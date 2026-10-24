package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	env_utils "databasus-backend/internal/util/env"
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

	for _, envMode := range []env_utils.EnvMode{env_utils.EnvModeProduction, env_utils.EnvModeDevelopment} {
		t.Run("answers JSON 404 for the API documentation in "+string(envMode)+" mode", func(t *testing.T) {
			testApp := newTestApp(t)
			testApp.environment.Config.EnvMode = envMode
			router := testApp.newRouter()

			for _, swaggerPath := range swaggerPaths {
				response := httptest.NewRecorder()
				router.ServeHTTP(
					response,
					httptest.NewRequestWithContext(t.Context(), http.MethodGet, swaggerPath, nil),
				)

				assert.Equal(t, http.StatusNotFound, response.Code, swaggerPath)

				var errorBody map[string]string
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &errorBody), swaggerPath)
				assert.Equal(t, "not found", errorBody["error"], swaggerPath)
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

// requireStatus polls until the server answers, since Serve opens its port in the test's goroutine.
// lib/testwait does not exist yet, so the poll is spelled out here.
func requireStatus(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	pollTicker := time.NewTicker(50 * time.Millisecond)

	defer pollTicker.Stop()

	for {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
			require.Equal(t, expectedStatus, response.StatusCode, url)

			return
		}

		var opError *net.OpError
		if !errors.As(err, &opError) || time.Now().After(deadline) {
			require.NoError(t, err, url)
		}

		<-pollTicker.C
	}
}
