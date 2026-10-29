package system_healthcheck

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/disk"
	verification_bridge "databasus-backend/internal/features/verification"
	"databasus-backend/internal/util/logger"
)

var getHealthcheckService = sync.OnceValue(func() *HealthcheckService {
	return &HealthcheckService{
		disk.GetDiskService(),
		backups_bridge.GetModule,
		verification_bridge.GetModule,
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
