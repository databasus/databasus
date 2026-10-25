package httpserver

import "github.com/gin-gonic/gin"

// NoStoreCacheControl marks every response uncacheable: the API is dynamic and often sensitive, and
// the image rewrites index.html and runtime-config.js on every container start. Pragma and Expires
// cover HTTP/1.0 intermediaries.
func NoStoreCacheControl() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ginContext.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		ginContext.Header("Pragma", "no-cache")
		ginContext.Header("Expires", "0")
		ginContext.Next()
	}
}
