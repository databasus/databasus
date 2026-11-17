// Package mysqlfamilytest runs a real bundled MySQL or MariaDB client tool against a port where
// nothing listens, so a test sees how the tool reads its arguments without a database server.
package mysqlfamilytest

import (
	"net"
	"strings"
	"testing"

	"databasus-backend/internal/lib/process"
)

// GetClosedPort returns a local port that refuses connections: it is taken from the kernel and
// released at once.
func GetClosedPort(t *testing.T) int {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on a local port: %v", err)
	}

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	if !isTCPAddress {
		t.Fatalf("the listener has no TCP address: %v", listener.Addr())
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("release the local port: %v", err)
	}

	return listenerAddress.Port
}

// RunClientTool runs the tool to its exit and returns what it wrote. The tool is expected to fail,
// so its exit status is not checked.
func RunClientTool(t *testing.T, toolPath string, args []string) string {
	t.Helper()

	toolOutput, _ := process.CombinedOutput(t.Context(), process.Spec{
		Path:       toolPath,
		Args:       args,
		UnsetEnv:   nil,
		Env:        nil,
		Dir:        "",
		Stdin:      strings.NewReader(""),
		Stdout:     nil,
		Stderr:     nil,
		StopSignal: nil,
		KillAfter:  0,
	})

	return string(toolOutput)
}

// IsConnectionFailure reports whether the tool got as far as dialing the server, which it does only
// after it accepted every argument.
func IsConnectionFailure(toolOutput string) bool {
	return strings.Contains(toolOutput, "Can't connect")
}
