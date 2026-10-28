package httpserver

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

// Recover answers a Problem 500 to a request whose handler panicked and logs the panic, so one broken
// handler does not take the server down and the panic text never reaches the client.
func Recover(logger *slog.Logger) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recoveredError, isError := recovered.(error); isError &&
					errors.Is(recoveredError, http.ErrAbortHandler) {
					panic(http.ErrAbortHandler)
				}
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
