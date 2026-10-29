package verification_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestVerificationAgentHealth(t *testing.T) {
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
			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				app.URL+"/api/v1/system/health",
				nil,
			)
			require.NoError(t, err)
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			status := response.StatusCode
			if scenario.isHealthy {
				assert.Equal(t, http.StatusOK, status, string(body))
			} else {
				assert.Equal(t, http.StatusServiceUnavailable, status, string(body))
				assert.Contains(t, string(body), "verification agents not seen")
			}
		})
	}
}
