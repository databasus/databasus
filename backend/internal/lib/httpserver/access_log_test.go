package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/logging"
)

func TestLogAccess(t *testing.T) {
	t.Parallel()

	t.Run("logs the method, path and status at info when the request succeeds", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		router := newAccessLoggedRouter(capturer)
		router.GET("/api/v1/storages", func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })

		response := serveAccessLogged(t, router, "/api/v1/storages")

		accessRecord := capturer.requireSingle(t)
		assert.Equal(t, slog.LevelInfo, accessRecord.level)
		assert.Equal(t, "request served", accessRecord.message)
		assert.Equal(t, "GET", accessRecord.attributes["method"])
		assert.Equal(t, "/api/v1/storages", accessRecord.attributes["path"])
		assert.Equal(t, "200", accessRecord.attributes["status"])
		assert.Equal(t, "/api/v1/storages", accessRecord.attributes["route"])
		assert.Equal(t, response.Header().Get(RequestIDHeader), accessRecord.attributes["request_id"])
	})

	t.Run("logs at error with the handler's error when the handler fails", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		router := newAccessLoggedRouter(capturer)
		router.GET("/api/v1/databases/:databaseId/crash", func(ginContext *gin.Context) {
			_ = ginContext.Error(errors.New("storage backend unreachable"))
			ginContext.Status(http.StatusInternalServerError)
		})

		serveAccessLogged(t, router, "/api/v1/databases/db-42/crash")

		accessRecord := capturer.requireSingle(t)
		assert.Equal(t, slog.LevelError, accessRecord.level)
		assert.Equal(t, "request failed", accessRecord.message)
		assert.Equal(t, "500", accessRecord.attributes["status"])
		assert.Equal(t, "/api/v1/databases/:databaseId/crash", accessRecord.attributes["route"])
		assert.Contains(t, accessRecord.attributes["error"], "storage backend unreachable")
	})

	t.Run("carries user_id when the users middleware adds it to the context", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		userID := uuid.NewString()
		router := newAccessLoggedRouter(capturer)
		router.Use(addContextAttribute("user_id", userID))
		router.GET("/api/v1/storages", func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })

		serveAccessLogged(t, router, "/api/v1/storages")

		assert.Equal(t, userID, capturer.requireSingle(t).attributes["user_id"])
	})

	t.Run("carries agent_id when the agent middleware adds it to the context", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		agentID := uuid.NewString()
		router := newAccessLoggedRouter(capturer)
		router.Use(addContextAttribute("agent_id", agentID))
		router.GET("/api/v1/agent/verification/heartbeat", func(ginContext *gin.Context) {
			ginContext.Status(http.StatusOK)
		})

		serveAccessLogged(t, router, "/api/v1/agent/verification/heartbeat")

		assert.Equal(t, agentID, capturer.requireSingle(t).attributes["agent_id"])
	})

	t.Run("leaves the query string out of the record when the path has one", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		router := newAccessLoggedRouter(capturer)
		router.GET("/api/v1/storages", func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })

		serveAccessLogged(t, router, "/api/v1/storages?token=secret-download-token")

		for _, attributeValue := range capturer.requireSingle(t).attributes {
			assert.NotContains(t, attributeValue, "secret-download-token")
		}
	})

	t.Run("marks the route as static when no route matches", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		router := newAccessLoggedRouter(capturer)
		router.NoRoute(func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })

		serveAccessLogged(t, router, "/assets/index.js")

		assert.Equal(t, staticRouteName, capturer.requireSingle(t).attributes["route"])
	})
}

func newAccessLoggedRouter(capturer *recordCapturer) *gin.Engine {
	router := gin.New()
	router.Use(AssignRequestID(), LogAccess(logging.NewWithSinks(capturer).GetLogger()))

	return router
}

func addContextAttribute(key, value string) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ginContext.Request = ginContext.Request.WithContext(
			logging.With(ginContext.Request.Context(), slog.String(key, value)),
		)
		ginContext.Next()
	}
}

func serveAccessLogged(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))

	return response
}

type capturedRecord struct {
	level      slog.Level
	message    string
	attributes map[string]string
}

type recordCapturer struct {
	mutex   sync.Mutex
	records []capturedRecord
}

func (c *recordCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (c *recordCapturer) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *recordCapturer) WithGroup(string) slog.Handler { return c }

func (c *recordCapturer) Handle(_ context.Context, record slog.Record) error {
	captured := capturedRecord{level: record.Level, message: record.Message, attributes: map[string]string{}}

	record.Attrs(func(attr slog.Attr) bool {
		captured.attributes[attr.Key] = attr.Value.String()

		return true
	})

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.records = append(c.records, captured)

	return nil
}

func (c *recordCapturer) requireSingle(t *testing.T) capturedRecord {
	t.Helper()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	require.Len(t, c.records, 1)

	return c.records[0]
}
