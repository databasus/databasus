package verification_runs

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
)

// Test_ClaimVerification_AssignmentDatabase_StripsSensitiveData locks down the
// agent-facing wire shape: anything that could leak credentials, tokens or
// notifier secrets must be empty when the JobAssignment leaves the server.
func Test_ClaimVerification_AssignmentDatabase_StripsSensitiveData(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	require.NotNil(t, database.PostgresqlLogical)
	require.NotEmpty(t, database.PostgresqlLogical.Password,
		"fixture must persist a password so we can prove HideSensitiveData stripped it")

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(
		t, router, owner.Token, "sanitize-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	require.NotNil(t, assignment.Database, "assignment must carry the database row")

	cases := []struct {
		name  string
		check func(t *testing.T, db *databases.Database)
	}{
		{
			name: "postgres password is stripped",
			check: func(t *testing.T, db *databases.Database) {
				require.NotNil(t, db.PostgresqlLogical)
				assert.Empty(t, db.PostgresqlLogical.Password)
			},
		},
		{
			name: "notifiers are stripped (carry webhook URLs and SMTP creds)",
			check: func(t *testing.T, db *databases.Database) {
				assert.Empty(t, db.Notifiers)
			},
		},
		{
			name: "non-sensitive identifying fields are preserved",
			check: func(t *testing.T, db *databases.Database) {
				assert.Equal(t, database.ID, db.ID)
				assert.Equal(t, databases.DatabaseTypePostgresLogical, db.Type)
				require.NotNil(t, db.PostgresqlLogical)
				assert.Equal(t, "16", string(db.PostgresqlLogical.Version))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, assignment.Database)
		})
	}
}
