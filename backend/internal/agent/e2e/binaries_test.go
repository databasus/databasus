package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

//nolint:gochecknoglobals // TestMain builds the binaries and sets their paths once before any scenario runs.
var (
	currentBinaryDirectory string
	olderBinaryDirectory   string
)

const olderVersion = "0.0.1"

// The agent starts its background upgrader for every version except "dev", as a release does.
const serverVersion = "0.0.2-e2e"

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "databasus-agent-e2e-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()

	//nolint:forbidigo // TestMain has no testing.T context; both builds share a bounded lifetime.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := os.Setenv("APP_VERSION", serverVersion); err != nil {
		panic(err)
	}

	currentBinaryDirectory = filepath.Join(directory, "current")
	olderBinaryDirectory = filepath.Join(directory, "older")

	for _, binary := range []struct {
		directory string
		version   string
	}{{currentBinaryDirectory, serverVersion}, {olderBinaryDirectory, olderVersion}} {
		if err := os.MkdirAll(binary.directory, 0o755); err != nil {
			panic(err)
		}

		command := exec.CommandContext(
			ctx,
			"go",
			"build",
			"-ldflags",
			"-X main.Version="+binary.version,
			"-o",
			filepath.Join(binary.directory, binaryName()),
			"./cmd/verification-agent",
		)

		command.Dir = "../../.."
		command.Env = append(command.Environ(), "CGO_ENABLED=0")
		if output, err := command.CombinedOutput(); err != nil {
			panic(fmt.Sprintf("build agent %s: %v\n%s", binary.version, err, output))
		}
	}

	m.Run()
}

func binaryName() string { return "databasus-verification-agent-linux-" + runtime.GOARCH }
