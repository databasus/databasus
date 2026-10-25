package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
)

// GetRequestContext returns the context of the request a generated strict handler serves. The
// strict server hands the handler its *gin.Context, which Gin reuses for the next request once this
// one is answered; a query that still watches that context, as database/sql does, would then read
// another request. Every strict handler passes the request's own context down instead.
func GetRequestContext(ctx context.Context) context.Context {
	if ginContext, isGinContext := ctx.(*gin.Context); isGinContext && ginContext.Request != nil {
		return ginContext.Request.Context()
	}

	return ctx
}
