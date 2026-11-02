package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestAgentDaemon(t *testing.T) {
	t.Parallel()

	t.Run("starts one instance and stops it with the daemon commands", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		admin := app.CreateAdmin(t)
		credentials := app.CreateVerificationAgent(t, admin)
		engine := newEngine(t, credentials.Agent.ID)
		gate := newGateway(t, app.URL, gatewayOptions{})
		agent := newAgentProcess(t, currentBinaryDirectory)

		status, err := agent.run(t, "status")
		require.NoError(t, err, status)
		assert.Contains(t, status, "Agent is not running")

		output, err := agent.run(t, agentArguments("start", gate.url, credentials)...)
		require.NoError(t, err, output)
		pidMatch := regexp.MustCompile(`Agent started in background \(PID (\d+)\)`).FindStringSubmatch(output)
		require.Len(t, pidMatch, 2, output)
		pid, err := strconv.Atoi(pidMatch[1])
		require.NoError(t, err)
		t.Logf("daemon PID %d", pid)

		t.Cleanup(func() {
			if syscall.Kill(pid, 0) == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		})

		require.Equal(t, pid, agent.getPID(t))
		_, err = os.Stat(filepath.Join(agent.directory, "databasus-verification.lock"))
		require.NoError(t, err)
		waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Time{})

		status, err = agent.run(t, "status")
		require.NoError(t, err, status)
		assert.Contains(t, status, fmt.Sprintf("Agent is running (PID %d)", pid))
		require.NoError(t, syscall.Kill(pid, 0))

		secondOutput, secondErr := agent.run(t, agentArguments("run", gate.url, credentials)...)
		require.Error(t, secondErr, secondOutput)
		assert.Contains(t, secondOutput, "another instance is already running")
		stopOutput, err := agent.run(t, "stop")
		require.NoError(t, err, stopOutput)

		testwait.Until(t, "the daemon PID exits", func(context.Context) error {
			if syscall.Kill(pid, 0) == nil {
				return fmt.Errorf("PID %d still exists", pid)
			}
			return nil
		})

		status, err = agent.run(t, "status")
		require.NoError(t, err, status)
		assert.Contains(t, status, "Agent is not running")

		requireNoLeftovers(t, engine, credentials.Agent.ID)
	})
}
