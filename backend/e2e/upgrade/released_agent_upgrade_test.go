package upgrade

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/process"
	"databasus-backend/internal/lib/testwait"
)

type releasedAgentUpgradeSpec struct {
	docker       *dockerHost
	api          *apiClient
	currentImage string
	upgradeData  writtenUpgradeData
}

// Both scenarios share the upgraded server, its backup and its pending verification.
func checkReleasedAgentUpgrade(t *testing.T, spec releasedAgentUpgradeSpec) {
	t.Helper()

	releasedBinary := extractReleasedAgent(t, spec.docker)
	var server struct {
		Version string `json:"version"`
	}
	require.NoError(t, spec.api.get(t.Context(), "/system/version", &server))
	require.NotEqual(t, "3.64.0", server.Version)

	for _, scenario := range []struct {
		name       string
		isReadOnly bool
	}{
		{"updates a 3.64.0 agent and accepts its heartbeat after the old heartbeat is rejected", false},
		{"restores verification after manual replacement when the agent cannot replace its binary", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			gateway := newAgentGateway(t, spec.api)
			agent := newUpgradeAgent(t, upgradeAgentSpec{
				api: spec.api, docker: spec.docker, image: spec.currentImage, releasedBinary: releasedBinary,
				gatewayURL: gateway.url, isReadOnly: scenario.isReadOnly,
			})

			require.NoError(t, agent.startWithInitialConfig(t.Context()))

			agent.waitForSignal(t, "the released heartbeat to receive 400", gateway.rejectedHeartbeat)
			gateway.releaseDiscovery()

			if scenario.isReadOnly {
				testwait.Until(t, "the real updater to fail writing its replacement", func(ctx context.Context) error {
					logs, err := agent.getLogs(ctx)
					if err != nil {
						return err
					}

					if !strings.Contains(logs, "read-only file system") {
						return fmt.Errorf("no read-only update failure yet")
					}

					return nil
				})

				require.Equal(t, "3.64.0", getAgentBinaryVersion(t, agent.binaryPath))

				require.NoError(t, agent.stop(t.Context()))

				agent.installCurrentBinary(t, spec.api)
				require.Equal(t, server.Version, getAgentBinaryVersion(t, agent.binaryPath))

				configPath := filepath.Join(agent.workingPath, "databasus-verification.json")
				preservedConfigPath := configPath + ".saved"
				require.NoError(t, os.Rename(configPath, preservedConfigPath))

				require.ErrorIs(t, agent.startFromSavedConfig(t.Context()), os.ErrNotExist)

				var agentExitError *exec.ExitError

				restartOutput, restartErr := process.CombinedOutput(t.Context(), process.Spec{
					Path: agent.binaryPath, Args: []string{"run", "--skip-update"}, Dir: agent.workingPath,
				})
				require.ErrorAs(t, restartErr, &agentExitError)

				require.NotZero(t, agentExitError.ExitCode())
				require.Contains(t, string(restartOutput), "no databasus-verification.json found")
				require.Contains(t, string(restartOutput), "databasus-host is required")

				require.NoError(t, os.Rename(preservedConfigPath, configPath))

				require.NoError(t, agent.startFromSavedConfig(t.Context()))
			}

			agent.waitForSignal(t, "the replacement agent heartbeat to receive 200", gateway.acceptedHeartbeat)
			require.Equal(t, server.Version, getAgentBinaryVersion(t, agent.binaryPath))
			checkUpgradedAgentCapacity(t, spec.api, agent.agentID)

			verificationID := getOrCreateAgentUpgradeVerification(t, spec.api, spec.upgradeData)
			gateway.releaseClaims()

			checkAgentBackupVerification(
				t,
				spec.api,
				assignedVerification{verificationID: verificationID, agentID: agent.agentID},
			)

			agent.waitForRestoreCleanup(t)
			require.NoError(t, agent.stop(t.Context()))
			agent.waitForRestoreCleanup(t)
		})
	}
}
