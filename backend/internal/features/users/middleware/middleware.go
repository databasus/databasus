package users_middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/lib/logging"
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

		ctx.Set("user", user)

		ctx.Request = ctx.Request.WithContext(
			logging.With(ctx.Request.Context(), slog.String("user_id", user.ID.String())),
		)

		ctx.Next()
	}
}

func RequireRole(requiredRole users_enums.UserRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userInterface, exists := ctx.Get("user")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			ctx.Abort()
			return
		}

		user, ok := userInterface.(*users_models.User)
		if !ok {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			ctx.Abort()
			return
		}

		if user.Role != requiredRole {
			logger.GetLogger().WarnContext(ctx.Request.Context(),
				"rejected a request from a user without the required role",
				"user_id", user.ID, "required_role", requiredRole, "user_role", user.Role,
				"client_ip", ctx.ClientIP())

			ctx.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// GetUserFromContext helper function to extract user from gin context
func GetUserFromContext(ctx *gin.Context) (*users_models.User, bool) {
	userInterface, exists := ctx.Get("user")
	if !exists {
		return nil, false
	}

	user, ok := userInterface.(*users_models.User)

	return user, ok
}
