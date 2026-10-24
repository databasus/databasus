//go:build testhelpers

package verification_agents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	test_utils "databasus-backend/internal/util/testing"
)

func CreateTestVerificationAgent(
	t *testing.T,
	router *gin.Engine,
	adminToken, name string,
) *CreatedAgentResponse {
	t.Helper()

	var response CreatedAgentResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification/agents",
		"Bearer "+adminToken,
		CreateAgentRequest{Name: name},
		http.StatusCreated,
		&response,
	)

	return &response
}

// RemoveTestVerificationAgent is the idempotent counterpart to CreateTestVerificationAgent.
// It tolerates 404 so tests that explicitly delete the agent in their body can still defer
// this helper without the defer panicking.
func RemoveTestVerificationAgent(
	t *testing.T,
	router *gin.Engine,
	adminToken string,
	agentID uuid.UUID,
) {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/api/v1/verification/agents/"+agentID.String(),
		nil,
	)
	if err != nil {
		t.Fatalf("build delete agent request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent && resp.Code != http.StatusNotFound {
		t.Fatalf(
			"remove test verification agent: unexpected status %d, body=%s",
			resp.Code, resp.Body.String(),
		)
	}
}
