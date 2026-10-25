package verification_agents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		GetAgentController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	GetAgentFacingController().RegisterRoutes(v1)

	audit_logs.SetupDependencies()

	return router
}

func cleanupAgents(t *testing.T) {
	t.Helper()
	if err := storage.GetDb().Exec("DELETE FROM verification_agents").Error; err != nil {
		t.Fatalf("failed to clean verification_agents: %v", err)
	}
}

func heartbeatPathFor(agentID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/agent/verification/%s/heartbeat", agentID)
}

func Test_CreateAgent_WhenUserIsAdmin_AgentCreated(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	name := "test-agent-" + uuid.New().String()
	var createResponse CreatedAgentResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		CreateAgentRequest{Name: name},
		http.StatusCreated, &createResponse,
	)

	assert.NotEmpty(t, createResponse.Token, "raw token should be returned exactly once")
	require.NotNil(t, createResponse.Agent)
	assert.Equal(t, name, createResponse.Agent.Name)
	assert.NotEqual(t, uuid.Nil, createResponse.Agent.ID)
	assert.Empty(t, createResponse.Agent.TokenHash, "token hash must never leave the server")

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	foundCreatedAgent := false
	for _, agent := range listedAgents {
		if agent.ID == createResponse.Agent.ID {
			foundCreatedAgent = true
			assert.Equal(t, name, agent.Name)
		}
	}
	assert.True(t, foundCreatedAgent, "newly created agent should appear in the list response")
}

func Test_CreateAgent_ResponseJSON_NeverIncludesTokenHashOrDeletedAt(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createResponse := test_utils.MakePostRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		CreateAgentRequest{Name: "json-leak-test-" + uuid.New().String()},
		http.StatusCreated,
	)

	createBody := string(createResponse.Body)
	assert.NotContains(t, createBody, "tokenHash")
	assert.NotContains(t, createBody, "token_hash")
	assert.NotContains(t, createBody, "deletedAt")
	assert.NotContains(t, createBody, "deleted_at")
	assert.Contains(t, createBody, "maxRamGb", "RAM should be reported in GB, not MB")
	assert.NotContains(t, createBody, "maxRamMb")
}

func Test_CreateAgent_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+member.Token,
		CreateAgentRequest{Name: "x"},
		http.StatusForbidden,
	)
}

func Test_CreateAgent_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/verification/agents",
		"",
		CreateAgentRequest{Name: "x"},
		http.StatusUnauthorized,
	)
}

func Test_CreateAgent_WhenUserIsWorkspaceOwner_StillReturnsForbidden(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws-"+uuid.New().String(), member, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+member.Token,
		CreateAgentRequest{Name: "x"},
		http.StatusForbidden,
	)
}

func Test_ListAgents_WhenUserIsAdmin_ReturnsAll(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	CreateTestVerificationAgent(t, router, admin.Token, "agent-a-"+uuid.New().String())
	CreateTestVerificationAgent(t, router, admin.Token, "agent-b-"+uuid.New().String())

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	assert.GreaterOrEqual(t, len(listedAgents), 2)
}

func Test_ListAgents_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+member.Token,
		http.StatusForbidden,
	)
}

func Test_ListAgents_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verification/agents",
		"",
		http.StatusUnauthorized,
	)
}

func Test_RotateToken_WhenUserIsAdmin_ReturnsNewToken_InvalidatesOld(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "rotate-test-"+uuid.New().String())
	originalToken := createdAgent.Token

	var firstRotation RotateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		map[string]any{},
		http.StatusOK, &firstRotation,
	)

	rotatedToken := firstRotation.Token
	assert.NotEmpty(t, rotatedToken)
	assert.NotEqual(t, originalToken, rotatedToken)

	var secondRotation RotateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		map[string]any{},
		http.StatusOK, &secondRotation,
	)
	assert.NotEqual(t, rotatedToken, secondRotation.Token)
}

func Test_RotateToken_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "rotate-forbid-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", createdAgent.Agent.ID),
		"Bearer "+member.Token,
		map[string]any{},
		http.StatusForbidden,
	)

	// Confirm the agent is still present after the forbidden attempt.
	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	foundCreatedAgent := false
	for _, agent := range listedAgents {
		if agent.ID == createdAgent.Agent.ID {
			foundCreatedAgent = true
		}
	}
	assert.True(t, foundCreatedAgent)
}

func Test_RotateToken_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	test_utils.MakePostRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", uuid.New()),
		"",
		map[string]any{},
		http.StatusUnauthorized,
	)
}

func Test_RotateToken_OnUnknownID_Returns404(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	test_utils.MakePostRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", uuid.New()),
		"Bearer "+admin.Token,
		map[string]any{},
		http.StatusNotFound,
	)
}

func Test_RotateToken_OnSoftDeletedAgent_Returns404(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "rotate-deleted-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	test_utils.MakePostRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		map[string]any{},
		http.StatusNotFound,
	)
}

func Test_DeleteAgent_WhenUserIsAdmin_RemovesFromList(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "delete-test-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)
	for _, agent := range listedAgents {
		assert.NotEqual(t, createdAgent.Agent.ID, agent.ID, "deleted agent must not appear in list")
	}
}

func Test_DeleteAgent_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "delete-forbid-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+member.Token,
		http.StatusForbidden,
	)

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	foundCreatedAgent := false
	for _, agent := range listedAgents {
		if agent.ID == createdAgent.Agent.ID {
			foundCreatedAgent = true
		}
	}
	assert.True(t, foundCreatedAgent, "member should not have been able to delete the agent")
}

func Test_DeleteAgent_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", uuid.New()),
		"",
		http.StatusUnauthorized,
	)
}

func Test_DeleteAgent_OnUnknownID_Returns404(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", uuid.New()),
		"Bearer "+admin.Token,
		http.StatusNotFound,
	)
}

func Test_DeleteAgent_Twice_SecondCallReturns404(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "delete-twice-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNotFound,
	)

	// Confirm that the second (failed) delete did NOT write a duplicate audit log.
	var deleteAuditCount int64
	storage.GetDb().
		Raw(`SELECT count(*) FROM lib_audit_events WHERE message LIKE ?`,
			"verification agent deleted: "+createdAgent.Agent.Name).
		Scan(&deleteAuditCount)
	assert.Equal(t, int64(1), deleteAuditCount,
		"only the successful first delete should write an audit log row")
}

func Test_ListAgents_ExcludesSoftDeletedAgents(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	keptAgent := CreateTestVerificationAgent(t, router, admin.Token, "kept-"+uuid.New().String())
	deletedAgent := CreateTestVerificationAgent(t, router, admin.Token, "deleted-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", deletedAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	foundKeptAgent := false
	for _, agent := range listedAgents {
		assert.NotEqual(t, deletedAgent.Agent.ID, agent.ID)
		if agent.ID == keptAgent.Agent.ID {
			foundKeptAgent = true
		}
	}
	assert.True(t, foundKeptAgent)

	// Both rows still exist physically, but only the deleted one has deleted_at set.
	var physicalRowCount int64
	storage.GetDb().Raw(`SELECT count(*) FROM verification_agents WHERE id IN (?, ?)`,
		keptAgent.Agent.ID, deletedAgent.Agent.ID).Scan(&physicalRowCount)
	assert.Equal(t, int64(2), physicalRowCount, "soft-delete must preserve the row")

	var softDeletedRowCount int64
	storage.GetDb().Raw(`SELECT count(*) FROM verification_agents WHERE id = ? AND deleted_at IS NOT NULL`,
		deletedAgent.Agent.ID).Scan(&softDeletedRowCount)
	assert.Equal(t, int64(1), softDeletedRowCount)
}

func Test_CreateAgent_WritesAuditLog(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	name := "audit-test-" + uuid.New().String()
	CreateTestVerificationAgent(t, router, admin.Token, name)

	var auditRow struct {
		UserID      *uuid.UUID
		WorkspaceID *uuid.UUID
		Message     string
	}
	err := storage.GetDb().
		Raw(`SELECT author_user_id AS user_id, workspace_id, message FROM lib_audit_events WHERE message = ?`,
			"verification agent created: "+name).
		Scan(&auditRow).Error
	require.NoError(t, err)

	require.NotNil(t, auditRow.UserID, "audit log must reference the admin who created the agent")
	assert.Equal(t, admin.UserID, *auditRow.UserID)
	assert.Nil(t, auditRow.WorkspaceID, "verification agents are system-global - workspace_id must be NULL")
}

// Helper to confirm the raw token shape (hex of 32 chars).
func Test_CreateAgent_TokenIsHex32(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hex-token-"+uuid.New().String())

	assert.Len(t, createdAgent.Token, 32, "token is uuid-without-dashes (32 hex chars)")
	assert.NotContains(t, createdAgent.Token, "-")
}

func Test_GetAvailability_WhenUserIsMember_ReturnsCountWithoutAgentDetails(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	CreateTestVerificationAgent(t, router, admin.Token, "avail-a-"+uuid.New().String())
	CreateTestVerificationAgent(t, router, admin.Token, "avail-b-"+uuid.New().String())

	availabilityResponse := test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verification/agents/availability",
		"Bearer "+member.Token,
		http.StatusOK,
	)

	var availability AvailabilityResponse
	require.NoError(t, json.Unmarshal(availabilityResponse.Body, &availability))
	assert.Equal(t, int64(2), availability.Count)
	assert.True(t, availability.HasAgents)

	availabilityBody := string(availabilityResponse.Body)
	assert.NotContains(t, availabilityBody, "tokenHash")
	assert.NotContains(t, availabilityBody, "token_hash")
	assert.NotContains(t, availabilityBody, "\"id\"")
	assert.NotContains(t, availabilityBody, "\"name\"")
	assert.NotContains(t, availabilityBody, "lastSeenAt")
}

func Test_GetAvailability_WhenNoAgents_ReturnsZeroAndFalse(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	var availability AvailabilityResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents/availability",
		"Bearer "+member.Token,
		http.StatusOK, &availability,
	)

	assert.Equal(t, int64(0), availability.Count)
	assert.False(t, availability.HasAgents)
}

func Test_GetAvailability_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verification/agents/availability",
		"",
		http.StatusUnauthorized,
	)
}

func Test_GetAvailability_ExcludesSoftDeletedAgents(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	keptAgent := CreateTestVerificationAgent(t, router, admin.Token, "avail-kept-"+uuid.New().String())
	deletedAgent := CreateTestVerificationAgent(t, router, admin.Token, "avail-deleted-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", deletedAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	var availability AvailabilityResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents/availability",
		"Bearer "+member.Token,
		http.StatusOK, &availability,
	)

	assert.Equal(t, int64(1), availability.Count)
	assert.True(t, availability.HasAgents)
	assert.NotEqual(t, uuid.Nil, keptAgent.Agent.ID)
}

// Confirms the live-list JSON does not include internal-only fields like deleted_at.
func Test_ListAgents_JSONShape_NoInternalFields(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	CreateTestVerificationAgent(t, router, admin.Token, "shape-"+uuid.New().String())

	listResponse := test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK,
	)

	listBody := string(listResponse.Body)
	assert.NotContains(t, listBody, "tokenHash")
	assert.NotContains(t, listBody, "deletedAt")

	var agentJsonMaps []map[string]any
	require.NoError(t, json.Unmarshal(listResponse.Body, &agentJsonMaps))
	require.NotEmpty(t, agentJsonMaps)
	_, hasName := agentJsonMaps[0]["name"]
	assert.True(t, hasName)
	_, hasMaxRamGb := agentJsonMaps[0]["maxRamGb"]
	assert.True(t, hasMaxRamGb)
	_, hasMaxRamMb := agentJsonMaps[0]["maxRamMb"]
	assert.False(t, hasMaxRamMb, "RAM is in GB - the legacy 'maxRamMb' key must not appear")
}
