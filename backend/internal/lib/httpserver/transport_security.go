package httpserver

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

type secureTransportKey struct{}

// RememberTransportSecurity records whether a request reached the user over an encrypted
// connection. A direct TLS connection counts, and so does an operator-configured public address
// that starts with https://, because a proxy in front of the instance then terminates TLS. Forwarded
// headers are never trusted, since any client can send them.
func RememberTransportSecurity(publicURL string) gin.HandlerFunc {
	isPublicURLSecure := strings.HasPrefix(strings.ToLower(publicURL), "https://")

	return func(ginContext *gin.Context) {
		isSecure := isPublicURLSecure || ginContext.Request.TLS != nil

		ginContext.Request = ginContext.Request.WithContext(
			context.WithValue(ginContext.Request.Context(), secureTransportKey{}, isSecure),
		)

		ginContext.Next()
	}
}

// IsSecureTransport reports false for a request that was not marked by RememberTransportSecurity.
func IsSecureTransport(ctx context.Context) bool {
	isSecure, _ := ctx.Value(secureTransportKey{}).(bool)

	return isSecure
}
