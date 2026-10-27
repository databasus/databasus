//go:build testhelpers

package backups_bridge

import (
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

// loadWithoutApp subscribes the module it builds to the storage transfers and the database events, as
// internal/app does for the module it builds, so a database whose config was saved can still be deleted.
func loadWithoutApp() backups.Module {
	backupsModule := backups.New(backups.Deps{
		DB:         storage.GetDb(),
		Transactor: storage.GetTransactor(),
		Logger:     logger.GetLogger(),
		Clock:      clock.NewSystem(),
		Workspaces: workspaces_bridge.GetModule(),
		Notifiers:  notifiers_bridge.GetModule(),
		Storages:   storages_bridge.GetModule(),
		Databases:  databases_bridge.GetModule(),
	})

	storages_bridge.GetModule().SubscribeToStorageTransferring(backupsModule.OnStorageTransferring)

	databasesModule := databases_bridge.GetModule()
	databasesModule.SubscribeToDatabaseCopied(backupsModule.OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseBackupTypeChanged(backupsModule.OnDatabaseBackupTypeChanged)
	databasesModule.SubscribeToDatabaseDeleting(backupsModule.OnDatabaseDeleting)

	return backupsModule
}
