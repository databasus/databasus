//go:build testhelpers

package verification_runs

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
)

var SetupTestDependencies = sync.OnceFunc(func() {
	backups_bridge.GetModule().SubscribeToLogicalBackupsDeleting(getVerificationService().OnLogicalBackupsDeleting)
})
