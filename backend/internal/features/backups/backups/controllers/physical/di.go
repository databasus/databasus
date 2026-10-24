package backups_controllers_physical

import (
	"sync"

	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_services "databasus-backend/internal/features/backups/backups/services"
)

var getPhysicalBackupController = sync.OnceValue(func() *PhysicalBackupController {
	return &PhysicalBackupController{
		backups_services.GetPhysicalBackupService(),
		backups_download.GetRestoreTokenService(),
	}
})

func GetPhysicalBackupController() *PhysicalBackupController {
	return getPhysicalBackupController()
}
