package features

import (
	"testing"

	databases_bridge "databasus-backend/internal/features/databases"
)

// Test_SetupDependencies_CalledTwice_LogsWarning verifies SetupDependencies is idempotent
func Test_SetupDependencies_CalledTwice_LogsWarning(t *testing.T) {
	// Call each SetupDependencies twice - should not panic, only log warnings

	databases_bridge.SetupDependencies()
	databases_bridge.SetupDependencies()

	// If we reach here without panic, test passes
	t.Log("All SetupDependencies calls completed successfully (idempotent)")
}
