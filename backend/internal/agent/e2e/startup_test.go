package e2e_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestAgentStartup(t *testing.T) {
	t.Parallel()

	t.Run("removes its leftovers while keeping another agent's network", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		admin := app.CreateAdmin(t)
		credentials := app.CreateVerificationAgent(t, admin)
		engine := newEngine(t, credentials.Agent.ID)

		createStrayContainer(t, engine, credentials.Agent.ID)
		createNetwork(t, engine, credentials.Agent.ID, uuid.New())

		otherAgentID := uuid.New()
		otherEngine := newEngine(t, otherAgentID)
		otherNetworkID := createNetwork(t, otherEngine, otherAgentID, uuid.New())

		gate := newGateway(t, app.URL, gatewayOptions{})
		agent := newAgentProcess(t, currentBinaryDirectory)
		agent.start(t, gate.url, credentials)
		waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Time{})
		requireNoLeftovers(t, engine, credentials.Agent.ID)

		networks, err := otherEngine.NetworkList(
			t.Context(),
			client.NetworkListOptions{Filters: agentFilter(otherAgentID)},
		)
		require.NoError(t, err)
		require.Len(t, networks.Items, 1)
		assert.Equal(t, otherNetworkID, networks.Items[0].ID)
	})
}
