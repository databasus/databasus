package backups_services

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	usecases_logical "databasus-backend/internal/features/backups/backups/usecases/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var taskCancellationRequester = task_cancellation.GetRequester()

var backupService = &LogicalBackupService{
	databases.GetDatabaseService(),
	storages.GetStorageService(),
	backups_core_logical.GetBackupRepository(),
	notifiers.GetNotifierService(),
	notifiers.GetNotifierService(),
	backups_config_logical.GetBackupConfigService(),
	encryption_secrets.GetSecretKeyService(),
	encryption.GetFieldEncryptor(),
	usecases_logical.GetCreateBackupUsecase(),
	logger.GetLogger(),
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	taskCancellationRequester,
	backups_download.GetDownloadTokenService(),
	backuping_logical.GetBackupsScheduler(),
	backuping_logical.GetBackupCleaner(),
}

func GetBackupService() *LogicalBackupService {
	return backupService
}

var physicalBackupService = &PhysicalBackupService{
	databases.GetDatabaseService(),
	workspaces_services.GetWorkspaceService(),
	backups_config_physical.GetBackupConfigService(),
	chain_view.GetChainViewService(),
	physical_core_service.GetPhysicalBackupService(),
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetIncrementalBackupRepository(),
	physical_repositories.GetWalSegmentRepository(),
	physical_repositories.GetInFlightBackupRepository(),
	backuping_physical.GetPhysicalBackupCanceller(),
	backuping_physical.GetPhysicalWalStreamSupervisor(),
	backups_download.GetRestoreStreamWriter(),
	backups_download.GetRestoreTokenService(),
	encryption_secrets.GetSecretKeyService(),
	audit_logs.GetAuditLogService(),
	logger.GetLogger(),
}

func GetPhysicalBackupService() *PhysicalBackupService {
	return physicalBackupService
}

var SetupDependencies = sync.OnceFunc(func() {
	backups_config_logical.
		GetBackupConfigService().
		SetDatabaseStorageChangeListener(backupService)
	backups_config_physical.
		GetBackupConfigService().
		SetDatabaseStorageChangeListener(physicalBackupService)

	databases.GetDatabaseService().AddDbRemoveListener(backupService)
	storages.GetStorageService().AddStorageBackupCounter(backupService)
	databases.GetDatabaseService().AddDbCopyListener(backups_config_logical.GetBackupConfigService())
})
