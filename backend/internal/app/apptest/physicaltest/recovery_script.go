package physicaltest

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	containers "databasus-backend/internal/lib/testcontainers"
)

type restoreSpec struct {
	App        *apptest.App
	Image      string
	BundlePath string
	TargetTime *time.Time
}

func fetchRecoveryScript(t *testing.T, testApp *apptest.App) string {
	t.Helper()

	status, body := testApp.CallPhysicalAPI(
		t, apptest.Session{},
		apptest.PhysicalScriptRequest(),
	)
	require.Equal(t, http.StatusOK, status, "%s", body)
	require.NotEmpty(t, body)
	hostPath := filepath.Join(t.TempDir(), "databasus-recovery.sh")
	require.NoError(t, os.WriteFile(hostPath, body, 0o700))
	return hostPath
}

func reconstructCluster(t *testing.T, target containers.RestoreTarget, spec restoreSpec) {
	t.Helper()

	hostScript := fetchRecoveryScript(t, spec.App)

	stagedBundle := "/tmp/restore.tar"
	stagedScript := "/tmp/databasus-recovery.sh"

	target.CopyFileToContainer(t, spec.BundlePath, stagedBundle, 0o644)
	target.CopyFileToContainer(t, hostScript, stagedScript, 0o755)

	if spec.TargetTime != nil {
		formattedTarget := spec.TargetTime.UTC().Format("2006-01-02 15:04:05-07:00")
		target.Exec(t, "sh", stagedScript, "--target-time", formattedTarget, stagedBundle, restoreWorkDir)
	} else {
		target.Exec(t, "sh", stagedScript, stagedBundle, restoreWorkDir)
	}

	target.Exec(t, "chown", "-R", "postgres:postgres", restoreWorkDir)
	target.Exec(t, "chmod", "0700", restoredClusterDir(spec.Image))
}
