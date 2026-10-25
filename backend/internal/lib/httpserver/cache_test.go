package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNoStoreCacheControl(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		path string
	}{
		{name: "marks the response uncacheable when an api route answers", path: "/api/v1/system/version"},
		{name: "marks the response uncacheable when the static fallback answers", path: "/assets/index-a1b2c3d4.js"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			router := gin.New()
			router.Use(NoStoreCacheControl())
			router.GET("/api/v1/system/version", func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })
			router.NoRoute(func(ginContext *gin.Context) { ginContext.Status(http.StatusOK) })

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.path, nil))

			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "no-store, no-cache, must-revalidate, max-age=0", response.Header().Get("Cache-Control"))
			assert.Equal(t, "no-cache", response.Header().Get("Pragma"))
			assert.Equal(t, "0", response.Header().Get("Expires"))
		})
	}
}
