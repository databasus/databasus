package verification_agents

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
)

type AgentController struct {
	agentService *AgentService
}

func (c *AgentController) RegisterRoutes(router *gin.RouterGroup) {
	publicAuth := router.Group("/verification/agents")
	publicAuth.GET("/availability", c.GetAvailability)

	adminOnly := router.Group("/verification/agents")
	adminOnly.Use(users_middleware.RequireRole(users_enums.UserRoleAdmin))

	adminOnly.POST("", c.CreateAgent)
	adminOnly.GET("", c.ListAgents)
	adminOnly.POST("/:id/rotate-token", c.RotateToken)
	adminOnly.DELETE("/:id", c.DeleteAgent)
}

func (c *AgentController) GetAvailability(ctx *gin.Context) {
	count, err := c.agentService.CountLiveAgents(ctx.Request.Context())
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, AvailabilityResponse{
		Count:     count,
		HasAgents: count > 0,
	})
}

func (c *AgentController) CreateAgent(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request CreateAgentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.agentService.CreateAgent(ctx.Request.Context(), user, &request)
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

func (c *AgentController) ListAgents(ctx *gin.Context) {
	agents, err := c.agentService.ListAgents(ctx.Request.Context())
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, agents)
}

func (c *AgentController) RotateToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	agentID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	token, err := c.agentService.RotateToken(ctx.Request.Context(), user, agentID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, RotateTokenResponse{Token: token})
}

func (c *AgentController) DeleteAgent(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	agentID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	if err := c.agentService.DeleteAgent(ctx.Request.Context(), user, agentID); err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}
