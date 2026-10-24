package storages

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	storage_files "databasus-backend/internal/features/storages/files"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var (
	getStorageRepository = sync.OnceValue(func() *StorageRepository {
		return &StorageRepository{}
	})
	getStorageService = sync.OnceValue(func() *StorageService {
		return &StorageService{
			getStorageRepository(),
			workspaces_services.GetWorkspaceService(),
			audit_logs.GetAuditLogService(),
			encryption.GetFieldEncryptor(),
			nil,
			nil,
		}
	})
)

var (
	getStorageFileDependencies = sync.OnceValue(func() storage_files.Dependencies {
		return storage_files.Dependencies{
			Repository:     &storage_files.PendingDeletionRepository{},
			Locator:        getStorageService(),
			FieldEncryptor: encryption.GetFieldEncryptor(),
			Logger:         logger.GetLogger(),
			Timings:        storage_files.ProductionTimings(),
		}
	})
	getStorageFileStore = sync.OnceValue(func() *storage_files.Store {
		return storage_files.NewStore(getStorageFileDependencies())
	})
	getStorageFileDeletionWorker = sync.OnceValue(func() *storage_files.DeletionWorker {
		return storage_files.NewDeletionWorker(getStorageFileStore(), getStorageFileDependencies())
	})
)

var getStorageController = sync.OnceValue(func() *StorageController {
	return &StorageController{
		getStorageService(),
		workspaces_services.GetWorkspaceService(),
	}
})

func GetStorageService() *StorageService {
	return getStorageService()
}

func GetStorageController() *StorageController {
	return getStorageController()
}

func GetStorageFileStore() *storage_files.Store {
	return getStorageFileStore()
}

func GetStorageFileDeletionWorker() *storage_files.DeletionWorker {
	return getStorageFileDeletionWorker()
}

var SetupDependencies = sync.OnceFunc(func() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(getStorageService())
})
