package users_middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/logger"
)

func AuthMiddleware(userService *users_services.UserService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			logger.GetLogger().WarnContext(ctx.Request.Context(),
				"rejected a request with no authorization header", "client_ip", ctx.ClientIP())

			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			ctx.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		user, err := userService.GetUserFromToken(ctx.Request.Context(), token)
		if err != nil {
			logger.GetLogger().WarnContext(ctx.Request.Context(),
				"rejected a request with an invalid token", "client_ip", ctx.ClientIP(), "error", err)

			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			ctx.Abort()
			return
		}

		requestContext := users.WithActor(ctx.Request.Context(), users.Actor{UserID: user.ID, Role: user.Role})
		ctx.Request = ctx.Request.WithContext(
			logging.With(requestContext, slog.String("user_id", user.ID.String())),
		)

		ctx.Next()
	}
}

func RequireRole(requiredRole users.Role) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		actor, isAuthenticated := users.GetActor(ctx.Request.Context())
		if !isAuthenticated {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			ctx.Abort()
			return
		}

		if actor.Role != requiredRole {
			logger.GetLogger().WarnContext(ctx.Request.Context(),
				"rejected a request from a user without the required role",
				"user_id", actor.UserID, "required_role", requiredRole, "user_role", actor.Role,
				"client_ip", ctx.ClientIP())

			ctx.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
