package containers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Kubelet grants a shutdown window even with zero deletion grace, so the primary container
// must receive SIGKILL before its pod is deleted.
func (cluster *CNPGCluster) KillPrimary(t *testing.T) RouteIdentity {
	t.Helper()

	previous := cluster.GetIdentity(t)
	cluster.RequireReady(t)
	candidate := cluster.GetReplica(previous.Node)
	cluster.RequireReplay(t, candidate)
	killedPod := cluster.getPodID(t, previous.Node)
	cluster.killedPods[previous.Node] = killedPod

	cluster.killPrimaryContainer(t, previous.Node)
	cluster.kubectl(t, "delete", "pod", previous.Node, "--grace-period=0", "--force", "--wait=false")

	current := cluster.waitForPromotion(t, promotion{Previous: previous, Candidate: candidate})
	cluster.RequireKilledPodGone(t, previous.Node)

	return current
}

// RequireKilledPodGone checks that the pod KillPrimary deleted no longer exists; CloudNativePG may
// have started a new pod for the same node, which joins as a standby.
func (cluster *CNPGCluster) RequireKilledPodGone(t *testing.T, node string) {
	t.Helper()

	killedPod, hasKilled := cluster.killedPods[node]
	require.True(t, hasKilled, "no pod of %s was killed", node)
	require.NotEqual(t, killedPod, cluster.getPodID(t, node), "the killed pod of %s must be gone", node)
}

func (cluster *CNPGCluster) killPrimaryContainer(t *testing.T, pod string) {
	t.Helper()

	containerURI := cluster.kubectl(t, "get", "pod", pod, "-o",
		`jsonpath={.status.containerStatuses[?(@.name=="postgres")].containerID}`)
	containerID, isContainerd := strings.CutPrefix(strings.TrimSpace(containerURI), "containerd://")
	require.True(t, isContainerd, "the primary must run in k3s containerd: %s", containerURI)
	require.NotEmpty(t, containerID)

	output, err := cluster.K3s.exec(t.Context(), dockerExec{Command: []string{
		"ctr", "-n", "k8s.io", "tasks", "kill", "--all", "--signal", "SIGKILL", containerID,
	}})
	require.NoError(t, err)
	require.Zero(t, output.ExitCode, "SIGKILL of primary container %s failed: %s", containerID, output.Stderr)
	t.Logf("sent SIGKILL to every process of primary %s in container %s", pod, containerID)
}

func (cluster *CNPGCluster) getPodID(t *testing.T, pod string) string {
	t.Helper()

	output, err := cluster.K3s.Kubectl(t.Context(), KubectlCommand{Arguments: []string{
		"-n", cluster.Namespace, "get", "pod", pod, "--ignore-not-found", "-o", "jsonpath={.metadata.uid}",
	}})
	require.NoError(t, err)

	return strings.TrimSpace(string(output))
}
