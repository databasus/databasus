//go:build testhelpers

package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testdb"
)

const childProcessVariable = "DATABASUS_CONFIG_TEST_CHILD"

// The window a blocked process gets to show that it does not continue. It starts only after the
// process reported that it reached the configuration load.
const blockedLoadWindow = time.Second

func TestLoadWithoutApp(t *testing.T) {
	t.Run("gives two test processes started at once different databases", func(t *testing.T) {
		lockDirectory := t.TempDir()
		firstProcess := startLoadingProcess(t, newTestChildCommand(t, lockDirectory))
		secondProcess := startLoadingProcess(t, newTestChildCommand(t, lockDirectory))

		firstDatabase := firstProcess.waitForLoadedDatabase(t)
		secondDatabase := secondProcess.waitForLoadedDatabase(t)

		assert.NotEqual(t, firstDatabase, secondDatabase)
	})

	t.Run("waits in a test binary while another process holds the fixtures lock exclusively", func(t *testing.T) {
		lockDirectory := t.TempDir()
		exclusiveLock := lockFixturesExclusively(t, lockDirectory)
		loadingProcess := startLoadingProcess(t, newTestChildCommand(t, lockDirectory))
		loadingProcess.waitForLine(t, "loading")

		assert.False(t, loadingProcess.hasLineWithin("loaded", blockedLoadWindow), "loaded while the lock was held")

		require.NoError(t, exclusiveLock.Close())
		loadingProcess.waitForLoadedDatabase(t)
	})

	t.Run("does not take the fixtures lock outside a test binary", func(t *testing.T) {
		lockDirectory := t.TempDir()
		lockFixturesExclusively(t, lockDirectory)
		probePath := filepath.Join(t.TempDir(), "load_config")
		buildOutput, err := exec.CommandContext(t.Context(), "go", "build", "-tags", "testhelpers", "-o", probePath,
			"./testdata/load_config").CombinedOutput()
		require.NoError(t, err, string(buildOutput))
		probeCommand := exec.CommandContext(t.Context(), probePath)
		probeCommand.Env = append(os.Environ(), "TMPDIR="+lockDirectory)

		loadingProcess := startLoadingProcess(t, probeCommand)

		loadingProcess.waitForLine(t, "loaded")
	})
}

// TestLoadWithoutAppChild is the test binary the scenarios above start as a separate process.
func TestLoadWithoutAppChild(t *testing.T) {
	if os.Getenv(childProcessVariable) == "" {
		t.Skip("runs only as a child process of TestLoadWithoutApp")
	}

	fmt.Println("loading")
	fmt.Println("loaded " + testdb.GetDatabaseName(GetEnv().DatabaseDsn))
}

func newTestChildCommand(t *testing.T, lockDirectory string) *exec.Cmd {
	t.Helper()

	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLoadWithoutAppChild$", "-test.count=1")
	command.Env = append(os.Environ(), childProcessVariable+"=1", "TMPDIR="+lockDirectory)

	return command
}

func lockFixturesExclusively(t *testing.T, lockDirectory string) *os.File {
	t.Helper()

	lockFile, err := os.OpenFile(filepath.Join(lockDirectory, fixturesLockFileName), os.O_RDONLY|os.O_CREATE, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lockFile.Close() })
	require.NoError(t, syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX))

	return lockFile
}

type loadingProcess struct {
	lines <-chan string
}

func startLoadingProcess(t *testing.T, command *exec.Cmd) *loadingProcess {
	t.Helper()

	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	command.Stderr = io.Discard
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})

	lines := make(chan string, 16)
	go func() {
		defer close(lines)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	return &loadingProcess{lines: lines}
}

func (p *loadingProcess) waitForLoadedDatabase(t *testing.T) string {
	t.Helper()

	databaseName := strings.TrimSpace(strings.TrimPrefix(p.waitForLine(t, "loaded"), "loaded"))
	require.NotEmpty(t, databaseName, "the process loaded no database")
	// The database of a child process lives as long as that process, so the test drops it itself
	// instead of leaving it to the abandoned-database cleanup.
	t.Cleanup(func() { testdb.Drop(t, databaseName) })

	return databaseName
}

func (p *loadingProcess) waitForLine(t *testing.T, prefix string) string {
	t.Helper()

	deadline, isDeadlineSet := t.Deadline()
	if !isDeadlineSet {
		deadline = time.Now().Add(time.Minute)
	}

	waitContext, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()

	for {
		select {
		case line, isOpen := <-p.lines:
			require.True(t, isOpen, "the process exited before printing %q", prefix)

			if strings.HasPrefix(line, prefix) {
				return line
			}
		case <-waitContext.Done():
			require.Failf(t, "the process did not print the line", "prefix: %q", prefix)
		}
	}
}

func (p *loadingProcess) hasLineWithin(prefix string, window time.Duration) bool {
	windowEnd := time.After(window)

	for {
		select {
		case line, isOpen := <-p.lines:
			if !isOpen {
				return false
			}

			if strings.HasPrefix(line, prefix) {
				return true
			}
		case <-windowEnd:
			return false
		}
	}
}
