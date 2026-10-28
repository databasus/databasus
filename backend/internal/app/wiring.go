package app

import (
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
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
	verification_runs.SetupDependencies()
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

func subscribeToDatabaseEvents(databasesModule databases.Module, backupsModule backups.Module) {
	databasesModule.SubscribeToDatabaseCreated(healthcheck_config.GetHealthcheckConfigService().OnDatabaseCreated)

	databasesModule.SubscribeToDatabaseCopied(backupsModule.OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseCopied(verification_config.GetVerificationConfigService().OnDatabaseCopied)

	databasesModule.SubscribeToDatabaseBackupTypeChanged(backupsModule.OnDatabaseBackupTypeChanged)
	databasesModule.SubscribeToDatabaseDeleting(backupsModule.OnDatabaseDeleting)
}

// subscribeToLogicalBackupEvents lets the old verification runs enqueue the after-backup verification
// of a completed logical backup until they move into a module.
func subscribeToLogicalBackupEvents(backupsModule backups.Module) {
	backupsModule.SubscribeToLogicalBackupCompleted(verification_runs.GetVerificationService().OnLogicalBackupCompleted)
}
