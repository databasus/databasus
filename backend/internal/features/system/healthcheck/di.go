package system_healthcheck

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/disk"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/internal/util/logger"
)

var getHealthcheckService = sync.OnceValue(func() *HealthcheckService {
	return &HealthcheckService{
		disk.GetDiskService(),
		backups_bridge.GetModule,
		nil,
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

func InstallVerificationModule(module verification.Module) {
	getHealthcheckService().getVerification = func() verification.Module { return module }
}
