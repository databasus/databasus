package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRememberClientIP(t *testing.T) {
	t.Parallel()

	t.Run("puts the client ip into the request context", func(t *testing.T) {
		t.Parallel()
		clientIPByRequest := make(chan string, 1)
		engine := gin.New()
		engine.Use(RememberClientIP())
		engine.GET("/api/v1/items", func(ginContext *gin.Context) {
			clientIPByRequest <- GetClientIP(ginContext.Request.Context())
			ginContext.Status(http.StatusNoContent)
		})

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/items", nil)
		request.RemoteAddr = "203.0.113.7:51234"

		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code)
		assert.Equal(t, "203.0.113.7", <-clientIPByRequest)
	})
}
