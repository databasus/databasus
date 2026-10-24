package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownPathHandler(t *testing.T) {
	t.Parallel()

	staticDir := filepath.Join(t.TempDir(), "ui", "build")
	require.NoError(t, os.MkdirAll(staticDir, 0o755))

	const frontendPage = "<html>frontend</html>"
	require.NoError(t, os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(frontendPage), 0o600))

	gin.SetMode(gin.TestMode)
	ginApp := gin.New()
	ginApp.NoRoute(newUnknownPathHandler(staticDir))

	t.Run("returns JSON 404 when the API path does not exist", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		ginApp.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/no-such-route", nil))

		assert.Equal(t, http.StatusNotFound, response.Code)

		var errorBody map[string]string
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &errorBody))
		assert.Equal(t, "not found", errorBody["error"])
	})

	t.Run("returns the frontend page when a frontend route has no file", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		ginApp.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/callback", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, frontendPage, response.Body.String())
	})
}

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("carries the idle limit and no whole-request read or write deadline", func(t *testing.T) {
		t.Parallel()

		server := newHTTPServer(http.NotFoundHandler())

		assert.Equal(t, 120*time.Second, server.IdleTimeout)
		assert.Zero(t, server.ReadTimeout)
		assert.Zero(t, server.WriteTimeout)
	})

	t.Run("closes the connection when the request line stays unfinished past the header limit", func(t *testing.T) {
		t.Parallel()

		server := newHTTPServer(http.NotFoundHandler())
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		go func() { _ = server.Serve(listener) }()
		t.Cleanup(func() { _ = server.Close() })

		connection, err := net.Dial("tcp", listener.Addr().String())
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
