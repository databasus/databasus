package physicaltest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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
			"=== restored server log ===\n%s",
			readRestoredServerLogs(context.WithoutCancel(t.Context()), target, image),
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

func readRestoredServerLogs(ctx context.Context, target containers.RestoreTarget, image string) string {
	return string(target.ExecBestEffort(
		ctx,
		"",
		"sh",
		"-c",
		`cat "$1"; cat "$2"/log/* 2>/dev/null; true`,
		"server-log",
		restoreWorkDir+"/pg.log",
		restoredClusterDir(image),
	))
}

// TCP applies the copied authentication rules, without relying on a local socket rule the source never had.
func waitForRecoveryToFinish(t *testing.T, target containers.RestoreTarget) {
	t.Helper()

	testwait.Until(t, "the restored cluster finishes replaying WAL", func(ctx context.Context) error {
		connection, err := pgx.Connect(ctx, fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
			target.Host(),
			target.MappedPort(),
			restoredPgUser,
			restoredPgPassword,
			restoredPgDatabase,
		))
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

		var isInRecovery bool
		if err := connection.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&isInRecovery); err != nil {
			return err
		}

		if isInRecovery {
			return fmt.Errorf("the restored cluster is still in recovery")
		}

		return nil
	})
}
