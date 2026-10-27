//go:build testhelpers

package backups_services

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
)

// SubscribeToDatabaseEventsWithoutApp subscribes the physical backup service to the config events of
// the module the bridge builds without the application, as internal/app does.
var SubscribeToDatabaseEventsWithoutApp = sync.OnceFunc(func() {
	backupsModule := backups_bridge.GetModule()
	backupsModule.SubscribeToPhysicalBackupConfigSaved(getPhysicalBackupService().OnPhysicalBackupConfigSaved)
})
