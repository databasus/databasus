package healthcheck_config

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
)

type HealthcheckConfigController struct {
	healthcheckConfigService *HealthcheckConfigService
}

func (c *HealthcheckConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/healthcheck-config", c.SaveHealthcheckConfig)
	router.GET("/healthcheck-config/:databaseId", c.GetHealthcheckConfig)
}

func (c *HealthcheckConfigController) SaveHealthcheckConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var configDTO HealthcheckConfigDTO
	if err := ctx.ShouldBindJSON(&configDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.healthcheckConfigService.Save(ctx.Request.Context(), *user, configDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Healthcheck configuration saved successfully",
	})
}

func (c *HealthcheckConfigController) GetHealthcheckConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("databaseId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	config, err := c.healthcheckConfigService.GetByDatabaseID(ctx.Request.Context(), *user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}
