package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gin's mode is process-wide, so it is set once before the parallel tests start.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func TestHandleUnknownPath(t *testing.T) {
	t.Parallel()

	staticDirectory := filepath.Join(t.TempDir(), "ui", "build")
	require.NoError(t, os.MkdirAll(staticDirectory, 0o755))

	const frontendPage = "<html>frontend</html>"
	require.NoError(t, os.WriteFile(filepath.Join(staticDirectory, "index.html"), []byte(frontendPage), 0o600))

	ginEngine := gin.New()
	ginEngine.NoRoute(HandleUnknownPath(staticDirectory))

	t.Run("returns Problem 404 when the API path does not exist", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		ginEngine.ServeHTTP(response, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/no-such-route", nil,
		))

		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
		assert.Equal(t, "not_found", decodeProblemCode(t, response))
	})

	t.Run("returns the frontend page when a frontend route has no file", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		ginEngine.ServeHTTP(
			response,
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/callback", nil),
		)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, frontendPage, response.Body.String())
	})
}

const guardedContract = `
openapi: 3.0.3
info:
  title: Guarded contract
  version: 1.0.0
servers:
  - url: /api/v1
paths:
  /system/version:
    get:
      operationId: getVersion
      responses:
        "200":
          description: The version
`

func TestRequireContractOperation(t *testing.T) {
	t.Parallel()

	t.Run("runs the handler when the contract describes the route", func(t *testing.T) {
		t.Parallel()
		guarded := newGuardedEngine(t, "/system/version")

		response := httptest.NewRecorder()
		guarded.engine.ServeHTTP(response, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/system/version", nil,
		))

		assert.Equal(t, http.StatusNoContent, response.Code)
		assert.Equal(t, int32(1), guarded.handledRequests.Load())
	})

	t.Run("answers 500 without running the handler when the contract lacks the route", func(t *testing.T) {
		t.Parallel()
		guarded := newGuardedEngine(t, "/system/settings")

		response := httptest.NewRecorder()
		guarded.engine.ServeHTTP(response, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/system/settings", nil,
		))

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Equal(t, "internal", decodeProblemCode(t, response))
		assert.Zero(t, guarded.handledRequests.Load())
		assert.Contains(t, guarded.logOutput.String(), `"method":"GET"`)
		assert.Contains(t, guarded.logOutput.String(), `"path":"/api/v1/system/settings"`)
	})
}

type guardedEngine struct {
	engine          *gin.Engine
	handledRequests *atomic.Int32
	logOutput       *bytes.Buffer
}

// newGuardedEngine mounts one GET route under the contract guard; the handler counts its calls.
func newGuardedEngine(t *testing.T, routePath string) guardedEngine {
	t.Helper()

	guarded := guardedEngine{engine: gin.New(), handledRequests: &atomic.Int32{}, logOutput: &bytes.Buffer{}}

	guard, err := RequireContractOperation(
		[]byte(guardedContract),
		slog.New(slog.NewJSONHandler(guarded.logOutput, nil)),
	)
	require.NoError(t, err)

	guarded.engine.Group("/api/v1", guard).GET(routePath, func(ginContext *gin.Context) {
		guarded.handledRequests.Add(1)
		ginContext.Status(http.StatusNoContent)
	})

	return guarded
}

func TestRecover(t *testing.T) {
	t.Parallel()

	t.Run("answers 500 when a handler panics", func(t *testing.T) {
		t.Parallel()

		ginEngine := gin.New()
		ginEngine.Use(Recover(slog.New(slog.DiscardHandler)))
		ginEngine.GET("/api/v1/broken", func(*gin.Context) { panic("broken handler") })

		response := httptest.NewRecorder()
		ginEngine.ServeHTTP(
			response,
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/broken", nil),
		)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
		assert.Equal(t, "internal", decodeProblemCode(t, response))
		assert.Contains(t, response.Body.String(), `"detail":"an internal error occurred"`)
		assert.NotContains(t, response.Body.String(), "broken handler")
	})
}

func TestListen(t *testing.T) {
	t.Parallel()

	t.Run("returns an error when the port is busy", func(t *testing.T) {
		t.Parallel()

		var listenConfig net.ListenConfig
		busyListener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = busyListener.Close() })

		_, err = Listen(t.Context(), busyListener.Addr().String(), http.NotFoundHandler())

		require.Error(t, err)
	})

	t.Run("carries the idle limit and no whole-request read or write deadline", func(t *testing.T) {
		t.Parallel()

		server := newHTTPServer(http.NotFoundHandler())

		assert.Equal(t, 120*time.Second, server.IdleTimeout)
		assert.Zero(t, server.ReadTimeout)
		assert.Zero(t, server.WriteTimeout)
	})

	t.Run("closes the connection when the request line stays unfinished past the header limit", func(t *testing.T) {
		t.Parallel()

		server, err := Listen(t.Context(), "127.0.0.1:0", http.NotFoundHandler())
		require.NoError(t, err)

		go func() { _ = server.Serve() }()
		t.Cleanup(func() { _ = server.Shutdown(t.Context()) })

		var dialer net.Dialer
		connection, err := dialer.DialContext(t.Context(), "tcp", server.GetAddress())
		require.NoError(t, err)
		t.Cleanup(func() { _ = connection.Close() })

		_, err = connection.Write([]byte("GET /api/v1/syst"))
		require.NoError(t, err)

		writtenAt := time.Now()
		require.NoError(t, connection.SetReadDeadline(writtenAt.Add(20*time.Second)))

		_, err = io.ReadAll(connection)
		closedAfter := time.Since(writtenAt)

		require.NoError(t, err, "server must close the connection before the client read deadline")
		assert.GreaterOrEqual(t, closedAfter, 10*time.Second)
	})
}

func decodeProblemCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()

	var problemBody struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problemBody))

	return problemBody.Code
}
