//go:build testhelpers

package backuping_physical

import (
	"sync"

	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	databases_bridge "databasus-backend/internal/features/databases"
)

// SubscribeToDatabaseDeletingWithoutApp subscribes the physical handlers to the deletions of the module
// the bridge builds without the application, in the order internal/app uses for the module it builds.
var SubscribeToDatabaseDeletingWithoutApp = sync.OnceFunc(func() {
	databasesModule := databases_bridge.GetModule()
	databasesModule.SubscribeToDatabaseDeleting(getPhysicalBackupCancellationListener().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(physical_service.GetPhysicalBackupService().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(getPhysicalSlotCleanupListener().OnDatabaseDeleting)
})
