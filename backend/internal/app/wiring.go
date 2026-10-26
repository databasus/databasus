package app

import (
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/restores"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/features/telemetry"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/workspaces"
)

// setUpDependencies registers the listeners between the old features. The server and the console
// commands share it, so a command writes the same audit records as the server.
func setUpDependencies() {
	backups_services.SetupDependencies()
	restores.SetupDependencies()
	healthcheck_config.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()
	verification_config.SetupDependencies()
	verification_runs.SetupDependencies()
	task_cancellation.SetupDependencies()
	telemetry.SetupDependencies()
}

// subscribeToWorkspaceDeleting sets the order in which a workspace deletion runs its handlers:
// the databases refuse a workspace that still has some before the notifiers and the storages are
// deleted.
func subscribeToWorkspaceDeleting(workspacesModule workspaces.Module, notifiersModule notifiers.Module) {
	workspacesModule.SubscribeToWorkspaceDeleting(databases.GetDatabaseService().OnWorkspaceDeleting)
	workspacesModule.SubscribeToWorkspaceDeleting(notifiersModule.OnWorkspaceDeleting)
	workspacesModule.SubscribeToWorkspaceDeleting(storages.GetStorageService().OnWorkspaceDeleting)
}

// subscribeToNotifierTransferring lets the databases refuse to move a notifier that a database
// outside the target workspace uses.
func subscribeToNotifierTransferring(notifiersModule notifiers.Module) {
	notifiersModule.SubscribeToNotifierTransferring(databases.GetDatabaseService().OnNotifierTransferring)
}
