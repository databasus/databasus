package app

import (
	"databasus-backend/internal/features/audit_logs"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/restores"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/features/telemetry"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
)

// setUpDependencies registers the listeners between the old features. The server and the console
// commands share it, so a command writes the same audit records as the server.
func setUpDependencies() {
	databases.SetupDependencies()
	backups_services.SetupDependencies()
	restores.SetupDependencies()
	healthcheck_config.SetupDependencies()
	audit_logs.SetupDependencies()
	notifiers.SetupDependencies()
	storages.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()
	verification_config.SetupDependencies()
	verification_runs.SetupDependencies()
	task_cancellation.SetupDependencies()
	telemetry.SetupDependencies()
}
