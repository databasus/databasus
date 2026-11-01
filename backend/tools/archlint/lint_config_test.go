package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintConfiguration(t *testing.T) {
	t.Parallel()
	moduleDefinition, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)
	moduleChecksums, err := os.ReadFile("../../go.sum")
	require.NoError(t, err)
	scenarios := []struct {
		name       string
		filePath   string
		source     string
		diagnostic string
		message    string
	}{
		{
			name:       "rejects a global application dependency without a suppression",
			filePath:   "client.go",
			source:     "package fixture\nimport \"net/http\"\nvar client = &http.Client{}\nfunc GetClient() *http.Client { return client }\n",
			diagnostic: "gochecknoglobals",
		},
		{
			name:       "rejects a global suppression without an explanation",
			filePath:   "table.go",
			source:     "package fixture\n//nolint:gochecknoglobals\nvar names = map[string]string{\"first\": \"first\"}\nfunc GetName() string { return names[\"first\"] }\n",
			diagnostic: "nolintlint",
		},
		{
			name:     "accepts an explained immutable lookup table",
			filePath: "table.go",
			source:   "package fixture\n//nolint:gochecknoglobals // Shared immutable names avoid rebuilding the lookup on each call; no caller mutates it.\nvar names = map[string]string{\"first\": \"first\"}\nfunc GetName() string { return names[\"first\"] }\n",
		},
		{
			name:       "checks another command for forbidden environment reads",
			filePath:   "cmd/other/main.go",
			source:     "package main\nimport \"os\"\nfunc main() { _ = os.Getenv(\"SETTING\") }\n",
			diagnostic: "forbidigo",
		},
		{
			name:     "rejects a fixed sleep when a module test waits",
			filePath: "internal/modules/example/sleep_test.go",
			source: `package example
import ("testing"; "time")
func TestSleep(t *testing.T) {
    t.Run("waits with a fixed pause", func(_ *testing.T) { time.Sleep(time.Millisecond) })
}
`,
			diagnostic: "forbidigo",
			message:    "wait for a condition with testwait.Until or move the clock from Deps",
		},
		{
			name:     "rejects a formatted message when logging a value",
			filePath: "internal/modules/example/service.go",
			source: `package example
import ("context"; "fmt"; "log/slog")
func LogBackup(ctx context.Context, logger *slog.Logger, name string) {
    logger.InfoContext(ctx, fmt.Sprintf("backup %s", name))
}
`,
			diagnostic: "sloglint",
		},
		{
			name:     "names the omitted field when a dependency literal is incomplete",
			filePath: "internal/modules/example/service.go",
			source: `package example
type Deps struct { DatabaseURL string; IsEnabled bool }
func GetDeps() Deps { return Deps{DatabaseURL: "database"} }
`,
			diagnostic: "exhaustruct",
			message:    "IsEnabled",
		},
		{
			name:     "rejects direct process execution when an engine starts a tool",
			filePath: "internal/engines/example/source.go",
			source: `package example
import ("context"; "os/exec")
func RunTool(ctx context.Context) error { return exec.CommandContext(ctx, "pg_dump").Run() }
`,
			diagnostic: "forbidigo",
			message:    "start external tools through lib/process",
		},
		{
			name:     "accepts direct process execution when a test starts its binary",
			filePath: "internal/engines/example/source_test.go",
			source: `package example
import ("os/exec"; "testing")
func TestServerBinary(t *testing.T) {
    if err := exec.CommandContext(t.Context(), "server").Run(); err != nil { t.Fatal(err) }
}
`,
		},
		{
			name:     "rejects an ORM import when code belongs to an engine",
			filePath: "internal/engines/example/source.go",
			source: `package example
import "gorm.io/gorm"
func GetDatabase() *gorm.DB { return nil }
`,
			diagnostic: "depguard",
			message:    "engine drivers know no tables",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			configPath, err := filepath.Abs("../../.golangci.yml")
			require.NoError(t, err)

			require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), moduleDefinition, 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(root, "go.sum"), moduleChecksums, 0o600))
			sourcePath := filepath.Join(root, scenario.filePath)
			require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))
			require.NoError(t, os.WriteFile(sourcePath, []byte(scenario.source), 0o600))

			command := exec.CommandContext(
				t.Context(),
				"golangci-lint",
				"run",
				"--allow-parallel-runners",
				"--config",
				configPath,
				"./...",
			)
			command.Dir = root
			command.Env = append(command.Environ(), "GOFLAGS=")

			formatter := exec.CommandContext(t.Context(), "golangci-lint", "fmt", "--config", configPath, "./...")
			formatter.Dir = root
			formatter.Env = command.Env
			formattedOutput, formatError := formatter.CombinedOutput()
			require.NoError(t, formatError, string(formattedOutput))

			output, lintError := command.CombinedOutput()
			if scenario.diagnostic == "" {
				require.NoError(t, lintError, string(output))
			} else {
				require.Error(t, lintError, string(output))
				assert.Contains(t, string(output), scenario.diagnostic)
				if scenario.message != "" {
					assert.Contains(t, string(output), scenario.message)
				}
			}
		})
	}
}
