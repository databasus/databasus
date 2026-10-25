package verification_agents

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

func setAgentTimestamps(
	t *testing.T,
	agentID uuid.UUID,
	lastSeenAt *time.Time,
	createdAt time.Time,
) {
	t.Helper()

	err := storage.GetDb().
		Model(&Agent{}).
		Where("id = ?", agentID).
		Updates(map[string]any{
			"last_seen_at": lastSeenAt,
			"created_at":   createdAt,
		}).Error
	require.NoError(t, err)
}

func Test_GetStaleAgents_WhenAgentHeartbeatedRecently_ReturnsEmpty(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "fresh-"+uuid.New().String())
	recentHeartbeat := time.Now().UTC().Add(-1 * time.Minute)
	setAgentTimestamps(t, createdAgent.Agent.ID, &recentHeartbeat, time.Now().UTC().Add(-1*time.Hour))

	staleAgents, err := GetAgentService().GetStaleAgents(5 * time.Minute)
	require.NoError(t, err)

	assert.Empty(t, staleAgents)
}

func Test_GetStaleAgents_WhenAgentLastSeenBeforeThreshold_ReturnsAgent(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "stale-"+uuid.New().String())
	oldHeartbeat := time.Now().UTC().Add(-10 * time.Minute)
	setAgentTimestamps(t, createdAgent.Agent.ID, &oldHeartbeat, oldHeartbeat)

	staleAgents, err := GetAgentService().GetStaleAgents(5 * time.Minute)
	require.NoError(t, err)

	require.Len(t, staleAgents, 1)
	assert.Equal(t, createdAgent.Agent.ID, staleAgents[0].ID)
}

func Test_GetStaleAgents_WhenAgentNeverHeartbeatedButCreatedRecently_ReturnsEmpty(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "newborn-"+uuid.New().String())
	setAgentTimestamps(t, createdAgent.Agent.ID, nil, time.Now().UTC())

	staleAgents, err := GetAgentService().GetStaleAgents(5 * time.Minute)
	require.NoError(t, err)

	assert.Empty(t, staleAgents)
}

func Test_GetStaleAgents_WhenAgentNeverHeartbeatedAndCreatedOld_ReturnsAgent(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "ghost-"+uuid.New().String())
	setAgentTimestamps(t, createdAgent.Agent.ID, nil, time.Now().UTC().Add(-10*time.Minute))

	staleAgents, err := GetAgentService().GetStaleAgents(5 * time.Minute)
	require.NoError(t, err)

	require.Len(t, staleAgents, 1)
	assert.Equal(t, createdAgent.Agent.ID, staleAgents[0].ID)
}

func Test_GetStaleAgents_WhenStaleAgentSoftDeleted_ReturnsEmpty(t *testing.T) {
	cleanupAgents(t)

	router := createTestRouter()
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	createdAgent := CreateTestVerificationAgent(t, router, admin.Token, "retired-"+uuid.New().String())
	oldHeartbeat := time.Now().UTC().Add(-10 * time.Minute)
	setAgentTimestamps(t, createdAgent.Agent.ID, &oldHeartbeat, oldHeartbeat)

	deletedAt := time.Now().UTC()
	err := storage.GetDb().
		Model(&Agent{}).
		Where("id = ?", createdAgent.Agent.ID).
		Update("deleted_at", deletedAt).Error
	require.NoError(t, err)

	staleAgents, err := GetAgentService().GetStaleAgents(5 * time.Minute)
	require.NoError(t, err)

	assert.Empty(t, staleAgents)
}
