package pg17_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestRestorePhysicalBackupChain(t *testing.T) {
	t.Parallel()

	t.Run(
		"recovers rows written before the full backup with only a full backup",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunFullOnly(t, "postgres:17")
		},
	)
	t.Run(
		"recovers rows of every backup with a full backup and two incrementals",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunFullAndIncrementals(t, "postgres:17")
		},
	)
}

func TestRestorePhysicalBackupToTargetTime(t *testing.T) {
	t.Parallel()

	t.Run(
		"recovers rows up to the target and drops later ones with streamed wal",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunTargetTime(t, "postgres:17")
		},
	)
	t.Run(
		"returns 409 target_time_unreachable for the restore token when a wal gap lies before the target",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunWalGap(t, "postgres:17")
		},
	)
	t.Run(
		"keeps a target past the full backup restorable after the orphan sweep",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunOrphanSweep(t, "postgres:17")
		},
	)
}

func TestRestorePhysicalBackupOnHost(t *testing.T) {
	t.Parallel()

	t.Run(
		"serves the restored rows through the image entrypoint after a host restore",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunHostRestore(t, "postgres:17")
		},
	)
	t.Run(
		"replays to the target through the image entrypoint after a host restore",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunHostTargetTime(t, "postgres:17")
		},
	)
	t.Run(
		"refuses to restore when the output directory holds a postgresql directory",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunMisaimedDirectory(t, "postgres:17")
		},
	)
}

func TestPhysicalBackupAfterPromotion(t *testing.T) {
	t.Parallel()

	t.Run(
		"catalogs the new timeline history on the database after the standby is promoted",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunTimelineHistory(t, "postgres:17")
		},
	)
	t.Run(
		"starts a full backup on the new timeline after the standby is promoted",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunPromotionReanchorsChain(t, "postgres:17")
		},
	)
}

func TestPhysicalBackupReplicationSlot(t *testing.T) {
	t.Parallel()

	t.Run(
		"creates the slot when wal streaming starts and drops it when the database is deleted",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunSlotLifecycle(t, "postgres:17")
		},
	)
	t.Run(
		"drops the slot when a database that streamed wal is deleted",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunDeleteStreamingDatabase(t, "postgres:17")
		},
	)
}

func TestPhysicalBackupOverSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run(
		"recovers rows up to the target with a chain built through the bastion",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunTunnelRestore(t, "postgres:17")
		},
	)
	t.Run(
		"keeps the slot and resumes streaming after the bastion dies mid-stream",
		func(t *testing.T) {
			t.Parallel()
			physicaltest.RunBastionOutage(t, "postgres:17")
		},
	)
}
