package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
)

type clientIPKey struct{}

// RememberClientIP puts the client's IP into the request context, so a service that sees only a
// context, such as one that verifies a human-verification answer, can read it.
func RememberClientIP() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		ginContext.Request = ginContext.Request.WithContext(
			context.WithValue(ginContext.Request.Context(), clientIPKey{}, ginContext.ClientIP()),
		)

		ginContext.Next()
	}
}

func GetClientIP(ctx context.Context) string {
	clientIP, _ := ctx.Value(clientIPKey{}).(string)

	return clientIP
}
