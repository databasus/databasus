package mysqlfamily

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/mysqlfamily/mysqlfamilytest"
)

func TestCompressionProbe(t *testing.T) {
	t.Parallel()

	t.Run("keeps a dash-led database name out of the options with every shipped dump tool", func(t *testing.T) {
		t.Parallel()

		for _, dumpTool := range getShippedDumpTools(t) {
			t.Run(dumpTool.name, func(t *testing.T) {
				t.Parallel()
				resultPath := filepath.Join(t.TempDir(), "result.sql")
				configFile, err := writeClientConfigFile(ClientConnection{
					Host:      "127.0.0.1",
					Port:      mysqlfamilytest.GetClosedPort(t),
					Username:  "backup_user",
					Password:  "stub-secret-password",
					SSLOption: dumpTool.sslOption,
				})
				require.NoError(t, err)
				t.Cleanup(configFile.remove)
				probe := compressionProbe{
					product:      dumpTool.name,
					dumpToolPath: dumpTool.path,
					configFile:   configFile,
					databaseName: "--result-file=" + resultPath,
					candidates:   NetworkCompressionCandidates{Compressed: nil, ProbeArgs: nil},
					logger:       slog.New(slog.DiscardHandler),
				}

				toolOutput := mysqlfamilytest.RunClientTool(t, dumpTool.path, buildCompressionProbeArgs(probe, nil))

				assert.NoFileExists(t, resultPath)
				assert.True(t, mysqlfamilytest.IsConnectionFailure(toolOutput), toolOutput)
			})
		}
	})
}

type shippedDumpTool struct {
	name      string
	path      string
	sslOption string
}

// getShippedDumpTools finds the dump tools in the bundle directly, because the engine packages that
// name the versions import this one.
func getShippedDumpTools(t *testing.T) []shippedDumpTool {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)

	bundleRoot, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	mysqldumpPaths, err := filepath.Glob(filepath.Join(bundleRoot, "mysql", "mysql-*", "bin", "mysqldump"))
	require.NoError(t, err)

	mariadbDumpPaths, err := filepath.Glob(filepath.Join(bundleRoot, "mariadb", "mariadb-*", "bin", "mariadb-dump"))
	require.NoError(t, err)

	require.NotEmpty(t, mysqldumpPaths)
	require.NotEmpty(t, mariadbDumpPaths)

	shippedDumpTools := make([]shippedDumpTool, 0, len(mysqldumpPaths)+len(mariadbDumpPaths))
	for _, mysqldumpPath := range mysqldumpPaths {
		shippedDumpTools = append(shippedDumpTools, shippedDumpTool{
			name:      filepath.Base(filepath.Dir(filepath.Dir(mysqldumpPath))),
			path:      mysqldumpPath,
			sslOption: "ssl-mode=DISABLED",
		})
	}

	for _, mariadbDumpPath := range mariadbDumpPaths {
		shippedDumpTools = append(shippedDumpTools, shippedDumpTool{
			name:      filepath.Base(filepath.Dir(filepath.Dir(mariadbDumpPath))),
			path:      mariadbDumpPath,
			sslOption: "ssl=false",
		})
	}

	return shippedDumpTools
}
