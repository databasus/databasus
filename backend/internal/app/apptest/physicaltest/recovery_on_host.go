package physicaltest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/process"
	containers "databasus-backend/internal/lib/testcontainers"
)

type hostRecoverySpec struct {
	ScriptPath      string
	BundlePath      string
	OutputDirectory string
	Image           string
	TargetTime      *time.Time
}

func runRecoveryScriptOnHost(t *testing.T, spec hostRecoverySpec) ([]byte, error) {
	t.Helper()

	args := []string{spec.ScriptPath, "--combine-image", spec.Image}
	if spec.TargetTime != nil {
		args = append(args, "--target-time", spec.TargetTime.UTC().Format("2006-01-02 15:04:05-07:00"))
	}
	args = append(args, spec.BundlePath, spec.OutputDirectory)

	out, err := process.CombinedOutput(t.Context(), process.Spec{Path: "sh", Args: args})

	return out, err
}

func reconstructOnHost(t *testing.T, spec restoreSpec) string {
	t.Helper()

	script := fetchRecoveryScript(t, spec.App)
	outDir := filepath.Join(t.TempDir(), "restored cluster")

	out, err := runRecoveryScriptOnHost(
		t,
		hostRecoverySpec{
			ScriptPath:      script,
			BundlePath:      spec.BundlePath,
			OutputDirectory: outDir,
			Image:           spec.Image,
			TargetTime:      spec.TargetTime,
		},
	)
	require.NoError(t, err, "recovery script must succeed into an empty dir:\n%s", out)

	return outDir
}

func bootBoundCluster(t *testing.T, directory clusterDirectory) containers.Endpoint {
	t.Helper()

	image, outDir := directory.Image, directory.OutputDirectory

	return containers.StartPostgresWithBoundDataDir(
		t,
		image,
		hostVolumeDirForMount(clusterDirectory{Image: image, OutputDirectory: outDir}),
	)
}

func requireRecoveryRefusesReRestore(t *testing.T, spec restoreSpec, outDir string) {
	t.Helper()

	script := fetchRecoveryScript(t, spec.App)

	out, err := runRecoveryScriptOnHost(
		t,
		hostRecoverySpec{ScriptPath: script, BundlePath: spec.BundlePath, OutputDirectory: outDir, Image: spec.Image},
	)
	require.Error(t, err, "re-restoring into an initialized cluster dir must fail; output:\n%s", out)
	require.Contains(t, string(out), "already holds a PostgreSQL cluster",
		"the guard must name the populated-cluster refusal; output:\n%s", out)
}

func requireRecoveryRefusesMisaimedDir(t *testing.T, testApp *apptest.App, image string) {
	t.Helper()

	outDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "pg_hba.conf"), []byte("x\n"), 0o600))

	script := fetchRecoveryScript(t, testApp)
	missingBundle := filepath.Join(t.TempDir(), "missing.tar")

	out, err := runRecoveryScriptOnHost(
		t,
		hostRecoverySpec{ScriptPath: script, BundlePath: missingBundle, OutputDirectory: outDir, Image: image},
	)
	require.Error(t, err, "restoring into a live data/install dir must fail; output:\n%s", out)
	require.Contains(t, string(out), "looks like a PostgreSQL data or install directory",
		"the guard must explain the misaimed-directory refusal; output:\n%s", out)
}
