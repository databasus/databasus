package users_controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	users_dto "databasus-backend/internal/features/users/dto"
	users_errors "databasus-backend/internal/features/users/errors"
	user_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/ratelimiter"
)

const (
	testEmailRateLimitScope  = "test-email"
	testEmailRateLimit       = 5
	testEmailRateLimitWindow = 10 * time.Minute
)

type SettingsController struct {
	settingsService *users_services.SettingsService
	rateLimiter     ratelimiter.Counter
	logger          *slog.Logger
}

func (c *SettingsController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/users/settings", c.GetUsersSettings)
	router.PUT(
		"/users/settings",
		user_middleware.RequireRole(users.RoleAdmin),
		c.UpdateUsersSettings,
	)
	router.POST(
		"/users/settings/test-email",
		user_middleware.RequireRole(users.RoleAdmin),
		c.SendTestEmail,
	)
}

func (c *SettingsController) GetUsersSettings(ctx *gin.Context) {
	settings, err := c.settingsService.GetSettingsResponse(ctx.Request.Context())
	if err != nil {
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	ctx.JSON(http.StatusOK, settings)
}

func (c *SettingsController) UpdateUsersSettings(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request users_dto.UpdateSettingsRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	settings, err := c.settingsService.UpdateSettings(ctx.Request.Context(), request, actor)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, settings)
}

func (c *SettingsController) SendTestEmail(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      testEmailRateLimitScope,
		Identifier: actor.UserID.String(),
		Limit:      testEmailRateLimit,
		Window:     testEmailRateLimitWindow,
	}) {
		return
	}

	recipientEmail, err := c.settingsService.SendTestEmail(ctx.Request.Context(), actor)

	switch {
	case errors.Is(err, users_errors.ErrEmailNotConfigured):
		respondWithCode(ctx, http.StatusBadRequest, err, "email_not_configured")
	case errors.Is(err, users_errors.ErrAdminEmailMissing):
		respondWithCode(ctx, http.StatusBadRequest, err, "admin_email_missing")
	case err != nil:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		ctx.JSON(http.StatusOK, users_dto.SendTestEmailResponseDTO{RecipientEmail: recipientEmail})
	}
}
