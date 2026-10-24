package verification_agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AgentFacingController struct {
	agentService *AgentService
}

func (c *AgentFacingController) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/agent/verification/:agentId")
	g.Use(c.agentService.RequireAgentAuth())

	g.POST("/heartbeat", c.Heartbeat)
}

func (c *AgentFacingController) Heartbeat(ctx *gin.Context) {
	agent, ok := GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	var request HeartbeatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seenAt, abortVerificationIDs, err := c.agentService.Heartbeat(ctx.Request.Context(), agent, &request)
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, HeartbeatResponse{
		LastSeenAt:           seenAt,
		AbortVerificationIDs: abortVerificationIDs,
	})
}
