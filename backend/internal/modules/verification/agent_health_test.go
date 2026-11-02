package verification_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationAgentHealth(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name             string
		isOld            bool
		hasHeartbeat     bool
		isDeleted        bool
		isHealthy        bool
		isHeartbeatStale bool
	}{
		{name: "stays healthy when an agent heartbeats recently", isOld: true, hasHeartbeat: true, isDeleted: false, isHealthy: true, isHeartbeatStale: false},
		{name: "reports an agent when it never heartbeats and was created long ago", isOld: true, hasHeartbeat: false, isDeleted: false, isHealthy: false, isHeartbeatStale: false},
		{name: "stays healthy when a new agent has never heartbeated", isOld: false, hasHeartbeat: false, isDeleted: false, isHealthy: true, isHeartbeatStale: false},
		{name: "ignores a stale agent after deletion", isOld: true, hasHeartbeat: false, isDeleted: true, isHealthy: true, isHeartbeatStale: false},
		{name: "reports an agent when its last heartbeat is stale", isOld: false, hasHeartbeat: true, isDeleted: false, isHealthy: false, isHeartbeatStale: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			agent := app.CreateVerificationAgent(t, admin)

			if scenario.isOld {
				app.Clock.Advance(6 * time.Minute)
			}
			if scenario.hasHeartbeat {
				status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
				require.Equal(t, http.StatusOK, status, string(body))
			}
			if scenario.isHeartbeatStale {
				app.Clock.Advance(6 * time.Minute)
			}
			if scenario.isDeleted {
				deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), agent.Agent.ID)
				require.NoError(t, err)
				require.Equal(t, http.StatusNoContent, deleted.StatusCode())
			}

			app.RunJobPass(t, "backups_logical_scheduler")
			response, err := app.NewClient(t).GetInstanceHealthWithResponse(t.Context())
			require.NoError(t, err)

			var checks []apiclient.HealthCheck
			if response.JSON200 != nil {
				checks = response.JSON200.Checks
			} else {
				require.NotNil(t, response.ApplicationproblemJSON503, string(response.Body))
				checks = response.ApplicationproblemJSON503.Checks
			}

			assert.NotContains(t, string(response.Body), agent.Token)
			assert.NotContains(t, string(response.Body), agent.Agent.ID.String())

			hasAgentCheck := false
			for _, check := range checks {
				if check.Name != "VERIFICATION_AGENTS" {
					continue
				}
				hasAgentCheck = true
				if scenario.isHealthy {
					assert.Equal(t, apiclient.HealthCheckOutcomePASSED, check.Outcome)
					assert.Equal(t, 0, *check.StaleAgentCount)
				} else {
					assert.Equal(t, apiclient.HealthCheckOutcomeFAILED, check.Outcome)
					assert.Equal(t, 1, *check.StaleAgentCount)
				}
			}
			assert.True(t, hasAgentCheck)
		})
	}
}
