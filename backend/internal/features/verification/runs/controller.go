package verification_runs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
)

type VerificationController struct {
	verificationService *VerificationService
}

func (c *VerificationController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/verifications/enqueue", c.EnqueueManual)
	router.POST("/verifications/:id/cancel", c.Cancel)
	router.GET("/verifications/:id", c.GetByID)
	router.GET("/verifications/by-database/:databaseId", c.ListByDatabase)
}

func (c *VerificationController) EnqueueManual(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req EnqueueManualRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	verification, err := c.verificationService.EnqueueManualVerification(ctx.Request.Context(), actor, req.BackupID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, verification)
}

func (c *VerificationController) Cancel(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	if err := c.verificationService.CancelVerification(ctx.Request.Context(), actor, verificationID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *VerificationController) GetByID(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	verification, err := c.verificationService.GetVerificationByID(ctx.Request.Context(), actor, verificationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, verification)
}

func (c *VerificationController) ListByDatabase(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("databaseId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var req GetVerificationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.verificationService.GetVerificationsByDatabaseID(
		ctx.Request.Context(),
		actor,
		databaseID,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
