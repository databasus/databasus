package users_controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	user_dto "databasus-backend/internal/features/users/dto"
	"databasus-backend/internal/util/ratelimiter"
)

const (
	signInCodeVerificationScope  = "signin-code-verification"
	signInCodeVerificationLimit  = 10
	signInCodeVerificationWindow = time.Minute
)

func (c *UserController) VerifySignInCode(ctx *gin.Context) {
	var request user_dto.VerifySignInCodeRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !verifyTurnstileOrRespond(ctx, request.CloudflareTurnstileToken) {
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      signInCodeVerificationScope,
		Identifier: request.PendingSignInID.String(),
		Limit:      signInCodeVerificationLimit,
		Window:     signInCodeVerificationWindow,
	}) {
		return
	}

	response, err := c.userService.VerifyTwoFactorCode(
		ctx.Request.Context(),
		request.PendingSignInID,
		request.Code,
	)
	if err != nil {
		respondToSignInError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *UserController) ResendSignInCode(ctx *gin.Context) {
	var request user_dto.ResendSignInCodeRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !verifyTurnstileOrRespond(ctx, request.CloudflareTurnstileToken) {
		return
	}

	response, err := c.userService.ResendTwoFactorCode(ctx.Request.Context(), request.PendingSignInID)
	if err != nil {
		respondToSignInError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, response)
}
