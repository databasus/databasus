//go:build testhelpers

package backups_config_logical

import (
	"sync"

	storages_bridge "databasus-backend/internal/features/storages"
)

// SetupDependencies subscribes the backup config service to the storage transfers of the module the
// bridge builds without the application, as internal/app does for the module it builds.
var SetupDependencies = sync.OnceFunc(func() {
	storages_bridge.GetModule().SubscribeToStorageTransferring(getBackupConfigService().OnStorageTransferring)
})
