package healthcheck_attempt

import (
	"sync"

	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var (
	getHealthcheckAttemptRepository = sync.OnceValue(func() *HealthcheckAttemptRepository {
		return &HealthcheckAttemptRepository{}
	})
	getHealthcheckAttemptService = sync.OnceValue(func() *HealthcheckAttemptService {
		return &HealthcheckAttemptService{
			getHealthcheckAttemptRepository(),
			databases.GetDatabaseService(),
			workspaces_services.GetWorkspaceService(),
		}
	})
)

var getCheckDatabaseHealthUseCase = sync.OnceValue(func() *CheckDatabaseHealthUseCase {
	return &CheckDatabaseHealthUseCase{
		getHealthcheckAttemptRepository(),
		notifiers.GetNotifierService(),
		databases.GetDatabaseService(),
	}
})

var getHealthcheckAttemptBackgroundService = sync.OnceValue(func() *HealthcheckAttemptBackgroundService {
	return &HealthcheckAttemptBackgroundService{
		healthcheckConfigService:   healthcheck_config.GetHealthcheckConfigService(),
		checkDatabaseHealthUseCase: getCheckDatabaseHealthUseCase(),
		logger:                     logger.GetLogger(),
	}
})

var getHealthcheckAttemptController = sync.OnceValue(func() *HealthcheckAttemptController {
	return &HealthcheckAttemptController{
		getHealthcheckAttemptService(),
	}
})

func GetHealthcheckAttemptRepository() *HealthcheckAttemptRepository {
	return getHealthcheckAttemptRepository()
}

func GetHealthcheckAttemptService() *HealthcheckAttemptService {
	return getHealthcheckAttemptService()
}

func GetHealthcheckAttemptBackgroundService() *HealthcheckAttemptBackgroundService {
	return getHealthcheckAttemptBackgroundService()
}

func GetHealthcheckAttemptController() *HealthcheckAttemptController {
	return getHealthcheckAttemptController()
}
