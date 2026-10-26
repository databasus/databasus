package app

import (
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/restores"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/features/telemetry"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

// setUpDependencies registers the listeners between the old features. The server and the console
// commands share it, so a command writes the same audit records as the server.
func setUpDependencies() {
	backups_services.SetupDependencies()
	restores.SetupDependencies()
	backuping_physical.SetupDependencies()
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
func subscribeToStorageTransferring(storagesModule storages.Module) {
	storagesModule.SubscribeToStorageTransferring(backups_config_logical.GetBackupConfigService().OnStorageTransferring)
}

// subscribeToDatabaseEvents subscribes the old features to the databases module. A deletion runs its
// handlers in this order: the logical backups refuse while one runs, then the physical cancellation
// registers the stop of the running backup and of pg_receivewal, the physical backups request the
// deletion of their files, and the slot cleanup registers the drop of the slots. The after-commit
// hooks run in the same order, so the WAL streamer is stopped before its slot is dropped.
func subscribeToDatabaseEvents(databasesModule databases.Module) {
	databasesModule.SubscribeToDatabaseCreated(healthcheck_config.GetHealthcheckConfigService().OnDatabaseCreated)

	databasesModule.SubscribeToDatabaseCopied(backups_config_logical.GetBackupConfigService().OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseCopied(verification_config.GetVerificationConfigService().OnDatabaseCopied)

	databasesModule.SubscribeToDatabaseBackupTypeChanged(
		backups_config_physical.GetBackupConfigService().OnBackupTypeChanged,
	)

	databasesModule.SubscribeToDatabaseDeleting(backups_services.GetBackupService().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(
		backuping_physical.GetPhysicalBackupCancellationListener().OnDatabaseDeleting,
	)
	databasesModule.SubscribeToDatabaseDeleting(physical_service.GetPhysicalBackupService().OnDatabaseDeleting)
	databasesModule.SubscribeToDatabaseDeleting(backuping_physical.GetPhysicalSlotCleanupListener().OnDatabaseDeleting)
}
