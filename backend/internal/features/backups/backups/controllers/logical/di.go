package backups_controllers_logical

import (
	"sync"

	backups_services "databasus-backend/internal/features/backups/backups/services"
)

var getBackupController = sync.OnceValue(func() *BackupController {
	return &BackupController{
		backups_services.GetBackupService(),
	}
})

func GetBackupController() *BackupController {
	return getBackupController()
}
