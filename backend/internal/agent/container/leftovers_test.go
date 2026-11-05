package container

import (
	"errors"
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const attemptNetworkName = "databasus-verif-v1"

// Pins issue #832: a network the previous attempt left behind used to fail
// every retry with "network with name databasus-verif-<id> already exists".
func TestPrepareAttemptNetwork(t *testing.T) {
	t.Run("removes it and creates network when a previous attempt leaves a network", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addNetwork("old-network", attemptLabels("agent-1", "v1"))

		networkID, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Equal(t, []string{networkID}, host.networkIDs())
		assert.NotEqual(t, "old-network", networkID)
	})
	t.Run(
		"removes container then network then creates when a previous attempt leaves a container and network",
		func(t *testing.T) {
			host := &fakeDockerHost{}
			host.addContainer("old-container", attemptLabels("agent-1", "v1"))
			host.addNetwork("old-network", attemptLabels("agent-1", "v1"))
			filter := formatLabels(attemptLabels("agent-1", "v1"))

			_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

			require.NoError(t, err)
			assert.Equal(t, []string{
				"list containers " + filter,
				"remove container old-container",
				"list networks " + filter,
				"remove network old-network",
				"create network " + attemptNetworkName,
			}, host.log)
		},
	)
	t.Run("creates network when leftovers are already removed", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("old-container", attemptLabels("agent-1", "v1"))
		host.addNetwork("old-network", attemptLabels("agent-1", "v1"))
		host.failRemoval("old-container", cerrdefs.ErrNotFound)
		host.failRemoval("old-network", cerrdefs.ErrNotFound)

		_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Contains(t, host.log, "create network "+attemptNetworkName)
	})
	t.Run("creates network without removals when there are no leftovers", func(t *testing.T) {
		host := &fakeDockerHost{}
		filter := formatLabels(attemptLabels("agent-1", "v1"))

		_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Equal(t, []string{
			"list containers " + filter,
			"list networks " + filter,
			"create network " + attemptNetworkName,
		}, host.log)
	})
	t.Run("returns leftover error and creates no network when leftover removal fails", func(t *testing.T) {
		for _, refused := range []string{"old-container", "old-network"} {
			t.Run(refused, func(t *testing.T) {
				host := &fakeDockerHost{}
				host.addContainer("old-container", attemptLabels("agent-1", "v1"))
				host.addNetwork("old-network", attemptLabels("agent-1", "v1"))
				host.failRemoval(refused, errors.New("removal refused"))

				_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

				require.Error(t, err)
				assert.Contains(t, err.Error(), "could not remove the leftovers of an earlier attempt")
				assert.Contains(t, err.Error(), refused)
				assert.NotContains(t, host.log, "create network "+attemptNetworkName)
			})
		}
	})
	t.Run("returns leftover error and creates no network when listing fails", func(t *testing.T) {
		for name, host := range map[string]*fakeDockerHost{
			"containers": {listContainersErr: errors.New("docker unreachable")},
			"networks":   {listNetworksErr: errors.New("docker unreachable")},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

				require.Error(t, err)
				assert.Contains(t, err.Error(), "could not remove the leftovers of an earlier attempt")
				assert.NotContains(t, host.log, "create network "+attemptNetworkName)
			})
		}
	})
	t.Run("leaves its resources in place when another verification is running", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("running-container", attemptLabels("agent-1", "v2"))
		host.addNetwork("running-network", attemptLabels("agent-1", "v2"))

		networkID, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Equal(t, []string{"running-container"}, host.containerIDs())
		assert.Equal(t, []string{"running-network", networkID}, host.networkIDs())
	})
	t.Run("leaves its resources in place when another agent has the same verification", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("foreign-container", attemptLabels("agent-2", "v1"))
		host.addNetwork("foreign-network", attemptLabels("agent-2", "v1"))

		networkID, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Equal(t, []string{"foreign-container"}, host.containerIDs())
		assert.Equal(t, []string{"foreign-network", networkID}, host.networkIDs())
	})
	t.Run("filters by agent and verification labels", func(t *testing.T) {
		host := &fakeDockerHost{}

		_, err := prepareAttemptNetwork(t.Context(), host, attemptNetworkName, attemptLabels("agent-1", "v1"))

		require.NoError(t, err)
		assert.Equal(t, []string{
			"list containers " + LabelAgentID + "=agent-1," + LabelVerificationID + "=v1",
			"list networks " + LabelAgentID + "=agent-1," + LabelVerificationID + "=v1",
			"create network " + attemptNetworkName,
		}, host.log)
	})
}
