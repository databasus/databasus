package browsertests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type browserChild struct {
	context      context.Context
	frontendRoot string
	runRoot      string
	phase        string
	runID        string
	address      string
	email        string
	password     string
}

func runBrowserChild(t *testing.T, child browserChild) {
	t.Helper()
	command := exec.CommandContext(
		child.context,
		"node",
		filepath.Join(child.frontendRoot, "tooling", "runBrowserE2EInside.ts"),
		child.phase,
	)
	command.Env = append(
		command.Environ(),
		"DATABASUS_BROWSER_RUN_ROOT="+child.runRoot,
		"DATABASUS_BROWSER_RUN_ID="+child.runID,
		"BROWSER_BASE_URL="+child.address,
		"E2E_EMAIL="+child.email,
		"E2E_PASSWORD="+child.password,
	)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.WaitDelay = 10 * time.Second
	command.Cancel = func() error { return command.Process.Signal(syscall.SIGTERM) }
	require.NoError(t, command.Run(), "browser %s failed", child.phase)
}
