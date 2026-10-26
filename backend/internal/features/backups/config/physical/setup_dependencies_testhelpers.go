//go:build testhelpers

package backups_config_physical

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
)

// SetupDependencies subscribes the physical backup configs to the backup type changes of the module
// the bridge builds without the application, as internal/app does for the module it builds.
var SetupDependencies = sync.OnceFunc(func() {
	databases_bridge.GetModule().SubscribeToDatabaseBackupTypeChanged(getBackupConfigService().OnBackupTypeChanged)
})
