package physicaltest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func prepareRestoreTarget(t *testing.T, image string) containers.RestoreTarget {
	t.Helper()

	return containers.StartPhysicalRestoreTarget(t, image)
}

func requireRestoredClusterNeedsNoZstd(t *testing.T, target containers.RestoreTarget, image string) {
	t.Helper()

	clusterDir := restoredClusterDir(image)

	autoConf := string(target.Exec(t, "cat", clusterDir+"/postgresql.auto.conf"))
	require.Contains(t, autoConf, "restore_command = 'cp ",
		"restore_command must be a plain cp so the runtime cluster never calls zstd")
	require.NotContains(t, autoConf, "zstd",
		"the recovery config must not reference zstd")

	leftoverCompressed := strings.TrimSpace(string(
		target.Exec(t, "sh", "-c", "ls "+clusterDir+"/databasus_wal_restore/*.zst 2>/dev/null | wc -l"),
	))
	require.Equal(t, "0", leftoverCompressed,
		"the WAL archive the cluster reads must hold no still-compressed segments")

	target.Exec(t, "sh", "-c",
		`p=$(command -v zstd); [ -n "$p" ] && rm -f "$p"; ! command -v zstd >/dev/null`)
}

func startRestoredCluster(t *testing.T, target containers.RestoreTarget, image string) {
	t.Helper()

	clusterDir := restoredClusterDir(image)

	target.Exec(t, "sh", "-c",
		"touch "+restoreWorkDir+"/pg.log && chown postgres:postgres "+restoreWorkDir+"/pg.log")

	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		t.Logf(
			"=== restored pg.log ===\n%s",
			target.ExecBestEffort(context.WithoutCancel(t.Context()), "", "cat", restoreWorkDir+"/pg.log"),
		)
	})

	target.ExecAs(t, "postgres",
		"pg_ctl", "-D", clusterDir, "-l", restoreWorkDir+"/pg.log", "-w", "start")

	t.Cleanup(func() {
		target.ExecBestEffort(context.WithoutCancel(t.Context()), "postgres",
			"pg_ctl", "-D", clusterDir, "-m", "immediate", "stop")
	})

	waitForRecoveryToFinish(t, target)
}

func waitForRecoveryToFinish(t *testing.T, target containers.RestoreTarget) {
	t.Helper()

	testwait.Until(t, "the restored cluster finishes replaying WAL", func(context.Context) error {
		recoveryState := target.ExecBestEffort(context.WithoutCancel(t.Context()),
			"postgres",
			"psql",
			"-U",
			restoredPgUser,
			"-d",
			restoredPgDatabase,
			"-tAc",
			"SELECT pg_is_in_recovery()",
		)
		if strings.TrimSpace(string(recoveryState)) != "f" {
			return fmt.Errorf("recovery state: %s", recoveryState)
		}
		return nil
	})
}
