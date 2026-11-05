package upgrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func checkUpgradedAgentCapacity(t *testing.T, api *apiClient, agentID string) {
	t.Helper()

	var agents struct {
		Items []struct {
			ID                string     `json:"id"`
			MaxCPU            int        `json:"maxCpu"`
			MaxRAMMb          int        `json:"maxRamMb"`
			MaxDiskGb         int        `json:"maxDiskGb"`
			MaxConcurrentJobs int        `json:"maxConcurrentJobs"`
			LastSeenAt        *time.Time `json:"lastSeenAt"`
		} `json:"items"`
	}

	require.NoError(t, api.get(t.Context(), "/verification-agents?limit=100", &agents))

	for _, agent := range agents.Items {
		if agent.ID == agentID {
			require.NotNil(t, agent.LastSeenAt)
			require.Equal(t, 1, agent.MaxCPU)
			require.Equal(t, 2048, agent.MaxRAMMb)
			require.Equal(t, 10, agent.MaxDiskGb)
			require.Equal(t, 1, agent.MaxConcurrentJobs)

			return
		}
	}

	t.Fatalf("upgraded agent %s is not listed", agentID)
}
