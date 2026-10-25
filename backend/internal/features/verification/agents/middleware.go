package verification_agents

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/util/ratelimiter"
)

const (
	agentContextKey = "verification_agent"

	rateLimitAgentEndpoint = "verification-agent-id"
	rateLimitAgentMax      = 10
	rateLimitAgentWindow   = time.Second

	genericAuthError = "invalid agent credentials"
)

func (s *AgentService) RequireAgentAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := ctx.ClientIP()

		agentID, err := uuid.Parse(ctx.Param("agentId"))
		if err != nil {
			s.logger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "reason", "invalid_uuid")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		isAllowed, rateLimitErr := s.rateLimiter.RecordAttemptAndCheckIsAllowed(
			ctx.Request.Context(),
			ratelimiter.Attempt{
				Scope:      rateLimitAgentEndpoint,
				Identifier: agentID.String(),
				Limit:      rateLimitAgentMax,
				Window:     rateLimitAgentWindow,
			},
		)
		if rateLimitErr != nil {
			s.logger.ErrorContext(ctx.Request.Context(), "verification agent rate limit check failed",
				"error", rateLimitErr)
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests"})

			return
		}
		if !isAllowed {
			s.logger.WarnContext(ctx.Request.Context(), "verification agent per-agent rate limit hit",
				"agent_id", agentID, "client_ip", clientIP)
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests"})

			return
		}

		header := ctx.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			s.logger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "agent_id", agentID, "reason", "missing_token")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		agent, err := s.VerifyAgentCredentials(agentID, token)
		if err != nil {
			s.logger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "agent_id", agentID, "reason", "invalid_credentials")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		ctx.Set(agentContextKey, agent)
		ctx.Request = ctx.Request.WithContext(
			logging.With(ctx.Request.Context(), slog.String("agent_id", agent.ID.String())),
		)
		ctx.Next()
	}
}

func GetAgentFromContext(ctx *gin.Context) (*Agent, bool) {
	v, exists := ctx.Get(agentContextKey)
	if !exists {
		return nil, false
	}

	agent, ok := v.(*Agent)

	return agent, ok
}
