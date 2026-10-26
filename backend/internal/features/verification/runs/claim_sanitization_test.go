package verification_runs

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func TestClaimVerification(t *testing.T) {
	t.Run("assigns only the type and the version of the database without connection settings", func(t *testing.T) {
		router := createTestRouter()
		owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
		workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
		defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

		testStorage := storages_bridge.CreateTestStorage(workspace.ID)
		defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

		notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
		defer notifiers_bridge.RemoveTestNotifier(notifier)

		database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
		defer databases_bridge.RemoveTestDatabase(t.Context(), database)

		backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
		agent := verification_agents.CreateTestVerificationAgent(
			t, router, owner.Token, "sanitize-"+uuid.New().String(),
		)
		defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

		EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

		claimResponse := test_utils.MakePostRequest(
			t, router,
			"/api/v1/agent/verifications/"+agent.Agent.ID.String()+"/claim",
			"Bearer "+agent.Token,
			ClaimRequest{Capacity: AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2}},
			http.StatusOK,
		)

		var assignmentBody struct {
			Database map[string]json.RawMessage `json:"database"`
		}
		require.NoError(t, json.Unmarshal(claimResponse.Body, &assignmentBody))

		assert.Len(t, assignmentBody.Database, 2)
		assert.JSONEq(t, `"POSTGRES_LOGICAL"`, string(assignmentBody.Database["type"]))
		assert.JSONEq(t, `{"version":"16"}`, string(assignmentBody.Database["postgresqlLogical"]))
	})
}
