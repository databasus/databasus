package container

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/stretchr/testify/assert"
)

func TestPurgeAgentContainers(t *testing.T) {
	t.Run("removes every container and network", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addContainer("c2", attemptLabels("agent-1", "v2"))
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))
		host.addNetwork("n2", attemptLabels("agent-1", "v2"))

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Empty(t, host.containerIDs())
		assert.Empty(t, host.networkIDs())
	})
	t.Run("removes nothing when nothing has this agent label", func(t *testing.T) {
		host := &fakeDockerHost{}

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Equal(t, []string{
			"list containers " + LabelAgentID + "=agent-1",
			"list networks " + LabelAgentID + "=agent-1",
		}, host.log)
	})
	t.Run("continues with remaining resources when container removal fails", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addContainer("c2", attemptLabels("agent-1", "v2"))
		host.addNetwork("n2", attemptLabels("agent-1", "v2"))
		host.failRemoval("c1", errors.New("container in use"))

		assert.NotPanics(t, func() {
			purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))
		})

		assert.Equal(t, []string{"c1"}, host.containerIDs())
		assert.Empty(t, host.networkIDs())
	})
	t.Run("removes network when a network has no container", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Empty(t, host.networkIDs())
	})
	t.Run("removes container before network when a container and network exist", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Equal(t, []string{
			"list containers " + LabelAgentID + "=agent-1",
			"remove container c1",
			"list networks " + LabelAgentID + "=agent-1",
			"remove network n1",
		}, host.log)
	})
	t.Run("leaves them in place when resources belong to another agent or have no agent label", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("foreign-container", attemptLabels("agent-2", "v1"))
		host.addNetwork("foreign-network", attemptLabels("agent-2", "v1"))
		host.addNetwork("unrelated-network", map[string]string{"com.example.app": "web"})

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Equal(t, []string{"foreign-container"}, host.containerIDs())
		assert.Equal(t, []string{"foreign-network", "unrelated-network"}, host.networkIDs())
	})
	t.Run("filters by agent label only", func(t *testing.T) {
		host := &fakeDockerHost{}

		purgeAgentContainers(t.Context(), host, "agent-1", slog.New(slog.DiscardHandler))

		assert.Equal(t, []string{
			"list containers " + formatLabels(agentLabels("agent-1")),
			"list networks " + formatLabels(agentLabels("agent-1")),
		}, host.log)
	})
	t.Run("logs and removes containers when network listing fails", func(t *testing.T) {
		host := &fakeDockerHost{listNetworksErr: errors.New("docker unreachable")}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))
		log, logs := captureLogger()

		assert.NotPanics(t, func() {
			purgeAgentContainers(t.Context(), host, "agent-1", log)
		})

		assert.Empty(t, host.containerIDs())
		assert.Equal(t, []string{"n1"}, host.networkIDs())
		assert.Contains(t, logs.String(), "level=WARN msg=\"startup purge left some resources in place\"")
		assert.Contains(t, logs.String(), "list networks: docker unreachable")
	})
	t.Run("continues with remaining resources when removal returns not found", func(t *testing.T) {
		host := &fakeDockerHost{}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addContainer("c2", attemptLabels("agent-1", "v2"))
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))
		host.failRemoval("c1", cerrdefs.ErrNotFound)
		log, logs := captureLogger()

		purgeAgentContainers(t.Context(), host, "agent-1", log)

		assert.Equal(t, []string{
			"list containers " + LabelAgentID + "=agent-1",
			"remove container c1",
			"remove container c2",
			"list networks " + LabelAgentID + "=agent-1",
			"remove network n1",
		}, host.log)
		assert.Empty(t, host.networkIDs())
		assert.NotContains(t, logs.String(), "level=WARN")
		assert.Contains(t, logs.String(), "startup purge removed stale containers and networks")
		assert.Contains(t, logs.String(), "removed=3")
	})
	t.Run("logs and removes networks when container listing fails", func(t *testing.T) {
		host := &fakeDockerHost{listContainersErr: errors.New("docker unreachable")}
		host.addContainer("c1", attemptLabels("agent-1", "v1"))
		host.addNetwork("n1", attemptLabels("agent-1", "v1"))
		host.addNetwork("n2", attemptLabels("agent-1", "v2"))
		host.failRemoval("n1", errors.New("network has active endpoints"))
		log, logs := captureLogger()

		assert.NotPanics(t, func() {
			purgeAgentContainers(t.Context(), host, "agent-1", log)
		})

		assert.Equal(t, []string{"c1"}, host.containerIDs())
		assert.Equal(t, []string{"n1"}, host.networkIDs())
		assert.Contains(t, logs.String(), "level=WARN msg=\"startup purge left some resources in place\"")
		assert.Contains(t, logs.String(), "list containers: docker unreachable")
		assert.Contains(t, logs.String(), "remove network n1 of verification v1: network has active endpoints")
	})
}

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	return slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug})), buffer
}
