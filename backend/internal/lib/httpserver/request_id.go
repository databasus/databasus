package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/logging"
)

const RequestIDHeader = "X-Request-Id"

// AssignRequestID ignores an inbound X-Request-Id: a client that picked its own ID could merge its
// actions into another user's trace.
func AssignRequestID() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		requestID := uuid.NewString()

		ginContext.Header(RequestIDHeader, requestID)
		ginContext.Request = ginContext.Request.WithContext(
			logging.With(ginContext.Request.Context(), slog.String("request_id", requestID)),
		)

		ginContext.Next()
	}
}
