package app

import (
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/healthcheck"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/internal/modules/workspaces"
)

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

type databaseSubscriptions struct {
	Databases    databases.Module
	Backups      backups.Module
	Verification verification.Module
	Healthcheck  healthcheck.Module
}

func subscribeToDatabaseEvents(subscriptions databaseSubscriptions) {
	subscriptions.Databases.SubscribeToDatabaseCopied(subscriptions.Backups.OnDatabaseCopied)
	subscriptions.Databases.SubscribeToDatabaseCopied(subscriptions.Verification.OnDatabaseCopied)

	subscriptions.Databases.SubscribeToDatabaseBackupTypeChanged(subscriptions.Backups.OnDatabaseBackupTypeChanged)
	subscriptions.Databases.SubscribeToDatabaseDeleting(subscriptions.Backups.OnDatabaseDeleting)
	subscriptions.Databases.SubscribeToDatabaseDeleting(subscriptions.Verification.OnDatabaseDeleting)
	subscriptions.Databases.SubscribeToDatabaseCreated(subscriptions.Healthcheck.OnDatabaseCreated)
	subscriptions.Databases.SubscribeToDatabaseCopied(subscriptions.Healthcheck.OnDatabaseCopied)
	subscriptions.Databases.SubscribeToDatabaseDeleting(subscriptions.Healthcheck.OnDatabaseDeleting)
}

func subscribeToLogicalBackupEvents(backupsModule backups.Module, verificationModule verification.Module) {
	backupsModule.SubscribeToLogicalBackupCompleted(verificationModule.OnLogicalBackupCompleted)
	backupsModule.SubscribeToLogicalBackupsDeleting(verificationModule.OnLogicalBackupsDeleting)
}
