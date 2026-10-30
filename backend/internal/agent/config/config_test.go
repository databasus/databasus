package config

import (
	"bufio"
	"flag"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadConfig(t *testing.T, args ...string) *Config {
	t.Helper()

	cfg := &Config{}
	require.NoError(
		t,
		cfg.LoadFromJSONAndArgs(
			t.Context(),
			slog.New(slog.DiscardHandler),
			flag.NewFlagSet("test", flag.ContinueOnError),
			args,
		),
	)

	return cfg
}

func getAllFlagNames() []string {
	return []string{
		"databasus-host", "agent-id", "token", "max-cpu", "max-ram-mb",
		"max-disk-gb", "max-concurrent-jobs", "allow-insecure-http", "verification-pg-image-repo",
	}
}

func emptyReader() *bufio.Reader {
	return bufio.NewReader(strings.NewReader(""))
}

func TestValidate(t *testing.T) {
	t.Run("returns capacity when the config is complete", func(t *testing.T) {
		cfg := &Config{
			DatabasusHost:     "https://primary.example:4005",
			AgentID:           "agent-1",
			Token:             "secret",
			MaxCPU:            8,
			MaxRAMMb:          4096,
			MaxDiskGb:         100,
			MaxConcurrentJobs: 4,
		}

		capacity, err := cfg.Validate()

		require.NoError(t, err)
		assert.Equal(t, 2, capacity.CPUPerJob)
		assert.Equal(t, 1024, capacity.RAMMbPerJob)
	})
	t.Run("returns error when a required string is missing", func(t *testing.T) {
		for name, cfg := range map[string]*Config{
			"no databasus-host": {AgentID: "a", Token: "t", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
			"no agent-id":       {DatabasusHost: "https://x:4005", Token: "t", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
			"no token":          {DatabasusHost: "https://x:4005", AgentID: "a", MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := cfg.Validate()
				require.Error(t, err)
			})
		}
	})
	t.Run("returns capacity error when the configured capacity is invalid", func(t *testing.T) {
		cfg := &Config{
			DatabasusHost:     "https://primary.example:4005",
			AgentID:           "agent-1",
			Token:             "secret",
			MaxCPU:            2,
			MaxRAMMb:          4096,
			MaxDiskGb:         100,
			MaxConcurrentJobs: 4,
		}

		_, err := cfg.Validate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CPU per job")
	})
}

func TestDeriveCapacity(t *testing.T) {
	t.Run("derives per job split when the config is valid", func(t *testing.T) {
		cfg := &Config{MaxCPU: 8, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4}

		capacity, err := cfg.DeriveCapacity()

		require.NoError(t, err)
		assert.Equal(t, 2, capacity.CPUPerJob)
		assert.Equal(t, 1024, capacity.RAMMbPerJob)
		assert.Equal(t, 8, capacity.MaxCPU)
		assert.Equal(t, 4096, capacity.MaxRAMMb)
		assert.Equal(t, 100, capacity.MaxDiskGb)
		assert.Equal(t, 4, capacity.MaxConcurrentJobs)
	})
	t.Run("returns error when concurrent jobs exceed available CPUs", func(t *testing.T) {
		cfg := &Config{MaxCPU: 2, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4}

		_, err := cfg.DeriveCapacity()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CPU per job")
	})
	t.Run("returns error when RAM is below the minimum", func(t *testing.T) {
		cfg := &Config{MaxCPU: 4, MaxRAMMb: 256, MaxDiskGb: 100, MaxConcurrentJobs: 1}

		_, err := cfg.DeriveCapacity()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "max-ram-mb")
	})
	t.Run("returns error when RAM per job is below the minimum", func(t *testing.T) {
		cfg := &Config{MaxCPU: 8, MaxRAMMb: 1024, MaxDiskGb: 100, MaxConcurrentJobs: 8}

		_, err := cfg.DeriveCapacity()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "per job")
	})
	t.Run("returns error when a capacity field is zero", func(t *testing.T) {
		for name, cfg := range map[string]*Config{
			"no concurrent jobs": {MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 0},
			"no cpu":             {MaxCPU: 0, MaxRAMMb: 2048, MaxDiskGb: 10, MaxConcurrentJobs: 1},
			"no disk":            {MaxCPU: 4, MaxRAMMb: 2048, MaxDiskGb: 0, MaxConcurrentJobs: 1},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := cfg.DeriveCapacity()
				require.Error(t, err)
			})
		}
	})
}

func TestValidateTransport(t *testing.T) {
	t.Run("passes without prompt when the host uses HTTPS", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "https://primary.example:4005"}

		err := cfg.ValidateTransport(t.Context(), slog.New(slog.DiscardHandler), false, emptyReader())

		require.NoError(t, err)
	})
	t.Run("fails naming both fixes when HTTP has no interactive terminal or consent flag", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "http://primary.example:4005"}

		err := cfg.ValidateTransport(t.Context(), slog.New(slog.DiscardHandler), false, emptyReader())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "https://")
		assert.Contains(t, err.Error(), "--allow-insecure-http")
	})
	t.Run("passes with warn when HTTP is explicitly allowed", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "http://primary.example:4005", AllowInsecureHTTP: true}

		err := cfg.ValidateTransport(t.Context(), slog.New(slog.DiscardHandler), false, emptyReader())

		require.NoError(t, err)
	})
	t.Run("passes when the operator consents to HTTP", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "http://primary.example:4005"}

		err := cfg.ValidateTransport(
			t.Context(),
			slog.New(slog.DiscardHandler),
			true,
			bufio.NewReader(strings.NewReader("y\n")),
		)

		require.NoError(t, err)
	})
	t.Run("returns error when the operator declines HTTP", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "http://primary.example:4005"}

		err := cfg.ValidateTransport(
			t.Context(),
			slog.New(slog.DiscardHandler),
			true,
			bufio.NewReader(strings.NewReader("n\n")),
		)

		require.Error(t, err)
	})
	t.Run("returns error when the URL scheme is unsupported", func(t *testing.T) {
		cfg := &Config{DatabasusHost: "ftp://primary.example:4005"}

		err := cfg.ValidateTransport(t.Context(), slog.New(slog.DiscardHandler), true, emptyReader())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "https://")
	})
}

func TestMaskSensitive(t *testing.T) {
	t.Run("returns not set when the input is empty", func(t *testing.T) {
		assert.Equal(t, "(not set)", maskSensitive(""))
	})
	t.Run("reveals quarter then masks when a value is provided", func(t *testing.T) {
		assert.Equal(t, "ab***", maskSensitive("abcdefgh"))
	})
}

func TestGetVerificationPgImageRepo(t *testing.T) {
	t.Run("returns bundled default when no image repository is set", func(t *testing.T) {
		cfg := &Config{}

		assert.Equal(t, DefaultVerificationPgImageRepo, cfg.GetVerificationPgImageRepo())
	})
	t.Run("returns configured value when an image repository is set", func(t *testing.T) {
		cfg := &Config{VerificationPgImageRepo: "registry.internal/pg-verify"}

		assert.Equal(t, "registry.internal/pg-verify", cfg.GetVerificationPgImageRepo())
	})
}

func TestLoadFromJSONAndArgs(t *testing.T) {
	t.Run("overrides every value when flags provide values", func(t *testing.T) {
		t.Chdir(t.TempDir())

		cfg := loadConfig(t,
			"--databasus-host", "https://flag:4005",
			"--agent-id", "agent-x",
			"--token", "tok",
			"--max-cpu", "8",
			"--max-ram-mb", "4096",
			"--max-disk-gb", "100",
			"--max-concurrent-jobs", "4",
			"--allow-insecure-http",
			"--verification-pg-image-repo", "registry/pg",
		)

		assert.Equal(t, "https://flag:4005", cfg.DatabasusHost)
		assert.Equal(t, "agent-x", cfg.AgentID)
		assert.Equal(t, "tok", cfg.Token)
		assert.Equal(t, 8, cfg.MaxCPU)
		assert.Equal(t, 4096, cfg.MaxRAMMb)
		assert.Equal(t, 100, cfg.MaxDiskGb)
		assert.Equal(t, 4, cfg.MaxConcurrentJobs)
		assert.True(t, cfg.AllowInsecureHTTP)
		assert.Equal(t, "registry/pg", cfg.VerificationPgImageRepo)

		for _, name := range getAllFlagNames() {
			assert.Equal(t, "command line args", cfg.flagSources[name], name)
		}
	})
	t.Run("loads file and marks source when only the configuration file provides values", func(t *testing.T) {
		t.Chdir(t.TempDir())
		source := &Config{DatabasusHost: "https://file:4005", AgentID: "agent-file", MaxCPU: 6}
		require.NoError(t, source.SaveToJSON())

		cfg := loadConfig(t)

		assert.Equal(t, "https://file:4005", cfg.DatabasusHost)
		assert.Equal(t, "agent-file", cfg.AgentID)
		assert.Equal(t, 6, cfg.MaxCPU)
		assert.Equal(t, configFileName, cfg.flagSources["databasus-host"])
	})
	t.Run("flag overrides file when flags and the configuration file provide values", func(t *testing.T) {
		t.Chdir(t.TempDir())
		source := &Config{DatabasusHost: "https://file:4005", AgentID: "agent-file"}
		require.NoError(t, source.SaveToJSON())

		cfg := loadConfig(t, "--databasus-host", "https://flag:4005")

		assert.Equal(t, "https://flag:4005", cfg.DatabasusHost, "flag overrides file")
		assert.Equal(t, "agent-file", cfg.AgentID, "untouched field keeps the file value")
		assert.Equal(t, "command line args", cfg.flagSources["databasus-host"])
		assert.Equal(t, configFileName, cfg.flagSources["agent-id"])
	})
	t.Run("does not override file when flag values are zero", func(t *testing.T) {
		t.Chdir(t.TempDir())
		source := &Config{MaxCPU: 8, VerificationPgImageRepo: "registry/pg"}
		require.NoError(t, source.SaveToJSON())

		cfg := loadConfig(t, "--max-cpu", "0", "--verification-pg-image-repo", "")

		assert.Equal(t, 8, cfg.MaxCPU, "an explicit zero flag cannot unset a file value")
		assert.Equal(t, "registry/pg", cfg.VerificationPgImageRepo)
	})
	t.Run("stays true without flag when the file allows HTTP", func(t *testing.T) {
		t.Chdir(t.TempDir())
		source := &Config{AllowInsecureHTTP: true}
		require.NoError(t, source.SaveToJSON())

		cfg := loadConfig(t)

		assert.True(t, cfg.AllowInsecureHTTP)
		assert.Equal(t, configFileName, cfg.flagSources["allow-insecure-http"])
	})
	t.Run("marks all sources as not configured when no values are provided", func(t *testing.T) {
		t.Chdir(t.TempDir())

		cfg := loadConfig(t)

		for _, name := range getAllFlagNames() {
			assert.Equal(t, "not configured", cfg.flagSources[name], name)
		}
	})
}
