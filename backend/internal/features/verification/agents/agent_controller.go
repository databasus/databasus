package verification_agents

import (
	"net/http"

	"github.com/gin-gonic/gin"

	verification_bridge "databasus-backend/internal/features/verification"
	"databasus-backend/internal/modules/verification"
)

type AgentFacingController struct{}

func (c *AgentFacingController) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/agent/verification/:agentId")
	g.Use(RequireAgentAuth())

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

	abortVerificationIDs, seenAt, err := verification_bridge.GetModule().
		RecordAgentHeartbeat(ctx.Request.Context(), *agent, verification.AgentHeartbeat{Capacity: verification.HeartbeatCapacity{MaxCPU: request.MaxCPU, MaxRAMGb: request.MaxRAMGb, MaxDiskGb: request.MaxDiskGb, MaxConcurrentJobs: request.MaxConcurrentJobs}, CurrentVerificationIDs: request.CurrentVerificationIDs})
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
