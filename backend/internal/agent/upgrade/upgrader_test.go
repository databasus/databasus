package upgrade

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/api"
)

func goBuildHelper(t *testing.T, mainSrc string) string {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(src, []byte(mainSrc), 0o644))

	bin := filepath.Join(dir, "helper")

	out, err := exec.CommandContext(t.Context(), "go", "build", "-o", bin, src).CombinedOutput()
	require.NoError(t, err, "build helper: %s", out)

	return bin
}

func buildVersionHelper(t *testing.T, version string) string {
	t.Helper()

	return goBuildHelper(t, "package main\n"+
		"import (\"fmt\"; \"os\")\n"+
		"func main(){ if len(os.Args)>1 && os.Args[1]==\"version\" { fmt.Println(\""+
		version+"\"); return }; os.Exit(2) }\n")
}

func buildHangingHelper(t *testing.T) string {
	t.Helper()

	return goBuildHelper(t, "package main\n"+
		"import \"time\"\n"+
		"func main(){ time.Sleep(30 * time.Second) }\n")
}

func TestVerifyBinary(t *testing.T) {
	t.Run("returns nil when the binary reports the requested version", func(t *testing.T) {
		helper := buildVersionHelper(t, "v2.0.0")

		require.NoError(
			t,
			New(
				Deps{APIClient: nil, Logger: slog.New(slog.DiscardHandler)},
			).verifyBinary(t.Context(), helper, "v2.0.0"),
		)
	})
	t.Run("returns error when the binary reports a different version", func(t *testing.T) {
		helper := buildVersionHelper(t, "v2.0.0")

		err := New(
			Deps{APIClient: nil, Logger: slog.New(slog.DiscardHandler)},
		).verifyBinary(t.Context(), helper, "v9.9.9")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "version mismatch")
	})
	t.Run("returns error when the file is not executable", func(t *testing.T) {
		notBinary := filepath.Join(t.TempDir(), "not-a-binary")
		require.NoError(t, os.WriteFile(notBinary, []byte("plain text"), 0o644))

		require.Error(
			t,
			New(
				Deps{APIClient: nil, Logger: slog.New(slog.DiscardHandler)},
			).verifyBinary(t.Context(), notBinary, "v1.0.0"),
		)
	})
	t.Run("times out instead of blocking when the binary hangs", func(t *testing.T) {
		upgrader := New(Deps{APIClient: nil, Logger: slog.New(slog.DiscardHandler)})
		upgrader.verifyBinaryTimeout = 200 * time.Millisecond

		hanging := buildHangingHelper(t)

		start := time.Now()
		err := upgrader.verifyBinary(t.Context(), hanging, "v1.0.0")
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Less(t, elapsed, 10*time.Second, "verifyBinary must abort on timeout, not block")
		assert.NotContains(t, err.Error(), "version mismatch")
	})
}

func TestCheckAndUpdate(t *testing.T) {
	t.Run("does not contact server when development mode is enabled", func(t *testing.T) {
		var hits atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
		}))
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)

		upgraded, err := New(
			Deps{APIClient: client, Logger: slog.New(slog.DiscardHandler)},
		).CheckAndUpdate(t.Context(), "v1.0.0", true)

		require.NoError(t, err)
		assert.False(t, upgraded)
		assert.Equal(t, int32(0), hits.Load())
	})
	t.Run("does not download when the server version matches", func(t *testing.T) {
		var downloadHits atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/system/version" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"version": "v1.0.0"})
				return
			}

			downloadHits.Add(1)
		}))
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)

		upgraded, err := New(
			Deps{APIClient: client, Logger: slog.New(slog.DiscardHandler)},
		).CheckAndUpdate(t.Context(), "v1.0.0", false)

		require.NoError(t, err)
		assert.False(t, upgraded)
		assert.Equal(t, int32(0), downloadHits.Load())
	})
	t.Run(
		"skips upgrade when the version response is stale and the binary matches the current version",
		func(t *testing.T) {
			server := startFakeBackend(t, fakeBackend{
				versionEndpointReports: "v3.43.0",
				downloadHeaderReports:  "v3.45.0",
				servedBinaryPrints:     "v3.45.0",
			})
			client := api.NewClient(
				api.Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
			)

			upgraded, err := New(
				Deps{APIClient: client, Logger: slog.New(slog.DiscardHandler)},
			).CheckAndUpdate(t.Context(), "v3.45.0", false)

			require.NoError(t, err)
			assert.False(t, upgraded, "a stale /system/version must not make the agent re-exec into itself")
		},
	)
	t.Run("returns mismatch error when the binary disagrees with its version header", func(t *testing.T) {
		server := startFakeBackend(t, fakeBackend{
			versionEndpointReports: "v3.46.0",
			downloadHeaderReports:  "v3.46.0",
			servedBinaryPrints:     "v3.44.0",
		})
		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)

		upgraded, err := New(
			Deps{APIClient: client, Logger: slog.New(slog.DiscardHandler)},
		).CheckAndUpdate(t.Context(), "v3.44.0", false)

		require.Error(t, err)
		assert.False(t, upgraded)
		assert.Contains(t, err.Error(), "version mismatch")
	})
	t.Run("returns error when the server is unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		server.Close()

		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)

		upgraded, err := New(
			Deps{APIClient: client, Logger: slog.New(slog.DiscardHandler)},
		).CheckAndUpdate(t.Context(), "v1.0.0", false)

		require.Error(t, err)
		assert.False(t, upgraded)
		assert.Contains(t, err.Error(), "unable to check version")
	})
}

type fakeBackend struct {
	versionEndpointReports string
	downloadHeaderReports  string
	servedBinaryPrints     string
}

// Both callers below stop before CheckAndUpdate's os.Rename, which would
// otherwise overwrite the running test binary.
func startFakeBackend(t *testing.T, backend fakeBackend) *httptest.Server {
	t.Helper()

	binary, err := os.ReadFile(buildVersionHelper(t, backend.servedBinaryPrints))
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/system/version" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"version": backend.versionEndpointReports})

			return
		}

		w.Header().Set("X-Databasus-Version", backend.downloadHeaderReports)
		_, _ = w.Write(binary)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestBackgroundUpgrader(t *testing.T) {
	t.Run("never upgrades when development mode is enabled", func(t *testing.T) {
		client := api.NewClient(
			api.Deps{Host: "http://127.0.0.1:0", Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)
		upgrader := NewBackgroundUpgrader(
			BackgroundDeps{
				APIClient:      client,
				CurrentVersion: "v1.0.0",
				IsDev:          true,
				Cancel:         func() {},
				Logger:         slog.New(slog.DiscardHandler),
			},
		)

		assert.False(t, upgrader.checkAndUpgrade(t.Context()))
		assert.False(t, upgrader.IsUpgraded())
	})
	t.Run("panics when Run is called twice", func(t *testing.T) {
		client := api.NewClient(
			api.Deps{Host: "http://127.0.0.1:0", Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)},
		)
		ctx, cancel := context.WithCancel(t.Context())
		upgrader := NewBackgroundUpgrader(
			BackgroundDeps{
				APIClient:      client,
				CurrentVersion: "v1.0.0",
				IsDev:          true,
				Cancel:         cancel,
				Logger:         slog.New(slog.DiscardHandler),
			},
		)

		cancel()
		upgrader.Run(ctx)

		assert.Panics(t, func() { upgrader.Run(ctx) })
	})
}
