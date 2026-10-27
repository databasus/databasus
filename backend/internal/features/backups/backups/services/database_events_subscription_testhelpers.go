//go:build testhelpers

package backups_services

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
)

// SubscribeToDatabaseEventsWithoutApp subscribes the backup services to the config events and the logical
// handlers to the deletions of the modules the bridges build without the application, as internal/app does.
var SubscribeToDatabaseEventsWithoutApp = sync.OnceFunc(func() {
	backupsModule := backups_bridge.GetModule()
	backupsModule.SubscribeToLogicalBackupConfigStorageChanging(getBackupService().OnLogicalBackupConfigStorageChanging)
	backupsModule.SubscribeToPhysicalBackupConfigSaved(getPhysicalBackupService().OnPhysicalBackupConfigSaved)

	databasesModule := databases_bridge.GetModule()
	databasesModule.SubscribeToDatabaseDeleting(getBackupService().OnDatabaseDeleting)
})
