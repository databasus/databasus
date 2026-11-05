package httpserver

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Static assets and the SPA fallback match no route, so ginContext.FullPath() is empty for them. A
// marker keeps that traffic filterable instead of looking like a routing bug.
const staticRouteName = "static"

// LogAccess logs one record per request. It reads the request context after the handlers ran, so
// attributes that later middleware adds with logging.With, such as user_id and agent_id, reach the
// record. The query string is left out: download and restore tokens travel there.
func LogAccess(logger *slog.Logger) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		startedAt := time.Now()

		ginContext.Next()

		status := ginContext.Writer.Status()

		route := ginContext.FullPath()
		if route == "" {
			route = staticRouteName
		}

		attributes := []slog.Attr{
			slog.String("method", ginContext.Request.Method),
			slog.String("path", ginContext.Request.URL.Path),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			slog.String("route", route),
			slog.String("client_ip", ginContext.ClientIP()),
			slog.Int("response_bytes", ginContext.Writer.Size()),
		}

		if status < http.StatusInternalServerError {
			logger.LogAttrs(ginContext.Request.Context(), slog.LevelInfo, "request served", attributes...)

			return
		}

		// Handlers attach the cause with ginContext.Error so the access record carries it, rather
		// than each of them logging its own near-duplicate of this record.
		if handlerErrors := collectHandlerErrors(ginContext); handlerErrors != "" {
			attributes = append(attributes, slog.String("error", handlerErrors))
		}

		logger.LogAttrs(ginContext.Request.Context(), slog.LevelError, "request failed", attributes...)
	}
}

func collectHandlerErrors(ginContext *gin.Context) string {
	if len(ginContext.Errors) == 0 {
		return ""
	}

	messages := make([]string, 0, len(ginContext.Errors))
	for _, handlerError := range ginContext.Errors {
		messages = append(messages, handlerError.Error())
	}

	return strings.Join(messages, "; ")
}
