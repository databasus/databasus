package pg18_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestRestorePhysicalBackupThroughPgBouncer(t *testing.T) {
	t.Parallel()
	t.Run("recovers the full incremental and wal chain to the target through pgbouncer", func(t *testing.T) {
		t.Parallel()
		physicaltest.RunPgBouncerTargetTime(t, "postgres:18")
	})
}
