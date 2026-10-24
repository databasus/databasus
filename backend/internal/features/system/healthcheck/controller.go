package system_healthcheck

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthcheckController struct {
	healthcheckService *HealthcheckService
}

func (c *HealthcheckController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/system/health", c.CheckHealth)
}

func (c *HealthcheckController) CheckHealth(ctx *gin.Context) {
	// Allow unrestricted CORS for health check endpoint
	// This enables monitoring tools from any origin to check system health
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if ctx.Request.Method == "OPTIONS" {
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}

	err := c.healthcheckService.IsHealthy(ctx.Request.Context())

	if err == nil {
		ctx.JSON(
			http.StatusOK,
			HealthcheckResponse{
				Status: "Application is healthy, internal DB working fine and disk usage is below 95%. You can connect downdetector to this endpoint",
			},
		)
		return
	}

	ctx.JSON(http.StatusServiceUnavailable, HealthcheckResponse{Status: err.Error()})
}
