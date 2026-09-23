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

// SignUp
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SignUpRequestDTO true "User signup data"
// @Success 200 {object} users_dto.SignInResponseDTO
// @Failure 400
// @Router /users/signup [post]
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

// SignIn
// @Summary Authenticate a user
// @Description Authenticate a user with email and password. With the second factor on, a correct password answers with a pending sign-in instead of a token, and the emailed code completes it through /users/verify-signin-code.
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SignInRequestDTO true "User signin data"
// @Success 200 {object} users_dto.SignInOutcomeResponseDTO "An access token when the instance asks for one factor, and a pending sign-in when it asks for an emailed code"
// @Failure 400
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Router /users/signin [post]
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

// HasAnyUser
// @Summary Check whether the instance holds any account
// @Description Tells the entry screen whether to offer signing in or the registration that claims the instance, and whether registration is open at all
// @Tags users
// @Produce json
// @Success 200 {object} users_dto.HasAnyUserResponseDTO
// @Failure 500
// @Router /users/is-any-user-exist [get]
func (c *UserController) HasAnyUser(ctx *gin.Context) {
	hasAnyUser, err := c.userService.HasAnyUser()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	isSignUpAllowed, err := c.userService.IsSignUpAllowed(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user_dto.HasAnyUserResponseDTO{
		HasAnyUser:      hasAnyUser,
		IsSignUpAllowed: isSignUpAllowed,
	})
}

// ChangePassword
// @Summary Change user password
// @Description Change the password for the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.ChangePasswordRequestDTO true "New password data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/change-password [put]
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

// InviteUser
// @Summary Invite a new user
// @Description Invite a new user to the system with optional workspace assignment
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.InviteUserRequestDTO true "User invitation data"
// @Success 200 {object} users_dto.InviteUserResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/invite [post]
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

// GetCurrentUser
// @Summary Get current user profile
// @Description Get the profile information of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} users_dto.UserProfileResponseDTO
// @Failure 401 {object} map[string]string
// @Router /users/me [get]
func (c *UserController) GetCurrentUser(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	profile := c.userService.GetCurrentUserProfile(ctx.Request.Context(), user)
	ctx.JSON(http.StatusOK, profile)
}

// UpdateUserInfo
// @Summary Update current user information
// @Description Update name and/or email for the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.UpdateUserInfoRequestDTO true "User info update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/me [put]
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

// HandleGitHubOAuth
// @Summary Handle GitHub OAuth callback
// @Description Exchange GitHub authorization code for JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OAuthCallbackRequestDTO true "OAuth callback data"
// @Success 200 {object} users_dto.OAuthCallbackResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/github/callback [post]
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

// HandleGoogleOAuth
// @Summary Handle Google OAuth callback
// @Description Exchange Google authorization code for JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OAuthCallbackRequestDTO true "OAuth callback data"
// @Success 200 {object} users_dto.OAuthCallbackResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/google/callback [post]
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

// SendResetPasswordCode
// @Summary Send password reset code
// @Description Send a password reset code to the user's email
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SendResetPasswordCodeRequestDTO true "Email address"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /users/send-reset-password-code [post]
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

// ResetPassword
// @Summary Reset password with code
// @Description Reset user password using the code sent via email
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.ResetPasswordRequestDTO true "Reset password data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /users/reset-password [post]
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
