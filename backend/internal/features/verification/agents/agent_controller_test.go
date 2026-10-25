package verification_agents

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_Heartbeat_WhenTokenIsValid_UpdatesCapacityAndLastSeen(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-ok-"+uuid.New().String())

	beforeHeartbeat := time.Now().UTC()

	var heartbeatResponse HeartbeatResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		heartbeatPathFor(createdAgent.Agent.ID),
		"Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 8, MaxRAMGb: 32, MaxDiskGb: 500, MaxConcurrentJobs: 4},
		http.StatusOK, &heartbeatResponse,
	)

	assert.WithinDuration(t, time.Now().UTC(), heartbeatResponse.LastSeenAt, 10*time.Second)
	assert.True(t, !heartbeatResponse.LastSeenAt.Before(beforeHeartbeat))

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+admin.Token,
		http.StatusOK, &listedAgents,
	)

	var persistedAgent *Agent
	for i := range listedAgents {
		if listedAgents[i].ID == createdAgent.Agent.ID {
			persistedAgent = &listedAgents[i]
		}
	}
	require.NotNil(t, persistedAgent)
	assert.Equal(t, 8, persistedAgent.MaxCPU)
	assert.Equal(t, 32, persistedAgent.MaxRAMGb)
	assert.Equal(t, 500, persistedAgent.MaxDiskGb)
	assert.Equal(t, 4, persistedAgent.MaxConcurrentJobs)
	require.NotNil(t, persistedAgent.LastSeenAt)
	assert.WithinDuration(t, time.Now().UTC(), *persistedAgent.LastSeenAt, 10*time.Second)
}

func Test_Heartbeat_TwiceUpdatesSameRow(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-twice-"+uuid.New().String())
	heartbeatPath := heartbeatPathFor(createdAgent.Agent.ID)

	var firstHeartbeat HeartbeatResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router, heartbeatPath, "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 1}, http.StatusOK, &firstHeartbeat,
	)

	time.Sleep(10 * time.Millisecond)

	var secondHeartbeat HeartbeatResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router, heartbeatPath, "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 2}, http.StatusOK, &secondHeartbeat,
	)

	assert.True(t, secondHeartbeat.LastSeenAt.After(firstHeartbeat.LastSeenAt) ||
		secondHeartbeat.LastSeenAt.Equal(firstHeartbeat.LastSeenAt))

	var listedAgents []Agent
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/verification/agents",
		"Bearer "+admin.Token, http.StatusOK, &listedAgents,
	)

	matchCount := 0
	for _, agent := range listedAgents {
		if agent.ID == createdAgent.Agent.ID {
			matchCount++
			assert.Equal(t, 2, agent.MaxCPU)
		}
	}
	assert.Equal(t, 1, matchCount)
}

func Test_Heartbeat_PersistsRamAsGb_NotMb(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-gb-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxRAMGb: 64}, http.StatusOK,
	)

	listResponse := test_utils.MakeGetRequest(
		t, router, "/api/v1/verification/agents",
		"Bearer "+admin.Token, http.StatusOK,
	)

	listBody := string(listResponse.Body)
	assert.Contains(t, listBody, `"maxRamGb":64`)
	assert.NotContains(t, listBody, "maxRamMb")
}

func Test_Heartbeat_WhenNoToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-notoken-"+uuid.New().String())

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "",
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")
}

func Test_Heartbeat_WhenInvalidToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-bad-"+uuid.New().String())

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID),
		"Bearer "+uuid.New().String(),
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")
}

func Test_Heartbeat_WhenUserJwtInsteadOfAgentToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-jwt-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+admin.Token,
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
}

func Test_Heartbeat_WhenAgentIdInvalidUuid_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, "/api/v1/agent/verification/not-a-uuid/heartbeat",
		"Bearer "+uuid.New().String(),
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")
}

func Test_Heartbeat_WhenAgentIdUnknown_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, heartbeatPathFor(uuid.New()),
		"Bearer "+uuid.New().String(),
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")
}

func Test_Heartbeat_WhenAgentIdMismatchesToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	agentA := CreateTestVerificationAgent(t, router, admin.Token, "hb-mix-a-"+uuid.New().String())
	agentB := CreateTestVerificationAgent(t, router, admin.Token, "hb-mix-b-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(agentA.Agent.ID), "Bearer "+agentA.Token,
		HeartbeatRequest{MaxCPU: 1}, http.StatusOK,
	)

	var lastSeenBeforeMismatch *time.Time
	storage.GetDb().Raw(`SELECT last_seen_at FROM verification_agents WHERE id = ?`,
		agentA.Agent.ID).Scan(&lastSeenBeforeMismatch)
	require.NotNil(t, lastSeenBeforeMismatch)

	time.Sleep(20 * time.Millisecond)

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, heartbeatPathFor(agentA.Agent.ID), "Bearer "+agentB.Token,
		HeartbeatRequest{MaxCPU: 99}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")

	var lastSeenAfterMismatch *time.Time
	storage.GetDb().Raw(`SELECT last_seen_at FROM verification_agents WHERE id = ?`,
		agentA.Agent.ID).Scan(&lastSeenAfterMismatch)
	require.NotNil(t, lastSeenAfterMismatch)
	assert.True(t, lastSeenAfterMismatch.Equal(*lastSeenBeforeMismatch),
		"mismatched-token heartbeat must not update last_seen_at")

	var persistedMaxCPU int
	storage.GetDb().Raw(`SELECT max_cpu FROM verification_agents WHERE id = ?`,
		agentA.Agent.ID).Scan(&persistedMaxCPU)
	assert.Equal(t, 1, persistedMaxCPU)
}

func Test_Heartbeat_AfterRotateToken_OldTokenReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-rotate-"+uuid.New().String())
	originalToken := createdAgent.Token

	var rotation RotateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s/rotate-token", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		map[string]any{},
		http.StatusOK, &rotation,
	)

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+originalToken,
		HeartbeatRequest{}, http.StatusUnauthorized,
	)

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+rotation.Token,
		HeartbeatRequest{}, http.StatusOK,
	)
}

func Test_Heartbeat_AfterDelete_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-deleted-"+uuid.New().String())

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	heartbeatResponse := test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+createdAgent.Token,
		HeartbeatRequest{}, http.StatusUnauthorized,
	)
	assert.Contains(t, string(heartbeatResponse.Body), "invalid agent credentials")
}

func Test_Heartbeat_AfterDelete_DoesNotUpdateLastSeen(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-no-update-"+uuid.New().String())
	heartbeatPath := heartbeatPathFor(createdAgent.Agent.ID)

	test_utils.MakePostRequest(
		t, router, heartbeatPath, "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 1}, http.StatusOK,
	)

	var lastSeenBeforeDelete *time.Time
	storage.GetDb().Raw(`SELECT last_seen_at FROM verification_agents WHERE id = ?`,
		createdAgent.Agent.ID).Scan(&lastSeenBeforeDelete)
	require.NotNil(t, lastSeenBeforeDelete)

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/verification/agents/%s", createdAgent.Agent.ID),
		"Bearer "+admin.Token,
		http.StatusNoContent,
	)

	time.Sleep(20 * time.Millisecond)

	test_utils.MakePostRequest(
		t, router, heartbeatPath, "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 99}, http.StatusUnauthorized,
	)

	var lastSeenAfterDelete *time.Time
	storage.GetDb().Raw(`SELECT last_seen_at FROM verification_agents WHERE id = ?`,
		createdAgent.Agent.ID).Scan(&lastSeenAfterDelete)
	require.NotNil(t, lastSeenAfterDelete)
	assert.True(t, lastSeenAfterDelete.Equal(*lastSeenBeforeDelete),
		"heartbeat against a soft-deleted agent must not update last_seen_at")

	var persistedMaxCPU int
	storage.GetDb().Raw(`SELECT max_cpu FROM verification_agents WHERE id = ?`,
		createdAgent.Agent.ID).Scan(&persistedMaxCPU)
	assert.Equal(t, 1, persistedMaxCPU,
		"heartbeat against a soft-deleted agent must not update capacity")
}

func Test_Heartbeat_RejectsNegativeCapacity(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-neg-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+createdAgent.Token,
		map[string]any{"maxCpu": -1}, http.StatusBadRequest,
	)
}

func Test_Heartbeat_OverPerAgentRateLimit_Returns429(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-burst-"+uuid.New().String())
	heartbeatPath := heartbeatPathFor(createdAgent.Agent.ID)

	for range rateLimitAgentMax {
		response := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
			Method:    http.MethodPost,
			URL:       heartbeatPath,
			AuthToken: "Bearer " + createdAgent.Token,
			Body:      HeartbeatRequest{MaxCPU: 1},
		})
		require.Equal(t, http.StatusOK, response.StatusCode,
			"requests within the per-agent budget should succeed")
	}

	response := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:    http.MethodPost,
		URL:       heartbeatPath,
		AuthToken: "Bearer " + createdAgent.Token,
		Body:      HeartbeatRequest{MaxCPU: 1},
	})
	assert.Equal(t, http.StatusTooManyRequests, response.StatusCode,
		"the request past the per-agent budget should be rate-limited")
	assert.Contains(t, string(response.Body), "too many requests")
}

func Test_Heartbeat_InvalidUuidDoesNotConsumeRateLimit(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "hb-garbage-"+uuid.New().String())

	for range rateLimitAgentMax * 2 {
		response := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
			Method:    http.MethodPost,
			URL:       "/api/v1/agent/verification/not-a-uuid/heartbeat",
			AuthToken: "Bearer " + createdAgent.Token,
			Body:      HeartbeatRequest{},
		})
		require.Equal(t, http.StatusUnauthorized, response.StatusCode,
			"garbage-UUID requests must always reject as 401, never as 429")
	}

	test_utils.MakePostRequest(
		t, router, heartbeatPathFor(createdAgent.Agent.ID), "Bearer "+createdAgent.Token,
		HeartbeatRequest{MaxCPU: 1}, http.StatusOK,
	)
}

func Test_AdminEndpoint_WithAgentToken_ReturnsUnauthorized(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "cross-"+uuid.New().String())

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/verification/agents",
		"Bearer "+createdAgent.Token,
		CreateAgentRequest{Name: "should-not-work"},
		http.StatusUnauthorized,
	)
}
