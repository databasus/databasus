package app

import (
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/restores"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/features/telemetry"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

// setUpDependencies registers the listeners between the old features. The server and the console
// commands share it, so a command writes the same audit records as the server.
func setUpDependencies() {
	restores.SetupDependencies()
	verification_runs.SetupDependencies()
	task_cancellation.SetupDependencies()
	telemetry.SetupDependencies()
}

// subscribeToWorkspaceDeleting sets the order in which a workspace deletion runs its handlers:
// the databases refuse a workspace that still has some before the notifiers and the storages are
// deleted.
func subscribeToWorkspaceDeleting(
	workspacesModule workspaces.Module,
	databasesModule databases.Module,
	notifiersModule notifiers.Module,
	storagesModule storages.Module,
) {
	workspacesModule.SubscribeToWorkspaceDeleting(databasesModule.OnWorkspaceDeleting)
	workspacesModule.SubscribeToWorkspaceDeleting(notifiersModule.OnWorkspaceDeleting)
	workspacesModule.SubscribeToWorkspaceDeleting(storagesModule.OnWorkspaceDeleting)
}

// subscribeToNotifierTransferring lets the databases refuse to move a notifier that a database
// outside the target workspace uses.
func subscribeToNotifierTransferring(notifiersModule notifiers.Module, databasesModule databases.Module) {
	notifiersModule.SubscribeToNotifierTransferring(databasesModule.OnNotifierTransferring)
}

// subscribeToStorageTransferring lets the backup configs refuse to move a storage that a database
// outside the target workspace uses.
func subscribeToStorageTransferring(storagesModule storages.Module, backupsModule backups.Module) {
	storagesModule.SubscribeToStorageTransferring(backupsModule.OnStorageTransferring)
}

// subscribeToDatabaseEvents subscribes the backups module and the old features to the databases
// module. A deletion runs its handlers in this order: the logical backups refuse while one runs, then
// the physical cancellation registers the stop of the running backup and of pg_receivewal, the
// physical backups request the deletion of their files, the slot cleanup registers the drop of the
// slots, and the backup config goes last, so the handlers before it still read it. The after-commit
// hooks run in the same order, so the WAL streamer is stopped before its slot is dropped.
func subscribeToDatabaseEvents(databasesModule databases.Module, backupsModule backups.Module) {
	databasesModule.SubscribeToDatabaseCreated(healthcheck_config.GetHealthcheckConfigService().OnDatabaseCreated)

	databasesModule.SubscribeToDatabaseCopied(backupsModule.OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseCopied(verification_config.GetVerificationConfigService().OnDatabaseCopied)

	databasesModule.SubscribeToDatabaseBackupTypeChanged(backupsModule.OnDatabaseBackupTypeChanged)
	databasesModule.SubscribeToDatabaseBackupTypeChanged(
		backuping_physical.GetPhysicalBackupCancellationListener().OnDatabaseBackupTypeChanged,
	)

	databasesModule.SubscribeToDatabaseDeleting(backups_services.GetBackupService().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(
		backuping_physical.GetPhysicalBackupCancellationListener().OnDatabaseDeleting,
	)
	databasesModule.SubscribeToDatabaseDeleting(physical_service.GetPhysicalBackupService().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(backuping_physical.GetPhysicalSlotCleanupListener().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(backupsModule.OnDatabaseDeleting)
}

// subscribeToBackupConfigEvents lets the old backups react to a saved config until they move into the
// backups module. A logical config with another storage removes the logical backups in the save's
// transaction. A saved physical config first cancels what it disabled or moved, then removes the
// backups kept in another storage.
func subscribeToBackupConfigEvents(backupsModule backups.Module) {
	backupsModule.SubscribeToLogicalBackupConfigStorageChanging(
		backups_services.GetBackupService().OnLogicalBackupConfigStorageChanging,
	)

	backupsModule.SubscribeToPhysicalBackupConfigSaved(
		backuping_physical.GetPhysicalBackupCancellationListener().OnPhysicalBackupConfigSaved,
	)
	backupsModule.SubscribeToPhysicalBackupConfigSaved(
		backups_services.GetPhysicalBackupService().OnPhysicalBackupConfigSaved,
	)
}
