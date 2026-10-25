package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type requestContextKey struct{}

func TestGetRequestContext(t *testing.T) {
	t.Parallel()

	t.Run("returns the request's own context when given the gin context of the request", func(t *testing.T) {
		t.Parallel()
		request := httptest.NewRequestWithContext(
			context.WithValue(t.Context(), requestContextKey{}, "request value"), http.MethodGet, "/api/v1/items", nil,
		)
		ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
		ginContext.Request = request

		requestContext := GetRequestContext(ginContext)

		_, isGinContext := requestContext.(*gin.Context)
		assert.False(t, isGinContext)
		assert.Equal(t, "request value", requestContext.Value(requestContextKey{}))
	})

	t.Run("returns the context unchanged when it is not a gin context", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, t.Context(), GetRequestContext(t.Context()))
	})
}
