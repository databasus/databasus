package dashboard

import (
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
)

var dashboardService = &DashboardService{
	databases.GetDatabaseService(),
	backups_services.GetBackupService(),
	physical_core_service.GetPhysicalBackupService(),
	backups_config_logical.GetBackupConfigService(),
	backups_config_physical.GetBackupConfigService(),
	healthcheck_config.GetHealthcheckConfigService(),
	healthcheck_attempt.GetHealthcheckAttemptService(),
}

var dashboardController = &DashboardController{
	dashboardService,
}

func GetDashboardService() *DashboardService {
	return dashboardService
}

func GetDashboardController() *DashboardController {
	return dashboardController
}
