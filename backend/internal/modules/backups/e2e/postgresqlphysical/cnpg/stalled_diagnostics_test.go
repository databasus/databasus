package cnpg_test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func stallK3sAPI(t *testing.T, containerID string) {
	t.Helper()

	testwait.Within(3*time.Minute).
		Until(t, "the owned Kubernetes API accepts requests", func(ctx context.Context) error {
			output, err := exec.CommandContext(ctx, "docker", "exec", containerID, "kubectl", "--request-timeout=2s", "get", "--raw=/version").
				CombinedOutput()
			if err != nil {
				return fmt.Errorf("Kubernetes readiness: %w: %s", err, output)
			}

			return nil
		})

	// k3s init spawns the API server from a Go thread; stopping PID 1 leaves that server running.
	output, err := exec.CommandContext(t.Context(), "docker", "exec", containerID, "sh", "-c", `set -- $(cat /proc/1/task/*/children); [ "$#" -gt 0 ] && kill -STOP "$@"`).
		CombinedOutput()
	require.NoError(t, err, "%s", output)

	output, err = exec.CommandContext(t.Context(), "docker", "exec", containerID, "kubectl", "--request-timeout=2s", "get", "--raw=/version").
		CombinedOutput()
	require.Error(t, err, "the stopped Kubernetes API still answered: %s", output)
	require.True(t, isKubectlTimeout(string(output)), "kubectl failed without a timeout: %s", output)

	t.Log("stopped the owned k3s API process before startup cancellation")
}

// kubectl reports its timeout as "context deadline exceeded" once connected, and as
// "Client.Timeout exceeded" when the stopped server never completes the connection.
func isKubectlTimeout(output string) bool {
	return strings.Contains(output, "deadline exceeded") || strings.Contains(output, "Client.Timeout exceeded")
}
