package containers

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootContainer(t *testing.T) {
	t.Run("labels the container with the run ID when the run sets one", func(t *testing.T) {
		t.Setenv(testRunIDVariable, "run-under-test")
		minio := StartMinioContainer(t)

		inspectedContainer, err := minio.Container.Inspect(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "run-under-test", inspectedContainer.Config.Labels[testRunLabel])
	})

	t.Run("adds no run label when the run sets no ID", func(t *testing.T) {
		t.Setenv(testRunIDVariable, "")
		minio := StartMinioContainer(t)

		inspectedContainer, err := minio.Container.Inspect(context.Background())

		require.NoError(t, err)
		assert.NotContains(t, inspectedContainer.Config.Labels, testRunLabel)
	})
}

func TestStartNetwork(t *testing.T) {
	t.Run("labels the network with the run ID when the run sets one", func(t *testing.T) {
		t.Setenv(testRunIDVariable, "run-under-test")

		networkLabels := getNetworkLabels(t, StartNetwork(t))

		assert.Equal(t, "run-under-test", networkLabels[testRunLabel])
	})

	t.Run("adds no run label when the run sets no ID", func(t *testing.T) {
		t.Setenv(testRunIDVariable, "")

		networkLabels := getNetworkLabels(t, StartNetwork(t))

		assert.NotContains(t, networkLabels, testRunLabel)
	})
}

func getNetworkLabels(t *testing.T, networkName string) map[string]string {
	t.Helper()

	inspectOutput, err := exec.CommandContext(
		context.Background(), "docker", "network", "inspect", "--format", "{{json .Labels}}", networkName,
	).Output()
	require.NoError(t, err)

	var networkLabels map[string]string
	require.NoError(t, json.Unmarshal(inspectOutput, &networkLabels))

	return networkLabels
}
