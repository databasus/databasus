package verification_agents

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/features/caching"
	verification_bridge "databasus-backend/internal/features/verification"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/internal/util/logger"
)

const (
	agentContextKey = "verification_agent"

	rateLimitAgentEndpoint = "verification-agent-id"
	rateLimitAgentMax      = 10
	rateLimitAgentWindow   = time.Second

	genericAuthError = "invalid agent credentials"
)

func RequireAgentAuth() gin.HandlerFunc {
	authLogger := logger.GetLogger()
	return func(ctx *gin.Context) {
		clientIP := ctx.ClientIP()

		agentID, err := uuid.Parse(ctx.Param("agentId"))
		if err != nil {
			authLogger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "reason", "invalid_uuid")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		isAllowed, rateLimitErr := caching.GetRateLimiter().RecordAttemptAndCheckIsAllowed(
			ctx.Request.Context(),
			ratelimit.Attempt{
				Scope:      rateLimitAgentEndpoint,
				Identifier: agentID.String(),
				Limit:      rateLimitAgentMax,
				Window:     rateLimitAgentWindow,
			},
		)
		if rateLimitErr != nil {
			authLogger.ErrorContext(ctx.Request.Context(), "verification agent rate limit check failed",
				"error", rateLimitErr)
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests"})

			return
		}
		if !isAllowed {
			authLogger.WarnContext(ctx.Request.Context(), "verification agent per-agent rate limit hit",
				"agent_id", agentID, "client_ip", clientIP)
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests"})

			return
		}

		header := ctx.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			authLogger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "agent_id", agentID, "reason", "missing_token")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		agent, err := verification_bridge.GetModule().AuthenticateAgent(ctx.Request.Context(), agentID, token)
		if err != nil {
			authLogger.WarnContext(ctx.Request.Context(), "verification agent auth failure",
				"client_ip", clientIP, "agent_id", agentID, "reason", "invalid_credentials")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": genericAuthError})

			return
		}

		ctx.Set(agentContextKey, &agent)
		ctx.Request = ctx.Request.WithContext(
			logging.With(ctx.Request.Context(), slog.String("agent_id", agent.ID.String())),
		)
		ctx.Next()
	}
}

func GetAgentFromContext(ctx *gin.Context) (*verification.Agent, bool) {
	v, exists := ctx.Get(agentContextKey)
	if !exists {
		return nil, false
	}

	agent, ok := v.(*verification.Agent)

	return agent, ok
}
