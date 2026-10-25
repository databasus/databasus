package healthcheck_attempt

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
)

type HealthcheckAttemptController struct {
	healthcheckAttemptService *HealthcheckAttemptService
}

func (c *HealthcheckAttemptController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/healthcheck-attempts/:databaseId", c.GetAttemptsByDatabase)
}

func (c *HealthcheckAttemptController) GetAttemptsByDatabase(ctx *gin.Context) {
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

	afterDate := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if afterDateStr := ctx.Query("afterDate"); afterDateStr != "" {
		parsedDate, err := time.Parse(time.RFC3339, afterDateStr)
		if err != nil {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "invalid afterDate format, use RFC3339"},
			)
			return
		}
		afterDate = parsedDate
	}

	attempts, err := c.healthcheckAttemptService.GetAttemptsByDatabase(
		ctx.Request.Context(),
		actor,
		databaseID,
		afterDate,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, attempts)
}
