package httpserver

import (
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

func TestAssignRequestID(t *testing.T) {
	t.Parallel()

	t.Run("returns a uuid and logs the handler's records under it", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}

		response := serveRequestIDRoute(t, capturer, "")

		requestID := response.Header().Get(RequestIDHeader)
		_, err := uuid.Parse(requestID)
		require.NoError(t, err)
		assert.Equal(t, requestID, capturer.requireSingle(t).attributes["request_id"])
	})

	t.Run("returns a distinct id for every request", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}

		firstRequestID := serveRequestIDRoute(t, capturer, "").Header().Get(RequestIDHeader)
		secondRequestID := serveRequestIDRoute(t, capturer, "").Header().Get(RequestIDHeader)

		assert.NotEqual(t, firstRequestID, secondRequestID)
	})

	t.Run("ignores the id when the client sends one", func(t *testing.T) {
		t.Parallel()
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		clientRequestID := "11111111-1111-1111-1111-111111111111"

		response := serveRequestIDRoute(t, capturer, clientRequestID)

		assert.NotEqual(t, clientRequestID, response.Header().Get(RequestIDHeader))
		assert.NotEqual(t, clientRequestID, capturer.requireSingle(t).attributes["request_id"])
	})
}

func serveRequestIDRoute(t *testing.T, capturer *recordCapturer, inboundRequestID string) *httptest.ResponseRecorder {
	t.Helper()

	logger := logging.NewWithSinks(capturer).GetLogger()

	router := gin.New()
	router.Use(AssignRequestID())
	router.GET("/api/v1/system/version", func(ginContext *gin.Context) {
		logger.InfoContext(ginContext.Request.Context(), "version requested")
		ginContext.Status(http.StatusOK)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/system/version", nil)
	if inboundRequestID != "" {
		request.Header.Set(RequestIDHeader, inboundRequestID)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}
