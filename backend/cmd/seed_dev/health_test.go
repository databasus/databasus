package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestSeedHealth(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name              string
		isClientToolsLost bool
		isJobStale        bool
		failureReason     string
	}{
		{"creates nothing when a stale agent accompanies broken client tools", true, false, "CLIENT_TOOLS: TOOLS_UNUSABLE"},
		{"creates nothing when a stale agent accompanies stale background jobs", false, true, "stale background job: telemetry_ping"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			var options []apptest.Option
			if scenario.isClientToolsLost {
				options = append(options, apptest.WithClientToolsDir(t.TempDir()))
			}
			router := apptest.New(t, options...)
			admin := router.CreateAdmin(t)
			agent := router.CreateVerificationAgent(t, admin)
			router.ExecuteSQL(t, "UPDATE verification_agents SET created_at = ? WHERE id = ?",
				router.Clock.Now().Add(-time.Hour), agent.Agent.ID)
			if scenario.isJobStale {
				router.Clock.Advance(6 * time.Minute)
			}

			exitCode, output := runSeedForTest(t, buildTestSeedParams(router.URL))

			assert.Equal(t, 1, exitCode)
			assert.Contains(t, output, scenario.failureReason)
			assert.Contains(t, output, "VERIFICATION_AGENTS: AGENTS_STALE")
			assert.NotContains(t, output, "seeding with stale verification agents")
			assertBootstrapAssignment(t, router, admin.UserID)

			users, err := admin.Client.ListUsersWithResponse(t.Context(), nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, users.StatusCode())
			require.NotNil(t, users.JSON200)
			assert.Len(t, users.JSON200.Items, 1)
			assert.Empty(t, listWorkspaces(t, router, admin.Token))
		})
	}
}
