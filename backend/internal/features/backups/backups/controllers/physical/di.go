package backups_controllers_physical

import (
	"sync"

	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackupController = sync.OnceValue(func() *PhysicalBackupController {
	return &PhysicalBackupController{
		getBackups: backups_bridge.GetModule, logger: logger.GetLogger(),
	}
})

func GetPhysicalBackupController() *PhysicalBackupController {
	return getPhysicalBackupController()
}
