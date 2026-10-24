package httpserver

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

// Recover answers a Problem 500 to a request whose handler panicked and logs the panic, so one broken
// handler does not take the server down and the panic text never reaches the client.
func Recover(logger *slog.Logger) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ginContext.Request.Context(), "panic recovered in HTTP handler",
					slog.Any("error", recovered),
					slog.String("stacktrace", string(debug.Stack())),
					slog.String("method", ginContext.Request.Method),
					slog.String("path", ginContext.Request.URL.Path),
				)

				apierror.Write(ginContext, apierror.NewInternalProblem())
			}
		}()

		ginContext.Next()
	}
}

// Compress gzips responses except files that are compressed already.
func Compress() gin.HandlerFunc {
	return gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"}),
	)
}

// AllowAnyOrigin lets the frontend dev server on another port call the API. It belongs to
// development only.
func AllowAnyOrigin(exposedHeaders []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Language",
			"Accept-Encoding",
			"Access-Control-Request-Method",
			"Access-Control-Request-Headers",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
			"Access-Control-Allow-Origin",
		},
		ExposeHeaders:    exposedHeaders,
		AllowCredentials: true,
	})
}

// HandleUnknownPath answers an unknown API path with a Problem 404 and serves the frontend for every
// other path: a file when one exists, the single-page app's index.html otherwise.
func HandleUnknownPath(staticDirectory string) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		if strings.HasPrefix(ginContext.Request.URL.Path, "/api/") {
			apierror.Write(ginContext, &apierror.Problem{
				Code:   apierror.CodeNotFound,
				Detail: "no API operation matches the path",
				Errors: nil,
			})

			return
		}

		filePath := filepath.Join(staticDirectory, filepath.Clean("/"+ginContext.Request.URL.Path))

		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			ginContext.File(filePath)

			return
		}

		ginContext.File(filepath.Join(staticDirectory, "index.html"))
	}
}
