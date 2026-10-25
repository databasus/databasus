package users_bridge

import (
	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users"
)

// RequireAdmin guards a route of the old code that only an instance admin may call. It runs after
// the session middleware, which puts the actor into the context.
func RequireAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		actor, isFound := users.GetActor(ctx.Request.Context())
		if !isFound {
			apierror.Write(ctx, errUnauthenticated)

			return
		}

		if !actor.IsAdmin() {
			apierror.Write(ctx, errNotAdmin)

			return
		}

		ctx.Next()
	}
}
