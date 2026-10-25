package verification_agents

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/util/ratelimiter"
)

type failingRateLimitCounter struct {
	err error
}

func (c failingRateLimitCounter) RecordAttemptAndCheckIsAllowed(
	context.Context,
	ratelimiter.Attempt,
) (bool, error) {
	return false, c.err
}

func Test_RequireAgentAuth_WhenRateLimiterFails_RejectsWithoutLoggingAgentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logOutput bytes.Buffer
	service := &AgentService{
		rateLimiter: failingRateLimitCounter{err: errors.New("counter failed")},
		logger:      slog.New(slog.NewTextHandler(&logOutput, nil)),
	}
	router := gin.New()
	router.GET("/agents/:agentId", service.RequireAgentAuth(), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	agentID := uuid.New()
	request := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String(), nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.NotContains(t, logOutput.String(), agentID.String())
}

func TestRequireAgentAuth(t *testing.T) {
	t.Run("adds agent_id to the log records of the request when the agent authenticates", func(t *testing.T) {
		cleanupAgents(t)

		admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
		createdAgent := CreateTestVerificationAgent(t, createTestRouter(), admin.Token, "logged-"+uuid.NewString())

		var loggedAgentIDs []string
		requestLogger := logging.NewWithSinks(agentIDCapturer{agentIDs: &loggedAgentIDs}).GetLogger()

		router := gin.New()
		router.GET("/agents/:agentId", GetAgentService().RequireAgentAuth(), func(ctx *gin.Context) {
			requestLogger.InfoContext(ctx.Request.Context(), "heartbeat received")
			ctx.Status(http.StatusNoContent)
		})

		request := httptest.NewRequest(http.MethodGet, "/agents/"+createdAgent.Agent.ID.String(), nil)
		request.Header.Set("Authorization", "Bearer "+createdAgent.Token)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code)
		assert.Equal(t, []string{createdAgent.Agent.ID.String()}, loggedAgentIDs)
	})
}

type agentIDCapturer struct {
	agentIDs *[]string
}

func (c agentIDCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (c agentIDCapturer) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c agentIDCapturer) WithGroup(string) slog.Handler { return c }

func (c agentIDCapturer) Handle(_ context.Context, record slog.Record) error {
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "agent_id" {
			*c.agentIDs = append(*c.agentIDs, attr.Value.String())
		}

		return true
	})

	return nil
}
