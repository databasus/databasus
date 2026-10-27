package backups_controllers_physical

import (
	"sync"

	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackupController = sync.OnceValue(func() *PhysicalBackupController {
	return &PhysicalBackupController{
		backups_services.GetPhysicalBackupService(),
		backups_download.GetRestoreTokenService(),
		logger.GetLogger(),
	}
})

func GetPhysicalBackupController() *PhysicalBackupController {
	return getPhysicalBackupController()
}
