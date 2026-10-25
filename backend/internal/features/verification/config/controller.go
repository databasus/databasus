package verification_config

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
)

type VerificationConfigController struct {
	verificationConfigService *VerificationConfigService
}

func (c *VerificationConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/verification-config/:databaseId", c.GetByDatabaseID)
	router.PUT("/verification-config/:databaseId", c.Save)
}

func (c *VerificationConfigController) GetByDatabaseID(ctx *gin.Context) {
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

	config, err := c.verificationConfigService.GetByDatabaseID(ctx.Request.Context(), actor, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}

func (c *VerificationConfigController) Save(ctx *gin.Context) {
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

	var req SaveBackupVerificationConfigDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := c.verificationConfigService.Save(ctx.Request.Context(), actor, databaseID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}
