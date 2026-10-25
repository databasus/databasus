package httpserver

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// Compress gzips responses except files that are compressed already.
func Compress() gin.HandlerFunc {
	return gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"}),
	)
}
