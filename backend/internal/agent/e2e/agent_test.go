package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/process"
	"databasus-backend/internal/lib/testwait"
)

type agentProcess struct {
	directory  string
	binary     string
	outputPath string
	engineHost string
}

func newAgentProcess(t *testing.T, binaryDirectory string) *agentProcess {
	t.Helper()
	directory := t.TempDir()
	binary := filepath.Join(directory, binaryName())
	// A link, because a copy can fail to start with ETXTBSY: a process forked by a parallel scenario
	// while the copy is open for writing inherits that descriptor.
	require.NoError(t, os.Link(filepath.Join(binaryDirectory, binaryName()), binary))

	agent := &agentProcess{
		directory:  directory,
		binary:     binary,
		outputPath: filepath.Join(directory, "output.log"),
		engineHost: getEngineHost(),
	}

	t.Cleanup(func() {
		if t.Failed() {
			for _, name := range []string{"output.log", "databasus-verification.log", "databasus-verification.daemon.log"} {
				contents, _ := os.ReadFile(filepath.Join(directory, name))
				t.Logf("agent %s:\n%s", name, contents)
			}
		}
	})

	return agent
}

func (a *agentProcess) spec(args []string) process.Spec {
	return process.Spec{
		Path:       a.binary,
		Args:       args,
		Dir:        a.directory,
		Env:        []string{"DOCKER_HOST=" + a.engineHost},
		StopSignal: syscall.SIGTERM,
		KillAfter:  10 * time.Second,
	}
}

func (a *agentProcess) run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	contents, err := process.CombinedOutput(t.Context(), a.spec(args))
	return string(contents), err
}

func (a *agentProcess) start(t *testing.T, host string, credentials apptest.CreatedVerificationAgent) {
	t.Helper()
	output, err := os.OpenFile(a.outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	spec := a.spec(agentArguments("run", host, credentials))
	spec.Stdout = output
	spec.Stderr = output

	running, err := process.Start(ctx, spec)
	if err != nil {
		cancel()
		_ = output.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() { cancel(); _ = running.Wait(); require.NoError(t, output.Close()) })
}

func agentArguments(command, host string, credentials apptest.CreatedVerificationAgent) []string {
	return []string{
		command,
		"--databasus-host",
		host,
		"--agent-id",
		credentials.Agent.ID.String(),
		"--token",
		credentials.Token,
		"--allow-insecure-http",
		"--skip-update",
		"--max-cpu",
		"1",
		"--max-ram-mb",
		"2048",
		"--max-disk-gb",
		"10",
		"--max-concurrent-jobs",
		"1",
	}
}

func (a *agentProcess) getPID(t *testing.T) int {
	t.Helper()
	var pid int
	testwait.Until(t, "the agent writes its PID", func(context.Context) error {
		contents, err := os.ReadFile(filepath.Join(a.directory, "databasus-verification.lock"))
		if err != nil {
			return err
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(contents)))
		return err
	})
	t.Logf("agent PID %d in %s", pid, a.directory)
	return pid
}

func waitForHeartbeat(
	t *testing.T,
	app *apptest.App,
	admin apptest.Session,
	agentID uuid.UUID,
	after time.Time,
) apptest.VerificationAgent {
	t.Helper()
	var liveAgent apptest.VerificationAgent
	testwait.Until(t, "the agent heartbeats after "+after.String(), func(context.Context) error {
		for _, agent := range app.ListVerificationAgents(t, admin) {
			if agent.ID == agentID && agent.LastSeenAt != nil && agent.LastSeenAt.After(after) {
				liveAgent = agent
				return nil
			}
		}
		return fmt.Errorf("agent %s has no new heartbeat", agentID)
	})
	return liveAgent
}

func waitForVerification(
	t *testing.T,
	app *apptest.App,
	member apptest.Session,
	verificationID uuid.UUID,
) apptest.Verification {
	t.Helper()
	var verification apptest.Verification
	testwait.Until(t, "verification "+verificationID.String()+" reaches an outcome", func(context.Context) error {
		verification = app.GetVerification(t, member, verificationID)
		if verification.Status == "COMPLETED" || verification.Status == "FAILED" {
			return nil
		}
		return fmt.Errorf("verification is %s after %d attempts", verification.Status, verification.AttemptCount)
	})
	return verification
}
