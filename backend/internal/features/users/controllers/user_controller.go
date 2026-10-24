package users_controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/config"
	user_dto "databasus-backend/internal/features/users/dto"
	users_errors "databasus-backend/internal/features/users/errors"
	user_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/ratelimiter"
)

type UserController struct {
	userService *users_services.UserService
	rateLimiter ratelimiter.Counter
	logger      *slog.Logger
}

func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/users/signup", c.SignUp)
	router.POST("/users/signin", c.SignIn)
	router.POST("/users/verify-signin-code", c.VerifySignInCode)
	router.POST("/users/resend-signin-code", c.ResendSignInCode)

	router.GET("/users/is-any-user-exist", c.HasAnyUser)

	// Password reset (no auth required)
	router.POST("/users/send-reset-password-code", c.SendResetPasswordCode)
	router.POST("/users/reset-password", c.ResetPassword)

	// OAuth callbacks
	router.POST("/auth/github/callback", c.HandleGitHubOAuth)
	router.POST("/auth/google/callback", c.HandleGoogleOAuth)
}

func (c *UserController) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.GET("/users/me", c.GetCurrentUser)
	router.PUT("/users/me", c.UpdateUserInfo)
	router.PUT("/users/change-password", c.ChangePassword)
	router.POST("/users/invite", c.InviteUser)
}

func (c *UserController) SignUp(ctx *gin.Context) {
	var request user_dto.SignUpRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !verifyTurnstileOrRespond(ctx, request.CloudflareTurnstileToken) {
		return
	}

	user, err := c.userService.SignUp(ctx.Request.Context(), &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.userService.GenerateAccessToken(ctx.Request.Context(), user)
	if err != nil {
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *UserController) SignIn(ctx *gin.Context) {
	var request user_dto.SignInRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !verifyTurnstileOrRespond(ctx, request.CloudflareTurnstileToken) {
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      "signin",
		Identifier: request.Email,
		Limit:      10,
		Window:     time.Minute,
	}) {
		return
	}

	outcome, err := c.userService.SignIn(ctx.Request.Context(), &request)
	if err != nil {
		respondToSignInError(ctx, err)
		return
	}

	if outcome.PendingSignIn != nil {
		ctx.JSON(http.StatusOK, outcome.PendingSignIn)
		return
	}

	ctx.JSON(http.StatusOK, outcome.CompletedSignIn)
}

func (c *UserController) HasAnyUser(ctx *gin.Context) {
	hasAnyUser, err := c.userService.HasAnyUser()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user_dto.HasAnyUserResponseDTO{HasAnyUser: hasAnyUser})
}

func (c *UserController) ChangePassword(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.ChangePasswordRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if request.NewPassword == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "New password is required"})
		return
	}

	if len(request.NewPassword) < 8 {
		ctx.JSON(
			http.StatusBadRequest,
			gin.H{"error": "New password must be at least 8 characters long"},
		)
		return
	}

	if err := c.userService.ChangeUserPassword(ctx.Request.Context(), user.ID, request.NewPassword); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func (c *UserController) InviteUser(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.InviteUserRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.InviteUser(ctx.Request.Context(), &request, user)
	if err != nil {
		if errors.Is(err, users_errors.ErrInsufficientPermissionsToInviteUsers) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *UserController) GetCurrentUser(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	profile := c.userService.GetCurrentUserProfile(ctx.Request.Context(), user)
	ctx.JSON(http.StatusOK, profile)
}

func (c *UserController) UpdateUserInfo(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.UpdateUserInfoRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if request.Name == nil && request.Email == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := c.userService.UpdateUserInfo(ctx.Request.Context(), user.ID, &request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User info updated successfully"})
}

func (c *UserController) HandleGitHubOAuth(ctx *gin.Context) {
	env := config.GetEnv()
	if env.GitHubClientID == "" || env.GitHubClientSecret == "" {
		ctx.JSON(http.StatusNotImplemented, gin.H{"error": "GitHub OAuth is not configured"})
		return
	}

	var request user_dto.OAuthCallbackRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.HandleGitHubOAuth(ctx.Request.Context(), request.Code, request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *UserController) HandleGoogleOAuth(ctx *gin.Context) {
	env := config.GetEnv()
	if env.GoogleClientID == "" || env.GoogleClientSecret == "" {
		ctx.JSON(http.StatusNotImplemented, gin.H{"error": "Google OAuth is not configured"})
		return
	}

	var request user_dto.OAuthCallbackRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.HandleGoogleOAuth(ctx.Request.Context(), request.Code, request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *UserController) SendResetPasswordCode(ctx *gin.Context) {
	var request user_dto.SendResetPasswordCodeRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !verifyTurnstileOrRespond(ctx, request.CloudflareTurnstileToken) {
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      "reset-password",
		Identifier: request.Email,
		Limit:      3,
		Window:     time.Hour,
	}) {
		return
	}

	err := c.userService.SendResetPasswordCode(ctx.Request.Context(), request.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
}

func (c *UserController) ResetPassword(ctx *gin.Context) {
	var request user_dto.ResetPasswordRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := c.userService.ResetPassword(ctx.Request.Context(), request.Email, request.Code, request.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// Each refusal carries a code, because the status alone does not tell the
// interface what happened: a wrong code and a failed challenge are both 400, and
// a resend inside its minute and an exhausted hourly cap are both 429.
func respondToSignInError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, users_errors.ErrPendingSignInNotUsable):
		respondWithCode(ctx, http.StatusGone, err, "pending_sign_in_not_usable")
	case errors.Is(err, users_errors.ErrSignInCodeNotSent):
		respondWithCode(ctx, http.StatusServiceUnavailable, err, "sign_in_code_not_sent")
	case errors.Is(err, users_errors.ErrTooManySignInCodes):
		respondWithCode(ctx, http.StatusTooManyRequests, err, "too_many_sign_in_codes")
	case errors.Is(err, users_errors.ErrSignInCodeResentTooSoon):
		respondWithCode(ctx, http.StatusTooManyRequests, err, "sign_in_code_resent_too_soon")
	case errors.Is(err, users_errors.ErrSignInCodeIncorrect):
		respondWithCode(ctx, http.StatusBadRequest, err, "sign_in_code_incorrect")
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func respondWithCode(ctx *gin.Context, status int, err error, code string) {
	ctx.JSON(status, gin.H{"error": err.Error(), "code": code})
}
