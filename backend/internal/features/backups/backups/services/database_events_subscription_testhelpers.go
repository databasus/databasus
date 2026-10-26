//go:build testhelpers

package backups_services

import (
	"sync"

	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	databases_bridge "databasus-backend/internal/features/databases"
)

// SubscribeToDatabaseEventsWithoutApp subscribes the logical handlers to the module the bridge builds
// without the application, as internal/app does for the module it builds.
var SubscribeToDatabaseEventsWithoutApp = sync.OnceFunc(func() {
	databasesModule := databases_bridge.GetModule()
	databasesModule.SubscribeToDatabaseCopied(backups_config_logical.GetBackupConfigService().OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseDeleting(getBackupService().OnDatabaseDeleting)
})
