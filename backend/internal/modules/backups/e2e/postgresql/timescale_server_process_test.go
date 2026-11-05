package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/lib/testwait"
)

const timescaleServerConfigArgument = "--timescale-server-config="

type timescaleServerConfig struct {
	Config  config.Config
	Address string
}

func TestTimescaleShutdownServer(t *testing.T) {
	configPath := ""
	for _, argument := range os.Args {
		if path, isConfigArgument := strings.CutPrefix(argument, timescaleServerConfigArgument); isConfigArgument {
			configPath = path
		}
	}
	if configPath == "" {
		return
	}

	encodedConfig, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var serverConfig timescaleServerConfig
	require.NoError(t, json.Unmarshal(encodedConfig, &serverConfig))

	environment, err := app.NewEnvironment(t.Context(), serverConfig.Config, clock.NewSystem())
	require.NoError(t, err)
	application, err := app.New(t.Context(), environment)
	require.NoError(t, err)

	stopCtx, stopListening := shutdown.ListenForStopSignal(t.Context())
	defer stopListening()
	require.NoError(t, application.Serve(stopCtx, serverConfig.Address))
}

type timescaleServerProcess struct {
	command    *exec.Cmd
	URL        string
	outputPath string
	done       chan struct{}
	err        error
}

func startTimescaleServer(t *testing.T, serverConfig config.Config) *timescaleServerProcess {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	encodedConfig, err := json.Marshal(timescaleServerConfig{Config: serverConfig, Address: address})
	require.NoError(t, err)
	configPath := filepath.Join(t.TempDir(), "server.json")
	require.NoError(t, os.WriteFile(configPath, encodedConfig, 0o600))

	executable, err := os.Executable()
	require.NoError(t, err)
	command := exec.CommandContext(
		t.Context(),
		executable,
		"-test.run=^TestTimescaleShutdownServer$",
		"-test.timeout=2m",
		"--",
		timescaleServerConfigArgument+configPath,
	)
	command.Env = append(command.Environ(), "GORACE=atexit_sleep_ms=0")

	outputPath := filepath.Join(t.TempDir(), "server.log")
	output, err := os.Create(outputPath)
	require.NoError(t, err)
	command.Stdout = output
	command.Stderr = output

	process := &timescaleServerProcess{
		command:    command,
		URL:        "http://" + address,
		outputPath: outputPath,
		done:       make(chan struct{}),
		err:        nil,
	}

	require.NoError(t, command.Start())
	go func() { process.err = command.Wait(); _ = output.Close(); close(process.done) }()

	t.Cleanup(func() {
		_ = command.Process.Kill()
		<-process.done
		if t.Failed() {
			log, _ := os.ReadFile(outputPath)
			t.Log(string(log))
		}
	})

	testwait.Until(t, "the server process accepts requests", func(ctx context.Context) error {
		select {
		case <-process.done:
			log, _ := os.ReadFile(outputPath)
			t.Fatalf("server exited before accepting requests: %v\n%s", process.err, log)
		default:
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, process.URL+"/api/v1/system/version", nil)
		if err != nil {
			return err
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("server answered %d", response.StatusCode)
		}
		return nil
	})

	return process
}

func (p *timescaleServerProcess) Stop(t *testing.T) {
	t.Helper()
	stoppedAt := time.Now()
	require.NoError(t, p.command.Process.Signal(syscall.SIGTERM))
	select {
	case <-p.done:
		require.NoError(t, p.err)
		assert.Less(t, time.Since(stoppedAt), 8*time.Second)
	case <-time.After(8 * time.Second):
		t.Fatal("the server did not exit within eight seconds after SIGTERM")
	}
}
