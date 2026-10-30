package pg18_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestRestorePhysicalBackupChain(t *testing.T) {
	t.Parallel()

	t.Run(
		"recovers rows written before the full backup with only a full backup",
		func(t *testing.T) { physicaltest.RunFullOnly(t, "postgres:18") },
	)
	t.Run(
		"recovers rows of every backup with a full backup and two incrementals",
		func(t *testing.T) { physicaltest.RunFullAndIncrementals(t, "postgres:18") },
	)
}

func TestRestorePhysicalBackupToTargetTime(t *testing.T) {
	t.Parallel()

	t.Run(
		"recovers rows up to the target and drops later ones with streamed wal",
		func(t *testing.T) { physicaltest.RunTargetTime(t, "postgres:18") },
	)
	t.Run(
		"returns 409 target_time_unreachable for the restore token when a wal gap lies before the target",
		func(t *testing.T) { physicaltest.RunWalGap(t, "postgres:18") },
	)
	t.Run(
		"keeps a target past the full backup restorable after the orphan sweep",
		func(t *testing.T) { physicaltest.RunOrphanSweep(t, "postgres:18") },
	)
}

func TestRestorePhysicalBackupOnHost(t *testing.T) {
	t.Parallel()

	t.Run(
		"serves the restored rows through the image entrypoint after a host restore",
		func(t *testing.T) { physicaltest.RunHostRestore(t, "postgres:18") },
	)
	t.Run(
		"replays to the target through the image entrypoint after a host restore",
		func(t *testing.T) { physicaltest.RunHostTargetTime(t, "postgres:18") },
	)
	t.Run(
		"refuses to restore when the output directory holds a postgresql directory",
		func(t *testing.T) { physicaltest.RunMisaimedDirectory(t, "postgres:18") },
	)
}

func TestPhysicalBackupAfterPromotion(t *testing.T) {
	t.Parallel()

	t.Run(
		"catalogs the new timeline history on the database after the standby is promoted",
		func(t *testing.T) { physicaltest.RunTimelineHistory(t, "postgres:18") },
	)
	t.Run(
		"starts a full backup on the new timeline after the standby is promoted",
		func(t *testing.T) { physicaltest.RunPromotionReanchorsChain(t, "postgres:18") },
	)
}

func TestPhysicalBackupReplicationSlot(t *testing.T) {
	t.Parallel()

	t.Run(
		"creates the slot when wal streaming starts and drops it when the database is deleted",
		func(t *testing.T) { physicaltest.RunSlotLifecycle(t, "postgres:18") },
	)
	t.Run(
		"drops the slot when a database that streamed wal is deleted",
		func(t *testing.T) { physicaltest.RunDeleteStreamingDatabase(t, "postgres:18") },
	)
}
