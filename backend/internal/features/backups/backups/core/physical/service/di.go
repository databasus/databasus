package physical_service

import (
	"sync"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackupService = sync.OnceValue(func() *PhysicalBackupService {
	return &PhysicalBackupService{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		physical_repositories.GetWalStreamerRepository(),
		storages.GetStorageService(),
		storages.GetStorageFileStore(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
	}
})

func GetPhysicalBackupService() *PhysicalBackupService {
	return getPhysicalBackupService()
}
