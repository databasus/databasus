package agentapi

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/verification/internal/agents"
)

type agentKey struct{}

func (h *Handler) GetAuthenticationMiddleware(hooks *apierror.Hooks) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, hasBearerPrefix := strings.CutPrefix(ctx.GetHeader("Authorization"), "Bearer ")
		if !hasBearerPrefix {
			token = ""
		}

		agent, err := h.agents.AuthenticateAgent(
			ctx.Request.Context(),
			agents.AgentCredentials{AgentID: ctx.Param("agentId"), Token: token},
		)
		if err != nil {
			hooks.HandleHandlerError(ctx, err)
			ctx.Abort()
			return
		}

		requestContext := context.WithValue(ctx.Request.Context(), agentKey{}, &agent)
		ctx.Request = ctx.Request.WithContext(logging.With(requestContext, slog.String("agent_id", agent.ID.String())))
		ctx.Next()
	}
}

func getAgent(ctx context.Context) *agents.Agent {
	agent, _ := ctx.Value(agentKey{}).(*agents.Agent)
	return agent
}
