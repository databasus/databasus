package e2e_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestAgentUpdate(t *testing.T) {
	t.Parallel()

	t.Run("keeps heartbeating with its capacity after a background update", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		admin := app.CreateAdmin(t)
		credentials := app.CreateVerificationAgent(t, admin)
		engine := newEngine(t, credentials.Agent.ID)
		gate := newGateway(t, app.URL, gatewayOptions{})
		agent := newAgentProcess(t, olderBinaryDirectory)

		version, err := agent.run(t, "version")
		require.NoError(t, err, version)
		require.Equal(t, olderVersion, strings.TrimSpace(version))

		agent.start(t, gate.url, credentials)
		pid := agent.getPID(t)
		beforeUpdate := waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Time{})
		assert.Equal(t, 2048, beforeUpdate.MaxRAMMb)

		testwait.Until(
			t,
			"the binary on disk prints the server version and the agent re-executes",
			func(context.Context) error {
				version, err := agent.run(t, "version")
				if err != nil {
					return err
				}
				if strings.TrimSpace(version) != serverVersion {
					return fmt.Errorf("binary is still %s", strings.TrimSpace(version))
				}

				contents, err := os.ReadFile(agent.outputPath)
				if err != nil {
					return err
				}

				for _, message := range []string{"agent binary updated", "background upgrade complete, restarting...", "re-executing after upgrade..."} {
					if !strings.Contains(string(contents), message) {
						return fmt.Errorf("agent has not logged %s", message)
					}
				}
				return nil
			},
		)

		require.NoError(t, syscall.Kill(pid, 0))
		assert.Equal(t, pid, agent.getPID(t))

		afterUpdate := waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Now().UTC())
		assert.Equal(t, 2048, afterUpdate.MaxRAMMb)
		assert.True(t, afterUpdate.LastSeenAt.After(*beforeUpdate.LastSeenAt))

		requireNoLeftovers(t, engine, credentials.Agent.ID)
	})
}
