package pg18_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestRestorePhysicalSchemaAndSequences(t *testing.T) {
	t.Parallel()

	t.Run("recovers definitions with only a full backup and subsequent wal", func(t *testing.T) {
		t.Parallel()

		physicaltest.RunSchemaRecovery(t, physicaltest.SchemaRestoreSpec{
			Image: "postgres:18", Stage: physicaltest.SchemaLatest, HasIncrementals: false,
		})
	})

	t.Run("recovers definitions with two incrementals and subsequent wal", func(t *testing.T) {
		t.Parallel()

		physicaltest.RunSchemaRecovery(t, physicaltest.SchemaRestoreSpec{
			Image: "postgres:18", Stage: physicaltest.SchemaLatest, HasIncrementals: true,
		})
	})
}

func TestRestorePhysicalSchemaAndSequencesToTargetTime(t *testing.T) {
	t.Parallel()

	t.Run("excludes schema changes when their transaction commits after the target", func(t *testing.T) {
		t.Parallel()

		physicaltest.RunSchemaRecovery(t, physicaltest.SchemaRestoreSpec{
			Image: "postgres:18", Stage: physicaltest.SchemaBeforeWal, HasIncrementals: true,
		})
	})

	t.Run("recovers committed definitions without later changes after the target", func(t *testing.T) {
		t.Parallel()

		physicaltest.RunSchemaRecovery(t, physicaltest.SchemaRestoreSpec{
			Image: "postgres:18", Stage: physicaltest.SchemaAfterWal, HasIncrementals: true,
		})
	})
}
