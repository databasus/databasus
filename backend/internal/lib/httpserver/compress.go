package httpserver

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func Compress(excludedPathsRegexs []string) gin.HandlerFunc {
	return gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"}),
		gzip.WithExcludedPathsRegexs(excludedPathsRegexs),
	)
}
