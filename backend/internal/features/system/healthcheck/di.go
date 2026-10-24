package system_healthcheck

import (
	"sync"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	"databasus-backend/internal/features/disk"
	verification_agents "databasus-backend/internal/features/verification/agents"
	"databasus-backend/internal/util/logger"
)

var getHealthcheckService = sync.OnceValue(func() *HealthcheckService {
	return &HealthcheckService{
		disk.GetDiskService(),
		backuping_logical.GetBackupsScheduler(),
		verification_agents.GetAgentService(),
		logger.GetLogger(),
	}
})

var getHealthcheckController = sync.OnceValue(func() *HealthcheckController {
	return &HealthcheckController{
		getHealthcheckService(),
	}
})

func GetHealthcheckController() *HealthcheckController {
	return getHealthcheckController()
}
