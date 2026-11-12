package pg17_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestRestorePhysicalBackupCompletion(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name                 string
		shouldUseIncremental bool
	}{
		{name: "recovers the advertised completion point when WAL after a full backup has no transactions"},
		{name: "recovers the advertised completion point when WAL after an incremental backup has no transactions", shouldUseIncremental: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			physicaltest.RunBackupCompletionRecovery(t, physicaltest.BackupCompletionSpec{
				Image: "postgres:17", ShouldUseIncremental: scenario.shouldUseIncremental,
			})
		})
	}
}
